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

func (r *Repository) applyHabitCompletionOperation(
	ctx context.Context,
	tx pgx.Tx,
	userID uuid.UUID,
	operation TaskSyncOperation,
) (TaskSyncResult, bool, error) {
	result := TaskSyncResult{
		OperationID: operation.OperationID,
		Status:      "applied",
	}
	payload := operation.HabitCompletionPayload
	coupleID, err := r.syncedHabitScope(ctx, tx, userID, payload.HabitID)
	if errors.Is(err, pgx.ErrNoRows) {
		result.Status = "conflict"
		result.Error = "not_found"
		return result, true, nil
	}
	if err != nil {
		return TaskSyncResult{}, false, err
	}

	if operation.Operation == "delete" {
		return r.uncompleteSyncedHabit(ctx, tx, userID, operation, coupleID, result)
	}
	return r.completeSyncedHabit(ctx, tx, userID, operation, coupleID, result)
}

func (r *Repository) syncedHabitScope(
	ctx context.Context,
	tx pgx.Tx,
	userID uuid.UUID,
	habitID uuid.UUID,
) (*uuid.UUID, error) {
	var coupleID *uuid.UUID
	err := tx.QueryRow(ctx, `
		SELECT h.couple_id
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
		FOR SHARE
	`, habitID, userID).Scan(&coupleID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, pgx.ErrNoRows
	}
	if err != nil {
		return nil, fmt.Errorf("check habit completion access: %w", err)
	}
	return coupleID, nil
}

func (r *Repository) completeSyncedHabit(
	ctx context.Context,
	tx pgx.Tx,
	userID uuid.UUID,
	operation TaskSyncOperation,
	coupleID *uuid.UUID,
	result TaskSyncResult,
) (TaskSyncResult, bool, error) {
	payload := operation.HabitCompletionPayload
	var completion syncedCompletionRecord
	err := tx.QueryRow(ctx, `
		INSERT INTO habit_completions (id, habit_id, user_id, completed_on)
		VALUES ($1, $2, $3, $4::DATE)
		ON CONFLICT DO NOTHING
		RETURNING id, created_at
	`, operation.EntityID, payload.HabitID, userID, payload.CompletedOn).Scan(
		&completion.ID,
		&completion.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		err = tx.QueryRow(ctx, `
			SELECT id, created_at
			FROM habit_completions
			WHERE habit_id = $1 AND user_id = $2
			  AND completed_on = $3::DATE AND deleted_at IS NULL
		`, payload.HabitID, userID, payload.CompletedOn).Scan(
			&completion.ID,
			&completion.CreatedAt,
		)
		if errors.Is(err, pgx.ErrNoRows) {
			result.Status = "conflict"
			result.Error = "entity_id_conflict"
			return result, true, nil
		}
		if err != nil {
			return TaskSyncResult{}, false, fmt.Errorf("find existing synced habit completion: %w", err)
		}
	} else if err != nil {
		return TaskSyncResult{}, false, fmt.Errorf("insert synced habit completion: %w", err)
	} else {
		completion.HabitID = payload.HabitID
		completion.UserID = userID
		completion.CompletedOn = payload.CompletedOn
		if err := recordHabitCompletionSyncChange(
			ctx, tx, userID, coupleID, completion.ID, "upsert", completion,
		); err != nil {
			return TaskSyncResult{}, false, err
		}
	}
	completion.HabitID = payload.HabitID
	completion.UserID = userID
	completion.CompletedOn = payload.CompletedOn
	completion.IsMine = true
	synced := toSyncedCompletion(completion)
	result.HabitCompletion = &synced
	return result, true, nil
}

func (r *Repository) uncompleteSyncedHabit(
	ctx context.Context,
	tx pgx.Tx,
	userID uuid.UUID,
	operation TaskSyncOperation,
	coupleID *uuid.UUID,
	result TaskSyncResult,
) (TaskSyncResult, bool, error) {
	payload := operation.HabitCompletionPayload
	var completion syncedCompletionRecord
	err := tx.QueryRow(ctx, `
		UPDATE habit_completions
		SET deleted_at = NOW()
		WHERE habit_id = $1 AND user_id = $2
		  AND completed_on = $3::DATE AND deleted_at IS NULL
		RETURNING id, created_at
	`, payload.HabitID, userID, payload.CompletedOn).Scan(
		&completion.ID,
		&completion.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return result, true, nil
	}
	if err != nil {
		return TaskSyncResult{}, false, fmt.Errorf("delete synced habit completion: %w", err)
	}
	completion.HabitID = payload.HabitID
	completion.UserID = userID
	completion.CompletedOn = payload.CompletedOn
	completion.IsMine = true
	if err := recordHabitCompletionSyncChange(
		ctx, tx, userID, coupleID, completion.ID, "delete", completion,
	); err != nil {
		return TaskSyncResult{}, false, err
	}
	synced := toSyncedCompletion(completion)
	result.HabitCompletion = &synced
	return result, true, nil
}

type syncedCompletionRecord struct {
	ID          uuid.UUID
	HabitID     uuid.UUID
	UserID      uuid.UUID
	CompletedOn string
	IsMine      bool
	CreatedAt   time.Time
}

func toSyncedCompletion(record syncedCompletionRecord) SyncedCompletion {
	return SyncedCompletion{
		ID:          record.ID.String(),
		HabitID:     record.HabitID.String(),
		UserID:      record.UserID.String(),
		CompletedOn: record.CompletedOn,
		IsMine:      record.IsMine,
		CreatedAt:   record.CreatedAt,
	}
}

func recordHabitCompletionSyncChange(
	ctx context.Context,
	tx pgx.Tx,
	userID uuid.UUID,
	coupleID *uuid.UUID,
	completionID uuid.UUID,
	operation string,
	completion syncedCompletionRecord,
) error {
	payload := json.RawMessage(`{}`)
	if operation == "upsert" {
		var err error
		payload, err = json.Marshal(struct {
			HabitID     string `json:"habit_id"`
			UserID      string `json:"user_id"`
			CompletedOn string `json:"completed_on"`
		}{
			HabitID:     completion.HabitID.String(),
			UserID:      userID.String(),
			CompletedOn: completion.CompletedOn,
		})
		if err != nil {
			return fmt.Errorf("encode habit completion sync payload: %w", err)
		}
	}
	return recordScopedChange(
		ctx, tx, userID, coupleID, "habit_completion", completionID, operation, payload,
	)
}
