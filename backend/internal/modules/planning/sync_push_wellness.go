package planning

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var errWellnessEntityIDExists = errors.New("wellness entry id already exists")

func (r *Repository) applyWellnessOperation(
	ctx context.Context,
	tx pgx.Tx,
	userID uuid.UUID,
	operation TaskSyncOperation,
) (TaskSyncResult, bool, error) {
	result := TaskSyncResult{
		OperationID: operation.OperationID,
		Status:      "applied",
	}
	if operation.EntityType == "mood_entry" {
		return r.applyMoodEntryOperation(ctx, tx, userID, operation, result)
	}
	return r.applySleepEntryOperation(ctx, tx, userID, operation, result)
}

func (r *Repository) applyMoodEntryOperation(
	ctx context.Context,
	tx pgx.Tx,
	userID uuid.UUID,
	operation TaskSyncOperation,
	result TaskSyncResult,
) (TaskSyncResult, bool, error) {
	payload := operation.MoodPayload
	newCoupleID, err := r.resolveWellnessSharing(ctx, tx, userID, payload.SharedWithCouple)
	if errors.Is(err, ErrNoActiveCouple) {
		result.Status = "blocked"
		result.Error = "active_couple_required"
		return result, false, nil
	}
	if err != nil {
		return TaskSyncResult{}, false, err
	}

	record, exists, err := r.findMoodSyncRecord(ctx, tx, userID, operation.EntityID)
	if err != nil {
		return TaskSyncResult{}, false, err
	}
	if !exists {
		if operation.BaseVersion != 0 {
			return wellnessSyncConflict(result, "not_found")
		}
	} else {
		if operation.BaseVersion == 0 {
			return wellnessSyncConflict(result, "entity_id_conflict")
		}
		if record.Version != operation.BaseVersion {
			return wellnessSyncConflict(result, "version_conflict")
		}
	}

	if _, err := tx.Exec(ctx, "SAVEPOINT wellness_entry_upsert"); err != nil {
		return TaskSyncResult{}, false, fmt.Errorf("savepoint mood entry upsert: %w", err)
	}
	oldCoupleID := record.CoupleID
	if !exists {
		record, err = r.insertSyncedMoodEntry(ctx, tx, userID, operation, payload, newCoupleID)
	} else {
		record, err = r.updateSyncedMoodEntry(ctx, tx, userID, operation, payload, newCoupleID)
	}
	if err != nil {
		if isUniqueViolation(err, "mood_entries_owner_date_unique_idx") {
			if _, rollbackErr := tx.Exec(ctx, "ROLLBACK TO SAVEPOINT wellness_entry_upsert"); rollbackErr != nil {
				return TaskSyncResult{}, false, fmt.Errorf("rollback conflicting mood entry upsert: %w", rollbackErr)
			}
			if _, releaseErr := tx.Exec(ctx, "RELEASE SAVEPOINT wellness_entry_upsert"); releaseErr != nil {
				return TaskSyncResult{}, false, fmt.Errorf("release mood entry savepoint: %w", releaseErr)
			}
			result.Status = "conflict"
			result.Error = "entry_date_conflict"
			return result, true, nil
		}
		if errors.Is(err, errWellnessEntityIDExists) {
			if _, releaseErr := tx.Exec(ctx, "RELEASE SAVEPOINT wellness_entry_upsert"); releaseErr != nil {
				return TaskSyncResult{}, false, fmt.Errorf("release mood entry savepoint: %w", releaseErr)
			}
			result.Status = "conflict"
			result.Error = "entity_id_conflict"
			return result, true, nil
		}
		return TaskSyncResult{}, false, err
	}
	if _, err := tx.Exec(ctx, "RELEASE SAVEPOINT wellness_entry_upsert"); err != nil {
		return TaskSyncResult{}, false, fmt.Errorf("release mood entry savepoint: %w", err)
	}

	entry := toSyncedMoodEntry(record)
	if err := recordMoodSyncChanges(ctx, tx, userID, oldCoupleID, record.CoupleID, record, entry); err != nil {
		return TaskSyncResult{}, false, err
	}
	result.MoodEntry = &entry
	return result, true, nil
}

func (r *Repository) findMoodSyncRecord(
	ctx context.Context,
	tx pgx.Tx,
	userID uuid.UUID,
	entryID uuid.UUID,
) (moodSyncRecord, bool, error) {
	var record moodSyncRecord
	var date time.Time
	err := tx.QueryRow(ctx, `
		SELECT id, entry_date, mood, note, shared_couple_id,
		       version, created_at, updated_at
		FROM mood_entries
		WHERE id = $1 AND owner_user_id = $2 AND deleted_at IS NULL
		FOR UPDATE
	`, entryID, userID).Scan(
		&record.ID,
		&date,
		&record.Mood,
		&record.Note,
		&record.CoupleID,
		&record.Version,
		&record.CreatedAt,
		&record.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return moodSyncRecord{}, false, nil
	}
	if err != nil {
		return moodSyncRecord{}, false, fmt.Errorf("find synced mood entry: %w", err)
	}
	record.EntryDate = date.Format("2006-01-02")
	return record, true, nil
}

func (r *Repository) insertSyncedMoodEntry(
	ctx context.Context,
	tx pgx.Tx,
	userID uuid.UUID,
	operation TaskSyncOperation,
	payload *MoodSyncPayload,
	coupleID *uuid.UUID,
) (moodSyncRecord, error) {
	var record moodSyncRecord
	var date time.Time
	err := tx.QueryRow(ctx, `
		INSERT INTO mood_entries (
			id, owner_user_id, entry_date, mood, note, shared_couple_id
		)
		VALUES ($1, $2, $3::DATE, $4, $5, $6)
		ON CONFLICT (id) DO NOTHING
		RETURNING id, entry_date, mood, note, shared_couple_id,
		          version, created_at, updated_at
	`,
		operation.EntityID,
		userID,
		payload.EntryDate,
		payload.Mood,
		payload.Note,
		coupleID,
	).Scan(
		&record.ID,
		&date,
		&record.Mood,
		&record.Note,
		&record.CoupleID,
		&record.Version,
		&record.CreatedAt,
		&record.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return moodSyncRecord{}, errWellnessEntityIDExists
	}
	if err != nil {
		return moodSyncRecord{}, fmt.Errorf("insert synced mood entry: %w", err)
	}
	record.EntryDate = date.Format("2006-01-02")
	return record, nil
}

func (r *Repository) updateSyncedMoodEntry(
	ctx context.Context,
	tx pgx.Tx,
	userID uuid.UUID,
	operation TaskSyncOperation,
	payload *MoodSyncPayload,
	coupleID *uuid.UUID,
) (moodSyncRecord, error) {
	var record moodSyncRecord
	var date time.Time
	err := tx.QueryRow(ctx, `
		UPDATE mood_entries
		SET entry_date = $4::DATE, mood = $5, note = $6, shared_couple_id = $7,
		    version = version + 1, updated_at = NOW()
		WHERE id = $1 AND owner_user_id = $2 AND version = $3 AND deleted_at IS NULL
		RETURNING id, entry_date, mood, note, shared_couple_id,
		          version, created_at, updated_at
	`,
		operation.EntityID,
		userID,
		operation.BaseVersion,
		payload.EntryDate,
		payload.Mood,
		payload.Note,
		coupleID,
	).Scan(
		&record.ID,
		&date,
		&record.Mood,
		&record.Note,
		&record.CoupleID,
		&record.Version,
		&record.CreatedAt,
		&record.UpdatedAt,
	)
	if err != nil {
		return moodSyncRecord{}, fmt.Errorf("update synced mood entry: %w", err)
	}
	record.EntryDate = date.Format("2006-01-02")
	return record, nil
}

func (r *Repository) applySleepEntryOperation(
	ctx context.Context,
	tx pgx.Tx,
	userID uuid.UUID,
	operation TaskSyncOperation,
	result TaskSyncResult,
) (TaskSyncResult, bool, error) {
	payload := operation.SleepPayload
	newCoupleID, err := r.resolveWellnessSharing(ctx, tx, userID, payload.SharedWithCouple)
	if errors.Is(err, ErrNoActiveCouple) {
		result.Status = "blocked"
		result.Error = "active_couple_required"
		return result, false, nil
	}
	if err != nil {
		return TaskSyncResult{}, false, err
	}

	record, exists, err := r.findSleepSyncRecord(ctx, tx, userID, operation.EntityID)
	if err != nil {
		return TaskSyncResult{}, false, err
	}
	if !exists {
		if operation.BaseVersion != 0 {
			return wellnessSyncConflict(result, "not_found")
		}
	} else {
		if operation.BaseVersion == 0 {
			return wellnessSyncConflict(result, "entity_id_conflict")
		}
		if record.Version != operation.BaseVersion {
			return wellnessSyncConflict(result, "version_conflict")
		}
	}

	if _, err := tx.Exec(ctx, "SAVEPOINT wellness_entry_upsert"); err != nil {
		return TaskSyncResult{}, false, fmt.Errorf("savepoint sleep entry upsert: %w", err)
	}
	oldCoupleID := record.CoupleID
	if !exists {
		record, err = r.insertSyncedSleepEntry(ctx, tx, userID, operation, payload, newCoupleID)
	} else {
		record, err = r.updateSyncedSleepEntry(ctx, tx, userID, operation, payload, newCoupleID)
	}
	if err != nil {
		if isUniqueViolation(err, "sleep_entries_owner_date_unique_idx") {
			if _, rollbackErr := tx.Exec(ctx, "ROLLBACK TO SAVEPOINT wellness_entry_upsert"); rollbackErr != nil {
				return TaskSyncResult{}, false, fmt.Errorf("rollback conflicting sleep entry upsert: %w", rollbackErr)
			}
			if _, releaseErr := tx.Exec(ctx, "RELEASE SAVEPOINT wellness_entry_upsert"); releaseErr != nil {
				return TaskSyncResult{}, false, fmt.Errorf("release sleep entry savepoint: %w", releaseErr)
			}
			result.Status = "conflict"
			result.Error = "entry_date_conflict"
			return result, true, nil
		}
		if errors.Is(err, errWellnessEntityIDExists) {
			if _, releaseErr := tx.Exec(ctx, "RELEASE SAVEPOINT wellness_entry_upsert"); releaseErr != nil {
				return TaskSyncResult{}, false, fmt.Errorf("release sleep entry savepoint: %w", releaseErr)
			}
			result.Status = "conflict"
			result.Error = "entity_id_conflict"
			return result, true, nil
		}
		if isConstraintViolation(err, "sleep_entries_quality_check") ||
			isConstraintViolation(err, "sleep_entries_time_check") {
			return TaskSyncResult{}, false, ErrInvalidSyncPush
		}
		return TaskSyncResult{}, false, err
	}
	if _, err := tx.Exec(ctx, "RELEASE SAVEPOINT wellness_entry_upsert"); err != nil {
		return TaskSyncResult{}, false, fmt.Errorf("release sleep entry savepoint: %w", err)
	}

	entry := toSyncedSleepEntry(record)
	if err := recordSleepSyncChanges(ctx, tx, userID, oldCoupleID, record.CoupleID, record, entry); err != nil {
		return TaskSyncResult{}, false, err
	}
	result.SleepEntry = &entry
	return result, true, nil
}

func (r *Repository) findSleepSyncRecord(
	ctx context.Context,
	tx pgx.Tx,
	userID uuid.UUID,
	entryID uuid.UUID,
) (sleepSyncRecord, bool, error) {
	var record sleepSyncRecord
	var date time.Time
	err := tx.QueryRow(ctx, `
		SELECT id, entry_date, sleep_started_at, woke_at, quality,
		       shared_couple_id, version, created_at, updated_at
		FROM sleep_entries
		WHERE id = $1 AND owner_user_id = $2 AND deleted_at IS NULL
		FOR UPDATE
	`, entryID, userID).Scan(
		&record.ID,
		&date,
		&record.SleepStartedAt,
		&record.WokeAt,
		&record.Quality,
		&record.CoupleID,
		&record.Version,
		&record.CreatedAt,
		&record.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return sleepSyncRecord{}, false, nil
	}
	if err != nil {
		return sleepSyncRecord{}, false, fmt.Errorf("find synced sleep entry: %w", err)
	}
	record.EntryDate = date.Format("2006-01-02")
	return record, true, nil
}

func (r *Repository) insertSyncedSleepEntry(
	ctx context.Context,
	tx pgx.Tx,
	userID uuid.UUID,
	operation TaskSyncOperation,
	payload *SleepSyncPayload,
	coupleID *uuid.UUID,
) (sleepSyncRecord, error) {
	var record sleepSyncRecord
	var date time.Time
	err := tx.QueryRow(ctx, `
		INSERT INTO sleep_entries (
			id, owner_user_id, entry_date, sleep_started_at, woke_at,
			quality, shared_couple_id
		)
		VALUES ($1, $2, $3::DATE, $4, $5, $6, $7)
		ON CONFLICT (id) DO NOTHING
		RETURNING id, entry_date, sleep_started_at, woke_at, quality,
		          shared_couple_id, version, created_at, updated_at
	`,
		operation.EntityID,
		userID,
		payload.EntryDate,
		payload.SleepStartedAt,
		payload.WokeAt,
		payload.Quality,
		coupleID,
	).Scan(
		&record.ID,
		&date,
		&record.SleepStartedAt,
		&record.WokeAt,
		&record.Quality,
		&record.CoupleID,
		&record.Version,
		&record.CreatedAt,
		&record.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return sleepSyncRecord{}, errWellnessEntityIDExists
	}
	if err != nil {
		return sleepSyncRecord{}, fmt.Errorf("insert synced sleep entry: %w", err)
	}
	record.EntryDate = date.Format("2006-01-02")
	return record, nil
}

func (r *Repository) updateSyncedSleepEntry(
	ctx context.Context,
	tx pgx.Tx,
	userID uuid.UUID,
	operation TaskSyncOperation,
	payload *SleepSyncPayload,
	coupleID *uuid.UUID,
) (sleepSyncRecord, error) {
	var record sleepSyncRecord
	var date time.Time
	err := tx.QueryRow(ctx, `
		UPDATE sleep_entries
		SET entry_date = $4::DATE, sleep_started_at = $5, woke_at = $6,
		    quality = $7, shared_couple_id = $8, version = version + 1, updated_at = NOW()
		WHERE id = $1 AND owner_user_id = $2 AND version = $3 AND deleted_at IS NULL
		RETURNING id, entry_date, sleep_started_at, woke_at, quality,
		          shared_couple_id, version, created_at, updated_at
	`,
		operation.EntityID,
		userID,
		operation.BaseVersion,
		payload.EntryDate,
		payload.SleepStartedAt,
		payload.WokeAt,
		payload.Quality,
		coupleID,
	).Scan(
		&record.ID,
		&date,
		&record.SleepStartedAt,
		&record.WokeAt,
		&record.Quality,
		&record.CoupleID,
		&record.Version,
		&record.CreatedAt,
		&record.UpdatedAt,
	)
	if err != nil {
		return sleepSyncRecord{}, fmt.Errorf("update synced sleep entry: %w", err)
	}
	record.EntryDate = date.Format("2006-01-02")
	return record, nil
}

type moodSyncRecord struct {
	ID        uuid.UUID
	EntryDate string
	Mood      string
	Note      string
	CoupleID  *uuid.UUID
	Version   int64
	CreatedAt time.Time
	UpdatedAt time.Time
}

type sleepSyncRecord struct {
	ID             uuid.UUID
	EntryDate      string
	SleepStartedAt time.Time
	WokeAt         time.Time
	Quality        int
	CoupleID       *uuid.UUID
	Version        int64
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func toSyncedMoodEntry(record moodSyncRecord) SyncedMoodEntry {
	return SyncedMoodEntry{
		ID:               record.ID.String(),
		EntryDate:        record.EntryDate,
		Mood:             record.Mood,
		Note:             record.Note,
		SharedWithCouple: record.CoupleID != nil,
		IsMine:           true,
		Version:          record.Version,
		CreatedAt:        record.CreatedAt,
		UpdatedAt:        record.UpdatedAt,
	}
}

func toSyncedSleepEntry(record sleepSyncRecord) SyncedSleepEntry {
	return SyncedSleepEntry{
		ID:               record.ID.String(),
		EntryDate:        record.EntryDate,
		SleepStartedAt:   record.SleepStartedAt,
		WokeAt:           record.WokeAt,
		Quality:          record.Quality,
		SharedWithCouple: record.CoupleID != nil,
		IsMine:           true,
		Version:          record.Version,
		CreatedAt:        record.CreatedAt,
		UpdatedAt:        record.UpdatedAt,
	}
}

func (r *Repository) resolveWellnessSharing(
	ctx context.Context,
	tx pgx.Tx,
	userID uuid.UUID,
	shared bool,
) (*uuid.UUID, error) {
	if !shared {
		return nil, nil
	}
	return r.syncOperationScope(ctx, tx, userID, "couple")
}

func recordMoodSyncChanges(
	ctx context.Context,
	tx pgx.Tx,
	userID uuid.UUID,
	oldCoupleID *uuid.UUID,
	newCoupleID *uuid.UUID,
	record moodSyncRecord,
	entry SyncedMoodEntry,
) error {
	personalPayload, err := json.Marshal(struct {
		EntryDate string `json:"entry_date"`
		Mood      string `json:"mood"`
		Note      string `json:"note,omitempty"`
	}{
		EntryDate: entry.EntryDate,
		Mood:      entry.Mood,
		Note:      entry.Note,
	})
	if err != nil {
		return fmt.Errorf("encode private mood sync payload: %w", err)
	}
	if err := recordScopedChange(
		ctx, tx, userID, nil, "mood_entry", record.ID, "upsert", personalPayload,
	); err != nil {
		return err
	}
	couplePayload, err := json.Marshal(struct {
		EntryDate string `json:"entry_date"`
		Mood      string `json:"mood"`
	}{
		EntryDate: entry.EntryDate,
		Mood:      entry.Mood,
	})
	if err != nil {
		return fmt.Errorf("encode shared mood sync payload: %w", err)
	}
	return recordWellnessSharedChange(
		ctx, tx, userID, oldCoupleID, newCoupleID,
		"mood_entry_shared", record.ID, couplePayload,
	)
}

func recordSleepSyncChanges(
	ctx context.Context,
	tx pgx.Tx,
	userID uuid.UUID,
	oldCoupleID *uuid.UUID,
	newCoupleID *uuid.UUID,
	record sleepSyncRecord,
	entry SyncedSleepEntry,
) error {
	payload, err := json.Marshal(struct {
		EntryDate      string    `json:"entry_date"`
		SleepStartedAt time.Time `json:"sleep_started_at"`
		WokeAt         time.Time `json:"woke_at"`
		Quality        int       `json:"quality"`
	}{
		EntryDate:      entry.EntryDate,
		SleepStartedAt: entry.SleepStartedAt,
		WokeAt:         entry.WokeAt,
		Quality:        entry.Quality,
	})
	if err != nil {
		return fmt.Errorf("encode sleep sync payload: %w", err)
	}
	if err := recordScopedChange(
		ctx, tx, userID, nil, "sleep_entry", record.ID, "upsert", payload,
	); err != nil {
		return err
	}
	return recordWellnessSharedChange(
		ctx, tx, userID, oldCoupleID, newCoupleID,
		"sleep_entry_shared", record.ID, payload,
	)
}

func recordWellnessSharedChange(
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
			return fmt.Errorf("check previous wellness couple membership: %w", err)
		}
		if isMember {
			if err := recordScopedChange(
				ctx, tx, userID, oldCoupleID, entityType, entityID, "delete",
				json.RawMessage(`{}`),
			); err != nil {
				return err
			}
		}
	}
	if newCoupleID != nil {
		return recordScopedChange(ctx, tx, userID, newCoupleID, entityType, entityID, "upsert", payload)
	}
	return nil
}

func wellnessSyncConflict(result TaskSyncResult, code string) (TaskSyncResult, bool, error) {
	result.Status = "conflict"
	result.Error = code
	return result, true, nil
}

func isUniqueViolation(err error, constraint string) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) &&
		pgErr.Code == "23505" &&
		pgErr.ConstraintName == constraint
}
