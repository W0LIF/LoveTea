package planning

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (r *Repository) applyHabitOperation(
	ctx context.Context,
	tx pgx.Tx,
	userID uuid.UUID,
	operation TaskSyncOperation,
) (TaskSyncResult, bool, error) {
	result := TaskSyncResult{
		OperationID: operation.OperationID,
		Status:      "applied",
	}
	if operation.Operation == "delete" {
		return r.applyHabitDeletion(ctx, tx, userID, operation, result)
	}

	payload := operation.HabitPayload
	coupleID, err := r.syncOperationScope(ctx, tx, userID, payload.Scope)
	if errors.Is(err, ErrNoActiveCouple) {
		result.Status = "blocked"
		result.Error = "active_couple_required"
		return result, false, nil
	}
	if err != nil {
		return TaskSyncResult{}, false, err
	}
	if operation.BaseVersion == 0 {
		return r.insertSyncedHabit(ctx, tx, userID, operation, result, coupleID)
	}
	return r.updateSyncedHabit(ctx, tx, userID, operation, result, coupleID)
}

func (r *Repository) insertSyncedHabit(
	ctx context.Context,
	tx pgx.Tx,
	userID uuid.UUID,
	operation TaskSyncOperation,
	result TaskSyncResult,
	coupleID *uuid.UUID,
) (TaskSyncResult, bool, error) {
	payload := operation.HabitPayload
	var record habitSyncRecord
	err := tx.QueryRow(ctx, `
		INSERT INTO habits (
			id, owner_user_id, couple_id, title, description, target_per_week
		)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (id) DO NOTHING
		RETURNING id, title, description, target_per_week,
		          version, created_at, updated_at, couple_id
	`,
		operation.EntityID,
		userID,
		coupleID,
		payload.Title,
		payload.Description,
		payload.TargetPerWeek,
	).Scan(
		&record.ID,
		&record.Title,
		&record.Description,
		&record.TargetPerWeek,
		&record.Version,
		&record.CreatedAt,
		&record.UpdatedAt,
		&record.CoupleID,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		result.Status = "conflict"
		result.Error = "entity_id_conflict"
		return result, true, nil
	}
	if err != nil {
		return TaskSyncResult{}, false, fmt.Errorf("insert synced habit: %w", err)
	}
	if err := recordHabitSyncChange(ctx, tx, userID, record, "upsert"); err != nil {
		return TaskSyncResult{}, false, err
	}
	habit := toSyncedHabit(record)
	result.Habit = &habit
	return result, true, nil
}

func (r *Repository) updateSyncedHabit(
	ctx context.Context,
	tx pgx.Tx,
	userID uuid.UUID,
	operation TaskSyncOperation,
	result TaskSyncResult,
	coupleID *uuid.UUID,
) (TaskSyncResult, bool, error) {
	payload := operation.HabitPayload
	var record habitSyncRecord
	err := tx.QueryRow(ctx, `
		UPDATE habits h
		SET title = $4, description = $5, target_per_week = $6,
		    version = h.version + 1, updated_at = NOW()
		WHERE h.id = $1 AND h.version = $3 AND h.deleted_at IS NULL
		  AND (
		    ($7 = 'personal' AND h.owner_user_id = $2 AND h.couple_id IS NULL)
		    OR ($7 = 'couple' AND h.couple_id = $8)
		  )
		RETURNING h.id, h.title, h.description, h.target_per_week,
		          h.version, h.created_at, h.updated_at, h.couple_id
	`,
		operation.EntityID,
		userID,
		operation.BaseVersion,
		payload.Title,
		payload.Description,
		payload.TargetPerWeek,
		payload.Scope,
		coupleID,
	).Scan(
		&record.ID,
		&record.Title,
		&record.Description,
		&record.TargetPerWeek,
		&record.Version,
		&record.CreatedAt,
		&record.UpdatedAt,
		&record.CoupleID,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return r.habitSyncVersionResult(ctx, tx, userID, operation, payload.Scope, result)
	}
	if err != nil {
		return TaskSyncResult{}, false, fmt.Errorf("update synced habit: %w", err)
	}
	if err := recordHabitSyncChange(ctx, tx, userID, record, "upsert"); err != nil {
		return TaskSyncResult{}, false, err
	}
	habit := toSyncedHabit(record)
	result.Habit = &habit
	return result, true, nil
}

func (r *Repository) applyHabitDeletion(
	ctx context.Context,
	tx pgx.Tx,
	userID uuid.UUID,
	operation TaskSyncOperation,
	result TaskSyncResult,
) (TaskSyncResult, bool, error) {
	var coupleID *uuid.UUID
	err := tx.QueryRow(ctx, `
		UPDATE habits h
		SET deleted_at = NOW(), version = h.version + 1, updated_at = NOW()
		WHERE h.id = $1 AND h.version = $3 AND h.deleted_at IS NULL
		  AND (
		    (h.owner_user_id = $2 AND h.couple_id IS NULL)
		    OR h.couple_id = (
		      SELECT cm.couple_id
		      FROM couple_members cm
		      JOIN couples c ON c.id = cm.couple_id
		      WHERE cm.user_id = $2 AND cm.left_at IS NULL
		        AND c.status = 'active'
		      LIMIT 1
		    )
		  )
		RETURNING h.couple_id, h.version
	`, operation.EntityID, userID, operation.BaseVersion).Scan(
		&coupleID,
		&result.Version,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return r.habitSyncVersionResult(ctx, tx, userID, operation, "", result)
	}
	if err != nil {
		return TaskSyncResult{}, false, fmt.Errorf("delete synced habit: %w", err)
	}
	if err := recordScopedChange(
		ctx, tx, userID, coupleID, "habit", operation.EntityID, "delete",
		json.RawMessage(`{}`),
	); err != nil {
		return TaskSyncResult{}, false, err
	}
	return result, true, nil
}

func (r *Repository) habitSyncVersionResult(
	ctx context.Context,
	tx pgx.Tx,
	userID uuid.UUID,
	operation TaskSyncOperation,
	scope string,
	result TaskSyncResult,
) (TaskSyncResult, bool, error) {
	var exists bool
	err := tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM habits h
			WHERE h.id = $1 AND h.deleted_at IS NULL
			  AND (
			    (h.owner_user_id = $2 AND h.couple_id IS NULL)
			    OR h.couple_id = (
			      SELECT cm.couple_id
			      FROM couple_members cm
			      JOIN couples c ON c.id = cm.couple_id
			      WHERE cm.user_id = $2 AND cm.left_at IS NULL
			        AND c.status = 'active'
			      LIMIT 1
			    )
			  )
			  AND ($3 = '' OR
			       ($3 = 'personal' AND h.couple_id IS NULL) OR
			       ($3 = 'couple' AND h.couple_id IS NOT NULL))
		)
	`, operation.EntityID, userID, scope).Scan(&exists)
	if err != nil {
		return TaskSyncResult{}, false, fmt.Errorf("check synced habit access: %w", err)
	}
	result.Status = "conflict"
	if exists {
		result.Error = "version_conflict"
	} else {
		result.Error = "not_found"
	}
	return result, true, nil
}

type habitSyncRecord struct {
	ID            uuid.UUID
	Title         string
	Description   string
	TargetPerWeek int
	Version       int64
	CreatedAt     time.Time
	UpdatedAt     time.Time
	CoupleID      *uuid.UUID
}

func toSyncedHabit(record habitSyncRecord) SyncedHabit {
	scope := "personal"
	if record.CoupleID != nil {
		scope = "couple"
	}
	return SyncedHabit{
		ID:            record.ID.String(),
		Scope:         scope,
		Title:         record.Title,
		Description:   record.Description,
		TargetPerWeek: record.TargetPerWeek,
		Version:       record.Version,
		CreatedAt:     record.CreatedAt,
		UpdatedAt:     record.UpdatedAt,
	}
}

func recordHabitSyncChange(
	ctx context.Context,
	tx pgx.Tx,
	userID uuid.UUID,
	record habitSyncRecord,
	operation string,
) error {
	habit := toSyncedHabit(record)
	payload, err := json.Marshal(struct {
		Title         string `json:"title"`
		Description   string `json:"description"`
		TargetPerWeek int    `json:"target_per_week"`
		Scope         string `json:"scope"`
	}{
		Title:         habit.Title,
		Description:   habit.Description,
		TargetPerWeek: habit.TargetPerWeek,
		Scope:         habit.Scope,
	})
	if err != nil {
		return fmt.Errorf("encode habit sync payload: %w", err)
	}
	return recordScopedChange(ctx, tx, userID, record.CoupleID, "habit", record.ID, operation, payload)
}
