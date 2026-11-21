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

func (r *Repository) applyImportantDateOperation(
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
		return r.applyImportantDateDeletion(ctx, tx, userID, operation, result)
	}

	payload := operation.ImportantDatePayload
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
		return r.insertSyncedImportantDate(ctx, tx, userID, operation, result, coupleID)
	}
	return r.updateSyncedImportantDate(ctx, tx, userID, operation, result, coupleID)
}

func (r *Repository) insertSyncedImportantDate(
	ctx context.Context,
	tx pgx.Tx,
	userID uuid.UUID,
	operation TaskSyncOperation,
	result TaskSyncResult,
	coupleID *uuid.UUID,
) (TaskSyncResult, bool, error) {
	payload := operation.ImportantDatePayload
	reminderDays := make([]int16, 0, len(payload.ReminderDays))
	for _, day := range payload.ReminderDays {
		reminderDays = append(reminderDays, int16(day))
	}

	record, err := scanSyncedImportantDate(tx.QueryRow(ctx, `
		INSERT INTO important_dates (
			id, owner_user_id, couple_id, title, event_date,
			repeats_yearly, reminder_days
		)
		VALUES ($1, $2, $3, $4, $5::DATE, $6, $7)
		ON CONFLICT (id) DO NOTHING
		RETURNING id, title, event_date, repeats_yearly, reminder_days,
		          version, created_at, updated_at, couple_id
	`,
		operation.EntityID,
		userID,
		coupleID,
		payload.Title,
		payload.Date,
		payload.RepeatsYearly,
		reminderDays,
	))
	if errors.Is(err, pgx.ErrNoRows) {
		result.Status = "conflict"
		result.Error = "entity_id_conflict"
		return result, true, nil
	}
	if err != nil {
		return TaskSyncResult{}, false, fmt.Errorf("insert synced important date: %w", err)
	}

	if err := recordImportantDateChange(ctx, tx, userID, record, "upsert"); err != nil {
		return TaskSyncResult{}, false, err
	}
	date := toImportantDate(record)
	result.ImportantDate = &date
	return result, true, nil
}

func (r *Repository) updateSyncedImportantDate(
	ctx context.Context,
	tx pgx.Tx,
	userID uuid.UUID,
	operation TaskSyncOperation,
	result TaskSyncResult,
	coupleID *uuid.UUID,
) (TaskSyncResult, bool, error) {
	payload := operation.ImportantDatePayload
	reminderDays := make([]int16, 0, len(payload.ReminderDays))
	for _, day := range payload.ReminderDays {
		reminderDays = append(reminderDays, int16(day))
	}

	record, err := scanSyncedImportantDate(tx.QueryRow(ctx, `
		UPDATE important_dates d
		SET title = $4, event_date = $5::DATE, repeats_yearly = $6,
		    reminder_days = $7, version = d.version + 1, updated_at = NOW()
		WHERE d.id = $1 AND d.version = $3 AND d.deleted_at IS NULL
		  AND (
		    ($8 = 'personal' AND d.owner_user_id = $2 AND d.couple_id IS NULL)
		    OR ($8 = 'couple' AND d.couple_id = $9)
		  )
		RETURNING d.id, d.title, d.event_date, d.repeats_yearly,
		          d.reminder_days, d.version, d.created_at, d.updated_at,
		          d.couple_id
	`,
		operation.EntityID,
		userID,
		operation.BaseVersion,
		payload.Title,
		payload.Date,
		payload.RepeatsYearly,
		reminderDays,
		payload.Scope,
		coupleID,
	))
	if errors.Is(err, pgx.ErrNoRows) {
		return r.importantDateSyncVersionResult(
			ctx, tx, userID, operation, payload.Scope, result,
		)
	}
	if err != nil {
		return TaskSyncResult{}, false, fmt.Errorf("update synced important date: %w", err)
	}

	if err := recordImportantDateChange(ctx, tx, userID, record, "upsert"); err != nil {
		return TaskSyncResult{}, false, err
	}
	date := toImportantDate(record)
	result.ImportantDate = &date
	return result, true, nil
}

func (r *Repository) applyImportantDateDeletion(
	ctx context.Context,
	tx pgx.Tx,
	userID uuid.UUID,
	operation TaskSyncOperation,
	result TaskSyncResult,
) (TaskSyncResult, bool, error) {
	var coupleID *uuid.UUID
	err := tx.QueryRow(ctx, `
		UPDATE important_dates d
		SET deleted_at = NOW(), version = d.version + 1, updated_at = NOW()
		WHERE d.id = $1 AND d.version = $3 AND d.deleted_at IS NULL
		  AND (
		    (d.owner_user_id = $2 AND d.couple_id IS NULL)
		    OR d.couple_id = (
		      SELECT cm.couple_id
		      FROM couple_members cm
		      JOIN couples c ON c.id = cm.couple_id
		      WHERE cm.user_id = $2 AND cm.left_at IS NULL
		        AND c.status = 'active'
		      LIMIT 1
		    )
		  )
		RETURNING d.couple_id, d.version
	`, operation.EntityID, userID, operation.BaseVersion).Scan(
		&coupleID,
		&result.Version,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return r.importantDateSyncVersionResult(ctx, tx, userID, operation, "", result)
	}
	if err != nil {
		return TaskSyncResult{}, false, fmt.Errorf("delete synced important date: %w", err)
	}
	if err := recordScopedChange(
		ctx, tx, userID, coupleID, "important_date", operation.EntityID, "delete",
		json.RawMessage(`{}`),
	); err != nil {
		return TaskSyncResult{}, false, err
	}
	return result, true, nil
}

func (r *Repository) importantDateSyncVersionResult(
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
			FROM important_dates d
			WHERE d.id = $1 AND d.deleted_at IS NULL
			  AND (
			    (d.owner_user_id = $2 AND d.couple_id IS NULL)
			    OR d.couple_id = (
			      SELECT cm.couple_id
			      FROM couple_members cm
			      JOIN couples c ON c.id = cm.couple_id
			      WHERE cm.user_id = $2 AND cm.left_at IS NULL
			        AND c.status = 'active'
			      LIMIT 1
			    )
			  )
			  AND ($3 = '' OR
			       ($3 = 'personal' AND d.couple_id IS NULL) OR
			       ($3 = 'couple' AND d.couple_id IS NOT NULL))
		)
	`, operation.EntityID, userID, scope).Scan(&exists)
	if err != nil {
		return TaskSyncResult{}, false, fmt.Errorf("check synced important date access: %w", err)
	}
	result.Status = "conflict"
	if exists {
		result.Error = "version_conflict"
	} else {
		result.Error = "not_found"
	}
	return result, true, nil
}

type importantDateRecord struct {
	ID            uuid.UUID
	Title         string
	Date          string
	RepeatsYearly bool
	ReminderDays  []int16
	Version       int64
	CreatedAt     time.Time
	UpdatedAt     time.Time
	CoupleID      *uuid.UUID
}

func scanSyncedImportantDate(row pgx.Row) (importantDateRecord, error) {
	var record importantDateRecord
	var date time.Time
	err := row.Scan(
		&record.ID,
		&record.Title,
		&date,
		&record.RepeatsYearly,
		&record.ReminderDays,
		&record.Version,
		&record.CreatedAt,
		&record.UpdatedAt,
		&record.CoupleID,
	)
	record.Date = date.Format("2006-01-02")
	return record, err
}

func toImportantDate(record importantDateRecord) ImportantDate {
	reminderDays := make([]int, 0, len(record.ReminderDays))
	for _, day := range record.ReminderDays {
		reminderDays = append(reminderDays, int(day))
	}
	scope := "personal"
	if record.CoupleID != nil {
		scope = "couple"
	}
	return ImportantDate{
		ID:            record.ID.String(),
		Scope:         scope,
		Title:         record.Title,
		Date:          record.Date,
		RepeatsYearly: record.RepeatsYearly,
		ReminderDays:  reminderDays,
		Version:       record.Version,
		CreatedAt:     record.CreatedAt,
		UpdatedAt:     record.UpdatedAt,
	}
}

func recordImportantDateChange(
	ctx context.Context,
	tx pgx.Tx,
	userID uuid.UUID,
	record importantDateRecord,
	operation string,
) error {
	payload, err := json.Marshal(importantDatePayload{
		Title:         record.Title,
		Date:          record.Date,
		RepeatsYearly: record.RepeatsYearly,
		ReminderDays:  toImportantDate(record).ReminderDays,
	})
	if err != nil {
		return fmt.Errorf("encode important date sync payload: %w", err)
	}
	return recordScopedChange(
		ctx, tx, userID, record.CoupleID, "important_date", record.ID, operation, payload,
	)
}
