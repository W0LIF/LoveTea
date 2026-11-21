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

func (r *Repository) applyCalendarEventOperation(
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
		return r.applyCalendarEventDeletion(ctx, tx, userID, operation, result)
	}

	payload := operation.CalendarEventPayload
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
		return r.insertSyncedCalendarEvent(ctx, tx, userID, operation, result, coupleID)
	}
	return r.updateSyncedCalendarEvent(ctx, tx, userID, operation, result, coupleID)
}

func (r *Repository) insertSyncedCalendarEvent(
	ctx context.Context,
	tx pgx.Tx,
	userID uuid.UUID,
	operation TaskSyncOperation,
	result TaskSyncResult,
	coupleID *uuid.UUID,
) (TaskSyncResult, bool, error) {
	payload := operation.CalendarEventPayload
	record, err := scanSyncedCalendarEvent(tx.QueryRow(ctx, `
		INSERT INTO calendar_events (
			id, owner_user_id, couple_id, title, description, event_type,
			starts_at, ends_at, color
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (id) DO NOTHING
		RETURNING id, title, description, event_type, starts_at, ends_at,
		          color, version, created_at, updated_at, couple_id
	`,
		operation.EntityID,
		userID,
		coupleID,
		payload.Title,
		payload.Description,
		payload.EventType,
		payload.StartsAt,
		payload.EndsAt,
		payload.Color,
	))
	if errors.Is(err, pgx.ErrNoRows) {
		result.Status = "conflict"
		result.Error = "entity_id_conflict"
		return result, true, nil
	}
	if err != nil {
		return TaskSyncResult{}, false, fmt.Errorf("insert synced calendar event: %w", err)
	}
	if err := recordCalendarEventChange(ctx, tx, userID, record, "upsert"); err != nil {
		return TaskSyncResult{}, false, err
	}
	event := toCalendarEvent(record)
	result.CalendarEvent = &event
	return result, true, nil
}

func (r *Repository) updateSyncedCalendarEvent(
	ctx context.Context,
	tx pgx.Tx,
	userID uuid.UUID,
	operation TaskSyncOperation,
	result TaskSyncResult,
	coupleID *uuid.UUID,
) (TaskSyncResult, bool, error) {
	payload := operation.CalendarEventPayload
	record, err := scanSyncedCalendarEvent(tx.QueryRow(ctx, `
		UPDATE calendar_events e
		SET title = $4, description = $5, event_type = $6,
		    starts_at = $7, ends_at = $8, color = $9::VARCHAR(7),
		    version = e.version + 1, updated_at = NOW()
		WHERE e.id = $1 AND e.version = $3 AND e.deleted_at IS NULL
		  AND (
		    ($10 = 'personal' AND e.owner_user_id = $2 AND e.couple_id IS NULL)
		    OR ($10 = 'couple' AND e.couple_id = $11)
		  )
		RETURNING e.id, e.title, e.description, e.event_type,
		          e.starts_at, e.ends_at, e.color, e.version,
		          e.created_at, e.updated_at, e.couple_id
	`,
		operation.EntityID,
		userID,
		operation.BaseVersion,
		payload.Title,
		payload.Description,
		payload.EventType,
		payload.StartsAt,
		payload.EndsAt,
		payload.Color,
		payload.Scope,
		coupleID,
	))
	if errors.Is(err, pgx.ErrNoRows) {
		return r.calendarEventSyncVersionResult(
			ctx, tx, userID, operation, payload.Scope, result,
		)
	}
	if err != nil {
		return TaskSyncResult{}, false, fmt.Errorf("update synced calendar event: %w", err)
	}
	if err := recordCalendarEventChange(ctx, tx, userID, record, "upsert"); err != nil {
		return TaskSyncResult{}, false, err
	}
	event := toCalendarEvent(record)
	result.CalendarEvent = &event
	return result, true, nil
}

func (r *Repository) applyCalendarEventDeletion(
	ctx context.Context,
	tx pgx.Tx,
	userID uuid.UUID,
	operation TaskSyncOperation,
	result TaskSyncResult,
) (TaskSyncResult, bool, error) {
	var coupleID *uuid.UUID
	err := tx.QueryRow(ctx, `
		UPDATE calendar_events e
		SET deleted_at = NOW(), version = e.version + 1, updated_at = NOW()
		WHERE e.id = $1 AND e.version = $3 AND e.deleted_at IS NULL
		  AND (
		    (e.owner_user_id = $2 AND e.couple_id IS NULL)
		    OR e.couple_id = (
		      SELECT cm.couple_id
		      FROM couple_members cm
		      JOIN couples c ON c.id = cm.couple_id
		      WHERE cm.user_id = $2 AND cm.left_at IS NULL
		        AND c.status = 'active'
		      LIMIT 1
		    )
		  )
		RETURNING e.couple_id, e.version
	`, operation.EntityID, userID, operation.BaseVersion).Scan(
		&coupleID,
		&result.Version,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return r.calendarEventSyncVersionResult(ctx, tx, userID, operation, "", result)
	}
	if err != nil {
		return TaskSyncResult{}, false, fmt.Errorf("delete synced calendar event: %w", err)
	}
	if err := recordScopedChange(
		ctx, tx, userID, coupleID, "calendar_event", operation.EntityID, "delete",
		json.RawMessage(`{}`),
	); err != nil {
		return TaskSyncResult{}, false, err
	}
	return result, true, nil
}

func (r *Repository) calendarEventSyncVersionResult(
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
			FROM calendar_events e
			WHERE e.id = $1 AND e.deleted_at IS NULL
			  AND (
			    (e.owner_user_id = $2 AND e.couple_id IS NULL)
			    OR e.couple_id = (
			      SELECT cm.couple_id
			      FROM couple_members cm
			      JOIN couples c ON c.id = cm.couple_id
			      WHERE cm.user_id = $2 AND cm.left_at IS NULL
			        AND c.status = 'active'
			      LIMIT 1
			    )
			  )
			  AND ($3 = '' OR
			       ($3 = 'personal' AND e.couple_id IS NULL) OR
			       ($3 = 'couple' AND e.couple_id IS NOT NULL))
		)
	`, operation.EntityID, userID, scope).Scan(&exists)
	if err != nil {
		return TaskSyncResult{}, false, fmt.Errorf("check synced calendar event access: %w", err)
	}
	result.Status = "conflict"
	if exists {
		result.Error = "version_conflict"
	} else {
		result.Error = "not_found"
	}
	return result, true, nil
}

type calendarEventRecord struct {
	ID          uuid.UUID
	Title       string
	Description string
	EventType   string
	StartsAt    time.Time
	EndsAt      time.Time
	Color       *string
	Version     int64
	CreatedAt   time.Time
	UpdatedAt   time.Time
	CoupleID    *uuid.UUID
}

func scanSyncedCalendarEvent(row pgx.Row) (calendarEventRecord, error) {
	var record calendarEventRecord
	err := row.Scan(
		&record.ID,
		&record.Title,
		&record.Description,
		&record.EventType,
		&record.StartsAt,
		&record.EndsAt,
		&record.Color,
		&record.Version,
		&record.CreatedAt,
		&record.UpdatedAt,
		&record.CoupleID,
	)
	return record, err
}

func toCalendarEvent(record calendarEventRecord) CalendarEvent {
	scope := "personal"
	if record.CoupleID != nil {
		scope = "couple"
	}
	return CalendarEvent{
		ID:          record.ID.String(),
		Scope:       scope,
		Title:       record.Title,
		Description: record.Description,
		EventType:   record.EventType,
		StartsAt:    record.StartsAt,
		EndsAt:      record.EndsAt,
		Color:       record.Color,
		Version:     record.Version,
		CreatedAt:   record.CreatedAt,
		UpdatedAt:   record.UpdatedAt,
	}
}

func recordCalendarEventChange(
	ctx context.Context,
	tx pgx.Tx,
	userID uuid.UUID,
	record calendarEventRecord,
	operation string,
) error {
	payload, err := json.Marshal(eventPayload{
		Title:       record.Title,
		Description: record.Description,
		EventType:   record.EventType,
		StartsAt:    record.StartsAt,
		EndsAt:      record.EndsAt,
		Color:       record.Color,
	})
	if err != nil {
		return fmt.Errorf("encode calendar event sync payload: %w", err)
	}
	return recordScopedChange(
		ctx, tx, userID, record.CoupleID, "calendar_event", record.ID, operation, payload,
	)
}
