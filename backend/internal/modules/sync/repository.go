package sync

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func RecordChange(
	ctx context.Context,
	tx pgx.Tx,
	change NewChange,
) (int64, error) {
	if err := validateChange(change); err != nil {
		return 0, err
	}
	if len(change.Payload) == 0 {
		change.Payload = json.RawMessage(`{}`)
	}

	recipients := make([]uuid.UUID, 0, 2)
	if change.Scope.UserID != nil {
		recipients = append(recipients, *change.Scope.UserID)
	} else {
		rows, err := tx.Query(ctx, `
			SELECT user_id
			FROM couple_members
			WHERE couple_id = $1 AND left_at IS NULL
			ORDER BY user_id
		`, *change.Scope.CoupleID)
		if err != nil {
			return 0, fmt.Errorf("list sync change recipients: %w", err)
		}
		for rows.Next() {
			var recipient uuid.UUID
			if err := rows.Scan(&recipient); err != nil {
				rows.Close()
				return 0, fmt.Errorf("scan sync change recipient: %w", err)
			}
			recipients = append(recipients, recipient)
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return 0, fmt.Errorf("iterate sync change recipients: %w", err)
		}
		rows.Close()

		isMember := false
		for _, recipient := range recipients {
			isMember = isMember || recipient == change.ActorID
		}
		if !isMember || len(recipients) == 0 {
			return 0, ErrInvalidChange
		}
	}

	var (
		version int64
		err     error
	)
	if change.Scope.UserID != nil {
		err = tx.QueryRow(ctx, `
			INSERT INTO sync_entity_versions (
				owner_user_id, entity_type, entity_id, version
			)
			VALUES ($1, $2, $3, 1)
			ON CONFLICT (owner_user_id, entity_type, entity_id)
				WHERE owner_user_id IS NOT NULL
			DO UPDATE SET
				version = sync_entity_versions.version + 1,
				updated_at = NOW()
			RETURNING version
		`, *change.Scope.UserID, change.EntityType, change.EntityID).
			Scan(&version)
	} else {
		err = tx.QueryRow(ctx, `
			INSERT INTO sync_entity_versions (
				couple_id, entity_type, entity_id, version
			)
			VALUES ($1, $2, $3, 1)
			ON CONFLICT (couple_id, entity_type, entity_id)
				WHERE couple_id IS NOT NULL
			DO UPDATE SET
				version = sync_entity_versions.version + 1,
				updated_at = NOW()
			RETURNING version
		`, *change.Scope.CoupleID, change.EntityType, change.EntityID).
			Scan(&version)
	}
	if err != nil {
		return 0, fmt.Errorf("advance sync entity version: %w", err)
	}

	var actorCursor int64
	for _, recipient := range recipients {
		var userCursor int64
		err = tx.QueryRow(ctx, `
			INSERT INTO sync_cursors (user_id, cursor)
			VALUES ($1, 1)
			ON CONFLICT (user_id)
			DO UPDATE SET
				cursor = sync_cursors.cursor + 1,
				updated_at = NOW()
			RETURNING cursor
		`, recipient).Scan(&userCursor)
		if err != nil {
			return 0, fmt.Errorf("advance user sync cursor: %w", err)
		}

		if change.Scope.CoupleID == nil {
			_, err = tx.Exec(ctx, `
				INSERT INTO sync_changes (
					recipient_user_id, user_cursor, actor_user_id,
					entity_type, entity_id, version, operation, payload
				)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8::JSONB)
			`,
				recipient,
				userCursor,
				change.ActorID,
				change.EntityType,
				change.EntityID,
				version,
				change.Operation,
				[]byte(change.Payload),
			)
		} else {
			_, err = tx.Exec(ctx, `
			INSERT INTO sync_changes (
				recipient_user_id, user_cursor, couple_id, actor_user_id,
				entity_type, entity_id, version, operation, payload
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9::JSONB)
		`,
				recipient,
				userCursor,
				*change.Scope.CoupleID,
				change.ActorID,
				change.EntityType,
				change.EntityID,
				version,
				change.Operation,
				[]byte(change.Payload),
			)
		}
		if err != nil {
			return 0, fmt.Errorf("record sync change: %w", err)
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO realtime_outbox (recipient_user_id, sync_cursor)
			VALUES ($1, $2)
		`, recipient, userCursor); err != nil {
			return 0, fmt.Errorf("record realtime outbox event: %w", err)
		}
		if recipient == change.ActorID {
			actorCursor = userCursor
		}
	}

	return actorCursor, nil
}

func (r *Repository) Pull(
	ctx context.Context,
	userID uuid.UUID,
	cursor int64,
	limit int,
) (Page, error) {
	if err := validatePage(cursor, limit); err != nil {
		return Page{}, err
	}

	var watermark int64
	if err := r.db.QueryRow(ctx, `
		SELECT COALESCE(
			(SELECT cursor FROM sync_cursors WHERE user_id = $1),
			0
		)
	`, userID).Scan(&watermark); err != nil {
		return Page{}, fmt.Errorf("get sync watermark: %w", err)
	}

	rows, err := r.db.Query(ctx, `
		SELECT user_cursor, entity_type, entity_id, version,
		       operation, payload, created_at
		FROM sync_changes
		WHERE recipient_user_id = $1
		  AND user_cursor > $2 AND user_cursor <= $3
		  AND (
		    couple_id IS NULL
		    OR entity_type = 'couple_access_revoked'
		    OR EXISTS (
		      SELECT 1
		      FROM couple_members
		      WHERE user_id = $1
		        AND couple_id = sync_changes.couple_id
		        AND left_at IS NULL
		    )
		  )
		ORDER BY user_cursor
		LIMIT $4
	`, userID, cursor, watermark, limit+1)
	if err != nil {
		return Page{}, fmt.Errorf("pull sync changes: %w", err)
	}
	defer rows.Close()

	changes := make([]Change, 0, limit)
	for rows.Next() {
		var (
			change  Change
			payload []byte
		)
		if err := rows.Scan(
			&change.Cursor,
			&change.EntityType,
			&change.EntityID,
			&change.Version,
			&change.Operation,
			&payload,
			&change.CreatedAt,
		); err != nil {
			return Page{}, fmt.Errorf("scan sync change: %w", err)
		}
		change.Payload = append(json.RawMessage(nil), payload...)
		changes = append(changes, change)
	}
	if err := rows.Err(); err != nil {
		return Page{}, fmt.Errorf("iterate sync changes: %w", err)
	}

	hasMore := len(changes) > limit
	if hasMore {
		changes = changes[:limit]
	}

	nextCursor := watermark
	if cursor > watermark {
		nextCursor = cursor
	}
	if hasMore {
		nextCursor = changes[len(changes)-1].Cursor
	}

	return Page{
		Changes:    changes,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	}, nil
}
