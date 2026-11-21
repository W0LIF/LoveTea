package planning

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (r *Repository) PushSyncOperations(
	ctx context.Context,
	userID uuid.UUID,
	operations []TaskSyncOperation,
) (SyncPushResponse, error) {
	if _, err := r.db.Exec(ctx, `
		WITH expired AS (
			SELECT user_id, operation_id
			FROM sync_operation_receipts
			WHERE expires_at <= NOW()
			ORDER BY expires_at
			LIMIT 1000
			FOR UPDATE SKIP LOCKED
		)
		DELETE FROM sync_operation_receipts receipts
		USING expired
		WHERE receipts.user_id = expired.user_id
		  AND receipts.operation_id = expired.operation_id
	`); err != nil {
		return SyncPushResponse{}, fmt.Errorf("expire old sync operation receipts: %w", err)
	}

	response := SyncPushResponse{Results: make([]TaskSyncResult, 0, len(operations))}
	for _, operation := range operations {
		result, err := r.pushSyncOperation(ctx, userID, operation)
		if err != nil {
			return SyncPushResponse{}, err
		}
		response.Results = append(response.Results, result)
	}
	return response, nil
}

func (r *Repository) pushSyncOperation(
	ctx context.Context,
	userID uuid.UUID,
	operation TaskSyncOperation,
) (TaskSyncResult, error) {
	hash, err := syncOperationHash(operation)
	if err != nil {
		return TaskSyncResult{}, fmt.Errorf("hash sync operation: %w", err)
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return TaskSyncResult{}, fmt.Errorf("begin sync operation: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `
		DELETE FROM sync_operation_receipts
		WHERE user_id = $1 AND operation_id = $2 AND expires_at <= NOW()
	`, userID, operation.OperationID); err != nil {
		return TaskSyncResult{}, fmt.Errorf("expire sync operation receipt: %w", err)
	}

	tag, err := tx.Exec(ctx, `
		INSERT INTO sync_operation_receipts (user_id, operation_id, request_hash)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id, operation_id) DO NOTHING
	`, userID, operation.OperationID, hash)
	if err != nil {
		return TaskSyncResult{}, fmt.Errorf("reserve sync operation id: %w", err)
	}
	if tag.RowsAffected() == 0 {
		result, err := replaySyncOperation(ctx, tx, userID, operation, hash)
		if err != nil {
			return TaskSyncResult{}, err
		}
		return result, nil
	}

	result, persist, err := r.applySyncOperation(ctx, tx, userID, operation)
	if err != nil {
		return TaskSyncResult{}, err
	}
	if !persist {
		return result, nil
	}

	encoded, err := json.Marshal(result)
	if err != nil {
		return TaskSyncResult{}, fmt.Errorf("encode sync operation result: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE sync_operation_receipts
		SET result = $3::JSONB
		WHERE user_id = $1 AND operation_id = $2
	`, userID, operation.OperationID, encoded); err != nil {
		return TaskSyncResult{}, fmt.Errorf("save sync operation result: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return TaskSyncResult{}, fmt.Errorf("commit sync operation: %w", err)
	}
	return result, nil
}

func replaySyncOperation(
	ctx context.Context,
	tx pgx.Tx,
	userID uuid.UUID,
	operation TaskSyncOperation,
	hash []byte,
) (TaskSyncResult, error) {
	var storedHash, storedResult []byte
	err := tx.QueryRow(ctx, `
		SELECT request_hash, result
		FROM sync_operation_receipts
		WHERE user_id = $1 AND operation_id = $2
		FOR UPDATE
	`, userID, operation.OperationID).Scan(&storedHash, &storedResult)
	if err != nil {
		return TaskSyncResult{}, fmt.Errorf("read sync operation receipt: %w", err)
	}
	if !bytes.Equal(storedHash, hash) {
		return TaskSyncResult{
			OperationID: operation.OperationID,
			Status:      "conflict",
			Error:       "operation_id_reused",
		}, nil
	}
	if len(storedResult) == 0 {
		return TaskSyncResult{}, errors.New("sync operation receipt has no result")
	}

	var result TaskSyncResult
	if err := json.Unmarshal(storedResult, &result); err != nil {
		return TaskSyncResult{}, fmt.Errorf("decode sync operation receipt: %w", err)
	}
	result.Replayed = true
	return result, nil
}

func (r *Repository) applySyncOperation(
	ctx context.Context,
	tx pgx.Tx,
	userID uuid.UUID,
	operation TaskSyncOperation,
) (TaskSyncResult, bool, error) {
	if operation.EntityType == "habit_completion" {
		return r.applyHabitCompletionOperation(ctx, tx, userID, operation)
	}
	if operation.EntityType == "mood_entry" || operation.EntityType == "sleep_entry" {
		return r.applyWellnessOperation(ctx, tx, userID, operation)
	}
	if operation.EntityType == "habit" {
		return r.applyHabitOperation(ctx, tx, userID, operation)
	}
	if operation.EntityType == "calendar_event" {
		return r.applyCalendarEventOperation(ctx, tx, userID, operation)
	}
	if operation.EntityType == "important_date" {
		return r.applyImportantDateOperation(ctx, tx, userID, operation)
	}
	return r.applyTaskOperation(ctx, tx, userID, operation)
}

func (r *Repository) applyTaskOperation(
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
		return r.applyTaskDeletion(ctx, tx, userID, operation, result)
	}

	payload := operation.TaskPayload
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
		return r.insertSyncedTask(ctx, tx, userID, operation, result, coupleID)
	}
	return r.updateSyncedTask(ctx, tx, userID, operation, result, coupleID)
}

func (r *Repository) insertSyncedTask(
	ctx context.Context,
	tx pgx.Tx,
	userID uuid.UUID,
	operation TaskSyncOperation,
	result TaskSyncResult,
	coupleID *uuid.UUID,
) (TaskSyncResult, bool, error) {
	payload := operation.TaskPayload
	var record taskRecord
	err := tx.QueryRow(ctx, `
		INSERT INTO tasks (
			id, owner_user_id, couple_id, title, description, status,
			due_at, reminder_at, color
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (id) DO NOTHING
		RETURNING id, title, description, status, due_at, reminder_at,
		          color, version, created_at, updated_at, couple_id
	`,
		operation.EntityID,
		userID,
		coupleID,
		payload.Title,
		payload.Description,
		payload.Status,
		payload.DueAt,
		payload.ReminderAt,
		payload.Color,
	).Scan(
		&record.ID,
		&record.Title,
		&record.Description,
		&record.Status,
		&record.DueAt,
		&record.ReminderAt,
		&record.Color,
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
		if isConstraintViolation(err, "tasks_reminder_check") {
			return TaskSyncResult{}, false, ErrInvalidTask
		}
		return TaskSyncResult{}, false, fmt.Errorf("insert synced task: %w", err)
	}

	if err := recordTaskChange(ctx, tx, userID, record, "upsert"); err != nil {
		return TaskSyncResult{}, false, err
	}
	task := toTask(record)
	result.Task = &task
	return result, true, nil
}

func (r *Repository) updateSyncedTask(
	ctx context.Context,
	tx pgx.Tx,
	userID uuid.UUID,
	operation TaskSyncOperation,
	result TaskSyncResult,
	coupleID *uuid.UUID,
) (TaskSyncResult, bool, error) {
	payload := operation.TaskPayload
	var record taskRecord
	err := tx.QueryRow(ctx, `
		UPDATE tasks t
		SET title = $4, description = $5, status = $6,
		    due_at = $7, reminder_at = $8, color = $9,
		    version = t.version + 1, updated_at = NOW()
		WHERE t.id = $1 AND t.version = $3 AND t.deleted_at IS NULL
		  AND (
		    ($10 = 'personal' AND t.owner_user_id = $2 AND t.couple_id IS NULL)
		    OR ($10 = 'couple' AND t.couple_id = $11)
		  )
		RETURNING t.id, t.title, t.description, t.status,
		          t.due_at, t.reminder_at, t.color, t.version,
		          t.created_at, t.updated_at, t.couple_id
	`,
		operation.EntityID,
		userID,
		operation.BaseVersion,
		payload.Title,
		payload.Description,
		payload.Status,
		payload.DueAt,
		payload.ReminderAt,
		payload.Color,
		payload.Scope,
		coupleID,
	).Scan(
		&record.ID,
		&record.Title,
		&record.Description,
		&record.Status,
		&record.DueAt,
		&record.ReminderAt,
		&record.Color,
		&record.Version,
		&record.CreatedAt,
		&record.UpdatedAt,
		&record.CoupleID,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return r.taskSyncVersionResult(ctx, tx, userID, operation, payload.Scope, result)
	}
	if err != nil {
		if isConstraintViolation(err, "tasks_reminder_check") {
			return TaskSyncResult{}, false, ErrInvalidTask
		}
		return TaskSyncResult{}, false, fmt.Errorf("update synced task: %w", err)
	}

	if err := recordTaskChange(ctx, tx, userID, record, "upsert"); err != nil {
		return TaskSyncResult{}, false, err
	}
	task := toTask(record)
	result.Task = &task
	return result, true, nil
}

func (r *Repository) applyTaskDeletion(
	ctx context.Context,
	tx pgx.Tx,
	userID uuid.UUID,
	operation TaskSyncOperation,
	result TaskSyncResult,
) (TaskSyncResult, bool, error) {
	var record taskRecord
	err := tx.QueryRow(ctx, `
		UPDATE tasks t
		SET deleted_at = NOW(), version = t.version + 1, updated_at = NOW()
		WHERE t.id = $1 AND t.version = $3 AND t.deleted_at IS NULL
		  AND (
		    (t.owner_user_id = $2 AND t.couple_id IS NULL)
		    OR t.couple_id = (
		      SELECT cm.couple_id
		      FROM couple_members cm
		      JOIN couples c ON c.id = cm.couple_id
		      WHERE cm.user_id = $2 AND cm.left_at IS NULL
		        AND c.status = 'active'
		      LIMIT 1
		    )
		  )
		RETURNING t.id, t.title, t.description, t.status,
		          t.due_at, t.reminder_at, t.color, t.version,
		          t.created_at, t.updated_at, t.couple_id
	`, operation.EntityID, userID, operation.BaseVersion).Scan(
		&record.ID,
		&record.Title,
		&record.Description,
		&record.Status,
		&record.DueAt,
		&record.ReminderAt,
		&record.Color,
		&record.Version,
		&record.CreatedAt,
		&record.UpdatedAt,
		&record.CoupleID,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return r.taskSyncVersionResult(ctx, tx, userID, operation, "", result)
	}
	if err != nil {
		return TaskSyncResult{}, false, fmt.Errorf("delete synced task: %w", err)
	}

	if err := recordTaskChange(ctx, tx, userID, record, "delete"); err != nil {
		return TaskSyncResult{}, false, err
	}
	result.Version = record.Version
	return result, true, nil
}

func (r *Repository) taskSyncVersionResult(
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
			FROM tasks t
			WHERE t.id = $1 AND t.deleted_at IS NULL
			  AND (
			    (t.owner_user_id = $2 AND t.couple_id IS NULL)
			    OR t.couple_id = (
			      SELECT cm.couple_id
			      FROM couple_members cm
			      JOIN couples c ON c.id = cm.couple_id
			      WHERE cm.user_id = $2 AND cm.left_at IS NULL
			        AND c.status = 'active'
			      LIMIT 1
			    )
			  )
			  AND ($3 = '' OR
			       ($3 = 'personal' AND t.couple_id IS NULL) OR
			       ($3 = 'couple' AND t.couple_id IS NOT NULL))
		)
	`, operation.EntityID, userID, scope).Scan(&exists)
	if err != nil {
		return TaskSyncResult{}, false, fmt.Errorf("check synced task access: %w", err)
	}
	result.Status = "conflict"
	if exists {
		result.Error = "version_conflict"
	} else {
		result.Error = "not_found"
	}
	return result, true, nil
}

func (r *Repository) syncOperationScope(
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
		SELECT cm.couple_id
		FROM couple_members cm
		JOIN couples c ON c.id = cm.couple_id
		WHERE cm.user_id = $1 AND cm.left_at IS NULL
		  AND c.status = 'active'
		LIMIT 1
	`, userID).Scan(&coupleID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNoActiveCouple
	}
	if err != nil {
		return nil, fmt.Errorf("find active couple for sync operation: %w", err)
	}
	return &coupleID, nil
}
