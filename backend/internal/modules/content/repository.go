package content

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	syncmodule "github.com/W0LIF/LoveTea/backend/internal/modules/sync"
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

func (r *Repository) ListMedia(ctx context.Context, userID uuid.UUID) ([]MediaFile, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, CASE WHEN couple_id IS NULL THEN 'personal' ELSE 'couple' END,
		       content_type, size_bytes, status, version, created_at
		FROM media_files
		WHERE status = 'ready' AND deleted_at IS NULL
		  AND (
		    (owner_user_id = $1 AND couple_id IS NULL)
		    OR couple_id = (
		      SELECT couple_id FROM couple_members
		      WHERE user_id = $1 AND left_at IS NULL LIMIT 1
		    )
		  )
		ORDER BY created_at DESC, id
		LIMIT 200
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("list media: %w", err)
	}
	defer rows.Close()

	media := make([]MediaFile, 0)
	for rows.Next() {
		var id uuid.UUID
		var item MediaFile
		if err := rows.Scan(
			&id, &item.Scope, &item.ContentType, &item.SizeBytes,
			&item.Status, &item.Version, &item.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan media: %w", err)
		}
		item.ID = id.String()
		media = append(media, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate media: %w", err)
	}
	return media, nil
}

func (r *Repository) CreatePendingMedia(
	ctx context.Context,
	userID uuid.UUID,
	mediaID uuid.UUID,
	objectKey string,
	contentType string,
	scope string,
) (MediaFile, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return MediaFile{}, fmt.Errorf("begin media creation: %w", err)
	}
	defer tx.Rollback(ctx)

	coupleID, err := resolveScope(ctx, tx, userID, scope)
	if err != nil {
		return MediaFile{}, err
	}
	var item MediaFile
	err = tx.QueryRow(ctx, `
		INSERT INTO media_files (id, owner_user_id, couple_id, object_key, content_type)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, content_type, status, version, created_at
	`, mediaID, userID, coupleID, objectKey, contentType).Scan(
		&item.ID, &item.ContentType, &item.Status, &item.Version, &item.CreatedAt,
	)
	if err != nil {
		return MediaFile{}, fmt.Errorf("insert pending media: %w", err)
	}
	item.Scope = scope
	if err := tx.Commit(ctx); err != nil {
		return MediaFile{}, fmt.Errorf("commit pending media: %w", err)
	}
	return item, nil
}

func (r *Repository) GetMedia(
	ctx context.Context,
	userID uuid.UUID,
	mediaID uuid.UUID,
) (MediaFile, string, error) {
	var (
		item MediaFile
		id   uuid.UUID
		key  string
	)
	err := r.db.QueryRow(ctx, `
		SELECT m.id, CASE WHEN m.couple_id IS NULL THEN 'personal' ELSE 'couple' END,
		       m.content_type, COALESCE(m.size_bytes, 0), m.status, m.version,
		       m.created_at, m.object_key
		FROM media_files m
		WHERE m.id = $1 AND m.deleted_at IS NULL
		  AND (
		    (m.owner_user_id = $2 AND m.couple_id IS NULL)
		    OR m.couple_id = (
		      SELECT couple_id FROM couple_members
		      WHERE user_id = $2 AND left_at IS NULL LIMIT 1
		    )
		  )
	`, mediaID, userID).Scan(
		&id, &item.Scope, &item.ContentType, &item.SizeBytes, &item.Status,
		&item.Version, &item.CreatedAt, &key,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return MediaFile{}, "", ErrNotFound
	}
	if err != nil {
		return MediaFile{}, "", fmt.Errorf("get media: %w", err)
	}
	item.ID = id.String()
	return item, key, nil
}

func (r *Repository) CompleteMedia(
	ctx context.Context,
	userID uuid.UUID,
	mediaID uuid.UUID,
	size int64,
) (MediaFile, string, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return MediaFile{}, "", fmt.Errorf("begin media completion: %w", err)
	}
	defer tx.Rollback(ctx)

	var (
		item     MediaFile
		id       uuid.UUID
		key      string
		coupleID *uuid.UUID
	)
	err = tx.QueryRow(ctx, `
		SELECT m.id, CASE WHEN m.couple_id IS NULL THEN 'personal' ELSE 'couple' END,
		       m.content_type, COALESCE(m.size_bytes, 0), m.status, m.version,
		       m.created_at, m.object_key, m.couple_id
		FROM media_files m
		WHERE m.id = $1 AND m.deleted_at IS NULL
		  AND (
		    (m.owner_user_id = $2 AND m.couple_id IS NULL)
		    OR m.couple_id = (
		      SELECT couple_id FROM couple_members
		      WHERE user_id = $2 AND left_at IS NULL LIMIT 1
		    )
		  )
		FOR UPDATE
	`, mediaID, userID).Scan(
		&id, &item.Scope, &item.ContentType, &item.SizeBytes, &item.Status,
		&item.Version, &item.CreatedAt, &key, &coupleID,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return MediaFile{}, "", ErrNotFound
	}
	if err != nil {
		return MediaFile{}, "", fmt.Errorf("lock media for completion: %w", err)
	}
	item.ID = id.String()
	if item.Status == "ready" {
		if err := tx.Commit(ctx); err != nil {
			return MediaFile{}, "", fmt.Errorf("commit idempotent media completion: %w", err)
		}
		return item, key, nil
	}
	if item.Status != "pending" || size < 1 {
		return MediaFile{}, "", ErrInvalidContent
	}
	err = tx.QueryRow(ctx, `
		UPDATE media_files
		SET status = 'ready', size_bytes = $2,
		    version = version + 1, updated_at = NOW()
		WHERE id = $1
		RETURNING size_bytes, version
	`, mediaID, size).Scan(&item.SizeBytes, &item.Version)
	if err != nil {
		return MediaFile{}, "", fmt.Errorf("mark media ready: %w", err)
	}
	payload, err := json.Marshal(mediaPayload{
		ContentType: item.ContentType,
		SizeBytes:   item.SizeBytes,
	})
	if err != nil {
		return MediaFile{}, "", fmt.Errorf("encode media sync payload: %w", err)
	}
	if err := recordChange(ctx, tx, userID, coupleID, "media_file", mediaID, "upsert", payload); err != nil {
		return MediaFile{}, "", err
	}
	if err := tx.Commit(ctx); err != nil {
		return MediaFile{}, "", fmt.Errorf("commit media completion: %w", err)
	}
	return item, key, nil
}

func (r *Repository) DeleteMedia(
	ctx context.Context,
	userID uuid.UUID,
	mediaID uuid.UUID,
	version int64,
) (string, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return "", fmt.Errorf("begin media deletion: %w", err)
	}
	defer tx.Rollback(ctx)

	var (
		key      string
		coupleID *uuid.UUID
	)
	err = tx.QueryRow(ctx, `
		UPDATE media_files m
		SET status = 'deleted', deleted_at = NOW(),
		    version = m.version + 1, updated_at = NOW()
		WHERE m.id = $1 AND m.version = $3 AND m.deleted_at IS NULL
		  AND (
		    (m.owner_user_id = $2 AND m.couple_id IS NULL)
		    OR m.couple_id = (
		      SELECT couple_id FROM couple_members
		      WHERE user_id = $2 AND left_at IS NULL LIMIT 1
		    )
		  )
		RETURNING m.object_key, m.couple_id
	`, mediaID, userID, version).Scan(&key, &coupleID)
	if errors.Is(err, pgx.ErrNoRows) {
		var exists bool
		if checkErr := tx.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM media_files m
				WHERE m.id = $1 AND m.deleted_at IS NULL
				  AND (
				    (m.owner_user_id = $2 AND m.couple_id IS NULL)
				    OR m.couple_id = (
				      SELECT couple_id FROM couple_members
				      WHERE user_id = $2 AND left_at IS NULL LIMIT 1
				    )
				  )
			)
		`, mediaID, userID).Scan(&exists); checkErr != nil {
			return "", fmt.Errorf("check media after failed delete: %w", checkErr)
		}
		if exists {
			return "", ErrVersionConflict
		}
		return "", ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("mark media deleted: %w", err)
	}
	if err := recordChange(ctx, tx, userID, coupleID, "media_file", mediaID, "delete", json.RawMessage(`{}`)); err != nil {
		return "", err
	}
	if err := tx.Commit(ctx); err != nil {
		return "", fmt.Errorf("commit media deletion: %w", err)
	}
	return key, nil
}

func (r *Repository) ListMoments(ctx context.Context, userID uuid.UUID) ([]Moment, error) {
	rows, err := r.db.Query(ctx, `
		SELECT m.id, CASE WHEN m.couple_id IS NULL THEN 'personal' ELSE 'couple' END,
		       m.title, m.caption, m.happened_at, m.version, m.created_at, m.updated_at,
		       COALESCE(
		         jsonb_agg(jsonb_build_object(
		           'id', f.id, 'content_type', f.content_type, 'size_bytes', f.size_bytes
		         ) ORDER BY mm.position) FILTER (WHERE f.id IS NOT NULL),
		         '[]'::JSONB
		       )
		FROM moments m
		LEFT JOIN moment_media mm ON mm.moment_id = m.id
		LEFT JOIN media_files f ON f.id = mm.media_id
		  AND f.status = 'ready' AND f.deleted_at IS NULL
		WHERE m.deleted_at IS NULL
		  AND (
		    (m.owner_user_id = $1 AND m.couple_id IS NULL)
		    OR m.couple_id = (
		      SELECT couple_id FROM couple_members
		      WHERE user_id = $1 AND left_at IS NULL LIMIT 1
		    )
		  )
		GROUP BY m.id
		ORDER BY m.happened_at DESC, m.id
		LIMIT 200
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("list moments: %w", err)
	}
	defer rows.Close()

	moments := make([]Moment, 0)
	for rows.Next() {
		var (
			moment Moment
			id     uuid.UUID
			media  []byte
		)
		if err := rows.Scan(
			&id, &moment.Scope, &moment.Title, &moment.Caption, &moment.HappenedAt,
			&moment.Version, &moment.CreatedAt, &moment.UpdatedAt, &media,
		); err != nil {
			return nil, fmt.Errorf("scan moment: %w", err)
		}
		if err := json.Unmarshal(media, &moment.Media); err != nil {
			return nil, fmt.Errorf("decode moment media: %w", err)
		}
		moment.ID = id.String()
		moments = append(moments, moment)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate moments: %w", err)
	}
	return moments, nil
}

func (r *Repository) CreateMoment(
	ctx context.Context,
	userID uuid.UUID,
	request CreateMomentRequest,
	mediaIDs []uuid.UUID,
) (Moment, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return Moment{}, fmt.Errorf("begin moment creation: %w", err)
	}
	defer tx.Rollback(ctx)

	coupleID, err := resolveScope(ctx, tx, userID, request.Scope)
	if err != nil {
		return Moment{}, err
	}
	if len(mediaIDs) > 0 {
		var count int
		err := tx.QueryRow(ctx, `
			SELECT COUNT(*)
			FROM media_files
			WHERE id = ANY($1::UUID[])
			  AND status = 'ready' AND deleted_at IS NULL
			  AND couple_id IS NOT DISTINCT FROM $2
			  AND ($2::UUID IS NOT NULL OR owner_user_id = $3)
		`, mediaIDs, coupleID, userID).Scan(&count)
		if err != nil {
			return Moment{}, fmt.Errorf("validate moment media: %w", err)
		}
		if count != len(mediaIDs) {
			return Moment{}, ErrNotFound
		}
	}

	var moment Moment
	var id uuid.UUID
	err = tx.QueryRow(ctx, `
		INSERT INTO moments (owner_user_id, couple_id, title, caption, happened_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, title, caption, happened_at, version, created_at, updated_at
	`, userID, coupleID, request.Title, request.Caption, request.HappenedAt).Scan(
		&id, &moment.Title, &moment.Caption, &moment.HappenedAt,
		&moment.Version, &moment.CreatedAt, &moment.UpdatedAt,
	)
	if err != nil {
		return Moment{}, fmt.Errorf("insert moment: %w", err)
	}
	moment.ID = id.String()
	moment.Scope = request.Scope
	moment.Media = make([]MediaAttachment, 0, len(mediaIDs))
	for position, mediaID := range mediaIDs {
		if _, err := tx.Exec(ctx, `
			INSERT INTO moment_media (moment_id, media_id, position)
			VALUES ($1, $2, $3)
		`, id, mediaID, position); err != nil {
			return Moment{}, fmt.Errorf("attach media to moment: %w", err)
		}
		var attachment MediaAttachment
		if err := tx.QueryRow(ctx, `
			SELECT id, content_type, size_bytes
			FROM media_files WHERE id = $1
		`, mediaID).Scan(&attachment.ID, &attachment.ContentType, &attachment.SizeBytes); err != nil {
			return Moment{}, fmt.Errorf("read attached media: %w", err)
		}
		moment.Media = append(moment.Media, attachment)
	}
	payload, err := json.Marshal(momentPayload{
		Title:      moment.Title,
		Caption:    moment.Caption,
		HappenedAt: moment.HappenedAt,
		Media:      moment.Media,
	})
	if err != nil {
		return Moment{}, fmt.Errorf("encode moment sync payload: %w", err)
	}
	if err := recordChange(ctx, tx, userID, coupleID, "moment", id, "upsert", payload); err != nil {
		return Moment{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Moment{}, fmt.Errorf("commit moment: %w", err)
	}
	return moment, nil
}

func resolveScope(
	ctx context.Context,
	tx pgx.Tx,
	userID uuid.UUID,
	scope string,
) (*uuid.UUID, error) {
	if scope == "personal" {
		return nil, nil
	}
	var coupleID uuid.UUID
	err := tx.QueryRow(ctx, `
		SELECT couple_id FROM couple_members
		WHERE user_id = $1 AND left_at IS NULL
		LIMIT 1
	`, userID).Scan(&coupleID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNoActiveCouple
	}
	if err != nil {
		return nil, fmt.Errorf("resolve content scope: %w", err)
	}
	return &coupleID, nil
}

func recordChange(
	ctx context.Context,
	tx pgx.Tx,
	actorID uuid.UUID,
	coupleID *uuid.UUID,
	entityType string,
	entityID uuid.UUID,
	operation string,
	payload json.RawMessage,
) error {
	scope := syncmodule.Scope{UserID: &actorID}
	if coupleID != nil {
		scope = syncmodule.Scope{CoupleID: coupleID}
	}
	if _, err := syncmodule.RecordChange(ctx, tx, syncmodule.NewChange{
		Scope:      scope,
		ActorID:    actorID,
		EntityType: entityType,
		EntityID:   entityID,
		Operation:  operation,
		Payload:    payload,
	}); err != nil {
		return fmt.Errorf("record %s sync change: %w", entityType, err)
	}
	return nil
}

type mediaPayload struct {
	ContentType string `json:"content_type"`
	SizeBytes   int64  `json:"size_bytes"`
}

type momentPayload struct {
	Title      string            `json:"title"`
	Caption    string            `json:"caption"`
	HappenedAt time.Time         `json:"happened_at"`
	Media      []MediaAttachment `json:"media"`
}
