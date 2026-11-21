package wellness

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	syncmodule "github.com/W0LIF/LoveTea/backend/internal/modules/sync"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) ListMood(
	ctx context.Context,
	userID uuid.UUID,
	rangeDates dateRange,
) ([]MoodEntry, error) {
	rows, err := r.db.Query(ctx, `
		SELECT e.id, e.entry_date, e.mood,
		       CASE WHEN e.owner_user_id = $1 THEN e.note ELSE '' END,
		       COALESCE(e.shared_couple_id = (
		         SELECT cm.couple_id
		         FROM couple_members cm
		         JOIN couples c ON c.id = cm.couple_id
		         WHERE cm.user_id = $1 AND cm.left_at IS NULL
		           AND c.status = 'active'
		         LIMIT 1
		       ), FALSE),
		       e.owner_user_id = $1, e.version, e.created_at, e.updated_at
		FROM mood_entries e
		WHERE e.deleted_at IS NULL
		  AND e.entry_date BETWEEN $2::DATE AND $3::DATE
		  AND (
		    e.owner_user_id = $1
		    OR e.shared_couple_id = (
		      SELECT cm.couple_id
		      FROM couple_members cm
		      JOIN couples c ON c.id = cm.couple_id
		      WHERE cm.user_id = $1 AND cm.left_at IS NULL
		        AND c.status = 'active'
		      LIMIT 1
		    )
		  )
		ORDER BY e.entry_date DESC, e.id
		LIMIT 732
	`, userID, rangeDates.From, rangeDates.To)
	if err != nil {
		return nil, fmt.Errorf("list mood entries: %w", err)
	}
	defer rows.Close()

	entries := make([]MoodEntry, 0)
	for rows.Next() {
		var entry MoodEntry
		var id uuid.UUID
		var entryDate time.Time
		if err := rows.Scan(
			&id, &entryDate, &entry.Mood, &entry.Note,
			&entry.SharedWithCouple, &entry.IsMine, &entry.Version,
			&entry.CreatedAt, &entry.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan mood entry: %w", err)
		}
		entry.ID = id.String()
		entry.EntryDate = entryDate.Format("2006-01-02")
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate mood entries: %w", err)
	}
	return entries, nil
}

func (r *Repository) SaveMood(
	ctx context.Context,
	userID uuid.UUID,
	request SaveMoodRequest,
) (MoodEntry, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return MoodEntry{}, fmt.Errorf("begin mood entry save: %w", err)
	}
	defer tx.Rollback(ctx)

	var newCoupleID *uuid.UUID
	if request.SharedWithCouple {
		newCoupleID, err = activeCoupleID(ctx, tx, userID)
		if err != nil {
			return MoodEntry{}, err
		}
	}

	var (
		id          uuid.UUID
		oldCoupleID *uuid.UUID
		version     int64
		createdAt   time.Time
		updatedAt   time.Time
	)
	err = tx.QueryRow(ctx, `
		SELECT id, shared_couple_id, version, created_at, updated_at
		FROM mood_entries
		WHERE owner_user_id = $1
		  AND entry_date = $2::DATE
		  AND deleted_at IS NULL
		FOR UPDATE
	`, userID, request.EntryDate).Scan(
		&id, &oldCoupleID, &version, &createdAt, &updatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		if request.Version != nil {
			return MoodEntry{}, ErrVersionConflict
		}
		err = tx.QueryRow(ctx, `
			INSERT INTO mood_entries (
				owner_user_id, entry_date, mood, note, shared_couple_id
			)
			VALUES ($1, $2::DATE, $3, $4, $5)
			RETURNING id, version, created_at, updated_at
		`, userID, request.EntryDate, request.Mood, request.Note, newCoupleID).
			Scan(&id, &version, &createdAt, &updatedAt)
		if err != nil {
			if isUniqueViolation(err, "mood_entries_owner_date_unique_idx") {
				return MoodEntry{}, ErrVersionConflict
			}
			return MoodEntry{}, fmt.Errorf("insert mood entry: %w", err)
		}
	} else if err != nil {
		return MoodEntry{}, fmt.Errorf("find mood entry: %w", err)
	} else {
		if request.Version == nil || *request.Version != version {
			return MoodEntry{}, ErrVersionConflict
		}
		err = tx.QueryRow(ctx, `
			UPDATE mood_entries
			SET mood = $2, note = $3, shared_couple_id = $4,
			    version = version + 1, updated_at = NOW()
			WHERE id = $1
			RETURNING version, created_at, updated_at
		`, id, request.Mood, request.Note, newCoupleID).
			Scan(&version, &createdAt, &updatedAt)
		if err != nil {
			return MoodEntry{}, fmt.Errorf("update mood entry: %w", err)
		}
	}

	entry := MoodEntry{
		ID:               id.String(),
		EntryDate:        request.EntryDate,
		Mood:             request.Mood,
		Note:             request.Note,
		SharedWithCouple: newCoupleID != nil,
		IsMine:           true,
		Version:          version,
		CreatedAt:        createdAt,
		UpdatedAt:        updatedAt,
	}
	if err := recordMoodChanges(ctx, tx, userID, id, oldCoupleID, newCoupleID, entry); err != nil {
		return MoodEntry{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return MoodEntry{}, fmt.Errorf("commit mood entry: %w", err)
	}
	return entry, nil
}

func (r *Repository) ListSleep(
	ctx context.Context,
	userID uuid.UUID,
	rangeDates dateRange,
) ([]SleepEntry, error) {
	rows, err := r.db.Query(ctx, `
		SELECT e.id, e.entry_date, e.sleep_started_at, e.woke_at, e.quality,
		       COALESCE(e.shared_couple_id = (
		         SELECT cm.couple_id
		         FROM couple_members cm
		         JOIN couples c ON c.id = cm.couple_id
		         WHERE cm.user_id = $1 AND cm.left_at IS NULL
		           AND c.status = 'active'
		         LIMIT 1
		       ), FALSE),
		       e.owner_user_id = $1, e.version, e.created_at, e.updated_at
		FROM sleep_entries e
		WHERE e.deleted_at IS NULL
		  AND e.entry_date BETWEEN $2::DATE AND $3::DATE
		  AND (
		    e.owner_user_id = $1
		    OR e.shared_couple_id = (
		      SELECT cm.couple_id
		      FROM couple_members cm
		      JOIN couples c ON c.id = cm.couple_id
		      WHERE cm.user_id = $1 AND cm.left_at IS NULL
		        AND c.status = 'active'
		      LIMIT 1
		    )
		  )
		ORDER BY e.entry_date DESC, e.id
		LIMIT 732
	`, userID, rangeDates.From, rangeDates.To)
	if err != nil {
		return nil, fmt.Errorf("list sleep entries: %w", err)
	}
	defer rows.Close()

	entries := make([]SleepEntry, 0)
	for rows.Next() {
		var entry SleepEntry
		var id uuid.UUID
		var entryDate time.Time
		if err := rows.Scan(
			&id, &entryDate, &entry.SleepStartedAt, &entry.WokeAt,
			&entry.Quality, &entry.SharedWithCouple, &entry.IsMine,
			&entry.Version, &entry.CreatedAt, &entry.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan sleep entry: %w", err)
		}
		entry.ID = id.String()
		entry.EntryDate = entryDate.Format("2006-01-02")
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate sleep entries: %w", err)
	}
	return entries, nil
}

func (r *Repository) SaveSleep(
	ctx context.Context,
	userID uuid.UUID,
	request SaveSleepRequest,
) (SleepEntry, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return SleepEntry{}, fmt.Errorf("begin sleep entry save: %w", err)
	}
	defer tx.Rollback(ctx)

	var newCoupleID *uuid.UUID
	if request.SharedWithCouple {
		newCoupleID, err = activeCoupleID(ctx, tx, userID)
		if err != nil {
			return SleepEntry{}, err
		}
	}

	var (
		id          uuid.UUID
		oldCoupleID *uuid.UUID
		version     int64
		createdAt   time.Time
		updatedAt   time.Time
	)
	err = tx.QueryRow(ctx, `
		SELECT id, shared_couple_id, version, created_at, updated_at
		FROM sleep_entries
		WHERE owner_user_id = $1
		  AND entry_date = $2::DATE
		  AND deleted_at IS NULL
		FOR UPDATE
	`, userID, request.EntryDate).Scan(
		&id, &oldCoupleID, &version, &createdAt, &updatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		if request.Version != nil {
			return SleepEntry{}, ErrVersionConflict
		}
		err = tx.QueryRow(ctx, `
			INSERT INTO sleep_entries (
				owner_user_id, entry_date, sleep_started_at, woke_at,
				quality, shared_couple_id
			)
			VALUES ($1, $2::DATE, $3, $4, $5, $6)
			RETURNING id, version, created_at, updated_at
		`, userID, request.EntryDate, request.SleepStartedAt, request.WokeAt, request.Quality, newCoupleID).
			Scan(&id, &version, &createdAt, &updatedAt)
		if err != nil {
			if isUniqueViolation(err, "sleep_entries_owner_date_unique_idx") {
				return SleepEntry{}, ErrVersionConflict
			}
			return SleepEntry{}, fmt.Errorf("insert sleep entry: %w", err)
		}
	} else if err != nil {
		return SleepEntry{}, fmt.Errorf("find sleep entry: %w", err)
	} else {
		if request.Version == nil || *request.Version != version {
			return SleepEntry{}, ErrVersionConflict
		}
		err = tx.QueryRow(ctx, `
			UPDATE sleep_entries
			SET sleep_started_at = $2, woke_at = $3, quality = $4,
			    shared_couple_id = $5, version = version + 1, updated_at = NOW()
			WHERE id = $1
			RETURNING version, created_at, updated_at
		`, id, request.SleepStartedAt, request.WokeAt, request.Quality, newCoupleID).
			Scan(&version, &createdAt, &updatedAt)
		if err != nil {
			if isConstraintViolation(err, "sleep_entries_quality_check") ||
				isConstraintViolation(err, "sleep_entries_time_check") {
				return SleepEntry{}, ErrInvalidEntry
			}
			return SleepEntry{}, fmt.Errorf("update sleep entry: %w", err)
		}
	}

	entry := SleepEntry{
		ID:               id.String(),
		EntryDate:        request.EntryDate,
		SleepStartedAt:   request.SleepStartedAt,
		WokeAt:           request.WokeAt,
		Quality:          request.Quality,
		SharedWithCouple: newCoupleID != nil,
		IsMine:           true,
		Version:          version,
		CreatedAt:        createdAt,
		UpdatedAt:        updatedAt,
	}
	if err := recordSleepChanges(ctx, tx, userID, id, oldCoupleID, newCoupleID, entry); err != nil {
		return SleepEntry{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return SleepEntry{}, fmt.Errorf("commit sleep entry: %w", err)
	}
	return entry, nil
}

func activeCoupleID(ctx context.Context, tx pgx.Tx, userID uuid.UUID) (*uuid.UUID, error) {
	var coupleID uuid.UUID
	err := tx.QueryRow(ctx, `
		SELECT cm.couple_id
		FROM couple_members cm
		JOIN couples c ON c.id = cm.couple_id
		WHERE cm.user_id = $1 AND cm.left_at IS NULL AND c.status = 'active'
		LIMIT 1
	`, userID).Scan(&coupleID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNoActiveCouple
	}
	if err != nil {
		return nil, fmt.Errorf("find active couple: %w", err)
	}
	return &coupleID, nil
}

func recordMoodChanges(
	ctx context.Context,
	tx pgx.Tx,
	userID uuid.UUID,
	entryID uuid.UUID,
	oldCoupleID *uuid.UUID,
	newCoupleID *uuid.UUID,
	entry MoodEntry,
) error {
	personalPayload, err := json.Marshal(syncPayloadMood{
		EntryDate: entry.EntryDate,
		Mood:      entry.Mood,
		Note:      entry.Note,
	})
	if err != nil {
		return fmt.Errorf("encode mood entry sync payload: %w", err)
	}
	if err := recordChange(ctx, tx, userID, nil, "mood_entry", entryID, "upsert", personalPayload); err != nil {
		return err
	}
	couplePayload, err := json.Marshal(syncPayloadMood{
		EntryDate: entry.EntryDate,
		Mood:      entry.Mood,
	})
	if err != nil {
		return fmt.Errorf("encode shared mood sync payload: %w", err)
	}
	return recordSharedChange(ctx, tx, userID, oldCoupleID, newCoupleID, "mood_entry_shared", entryID, couplePayload)
}

func recordSleepChanges(
	ctx context.Context,
	tx pgx.Tx,
	userID uuid.UUID,
	entryID uuid.UUID,
	oldCoupleID *uuid.UUID,
	newCoupleID *uuid.UUID,
	entry SleepEntry,
) error {
	payload, err := json.Marshal(syncPayloadSleep{
		EntryDate:      entry.EntryDate,
		SleepStartedAt: entry.SleepStartedAt,
		WokeAt:         entry.WokeAt,
		Quality:        entry.Quality,
	})
	if err != nil {
		return fmt.Errorf("encode sleep entry sync payload: %w", err)
	}
	if err := recordChange(ctx, tx, userID, nil, "sleep_entry", entryID, "upsert", payload); err != nil {
		return err
	}
	return recordSharedChange(ctx, tx, userID, oldCoupleID, newCoupleID, "sleep_entry_shared", entryID, payload)
}

func recordSharedChange(
	ctx context.Context,
	tx pgx.Tx,
	userID uuid.UUID,
	oldCoupleID *uuid.UUID,
	newCoupleID *uuid.UUID,
	entityType string,
	entityID uuid.UUID,
	payload json.RawMessage,
) error {
	if oldCoupleID != nil && (newCoupleID == nil || *oldCoupleID != *newCoupleID) {
		var isMember bool
		if err := tx.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1
				FROM couple_members cm
				JOIN couples c ON c.id = cm.couple_id
				WHERE cm.user_id = $1 AND cm.couple_id = $2
				  AND cm.left_at IS NULL AND c.status = 'active'
			)
		`, userID, *oldCoupleID).Scan(&isMember); err != nil {
			return fmt.Errorf("check previous couple membership: %w", err)
		}
		if isMember {
			if err := recordChange(ctx, tx, userID, oldCoupleID, entityType, entityID, "delete", json.RawMessage(`{}`)); err != nil {
				return err
			}
		}
	}
	if newCoupleID != nil {
		return recordChange(ctx, tx, userID, newCoupleID, entityType, entityID, "upsert", payload)
	}
	return nil
}

func recordChange(
	ctx context.Context,
	tx pgx.Tx,
	userID uuid.UUID,
	coupleID *uuid.UUID,
	entityType string,
	entityID uuid.UUID,
	operation string,
	payload json.RawMessage,
) error {
	change := syncmodule.NewChange{
		ActorID:    userID,
		EntityType: entityType,
		EntityID:   entityID,
		Operation:  operation,
		Payload:    payload,
	}
	if coupleID == nil {
		change.Scope.UserID = &userID
	} else {
		change.Scope.CoupleID = coupleID
	}
	if _, err := syncmodule.RecordChange(ctx, tx, change); err != nil {
		return fmt.Errorf("record %s sync change: %w", entityType, err)
	}
	return nil
}

func isUniqueViolation(err error, constraint string) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) &&
		pgErr.Code == "23505" &&
		pgErr.ConstraintName == constraint
}

func isConstraintViolation(err error, constraint string) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) &&
		pgErr.Code == "23514" &&
		pgErr.ConstraintName == constraint
}
