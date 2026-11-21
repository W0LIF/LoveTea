package planning

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/W0LIF/LoveTea/backend/internal/modules/sync"
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

func (r *Repository) ListTasks(
	ctx context.Context,
	userID uuid.UUID,
	page ListPage,
) ([]Task, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, CASE WHEN couple_id IS NULL THEN 'personal' ELSE 'couple' END,
		       title, description, status, due_at, reminder_at, color,
		       version, created_at, updated_at
		FROM tasks
		WHERE deleted_at IS NULL
		  AND (
		    (owner_user_id = $1 AND couple_id IS NULL)
		    OR couple_id = (
		      SELECT couple_id
		      FROM couple_members
		      WHERE user_id = $1 AND left_at IS NULL
		      LIMIT 1
		    )
		  )
		ORDER BY due_at NULLS LAST, created_at DESC, id
		LIMIT $2 OFFSET $3
	`, userID, page.Limit, page.Offset)
	if err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}
	defer rows.Close()

	tasks := make([]Task, 0)
	for rows.Next() {
		task, err := scanTask(rows)
		if err != nil {
			return nil, fmt.Errorf("scan task: %w", err)
		}
		tasks = append(tasks, task)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate tasks: %w", err)
	}
	return tasks, nil
}

func (r *Repository) ListCalendarEvents(
	ctx context.Context,
	userID uuid.UUID,
	page ListPage,
) ([]CalendarEvent, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, CASE WHEN couple_id IS NULL THEN 'personal' ELSE 'couple' END,
		       title, description, event_type, starts_at, ends_at,
		       color, version, created_at, updated_at
		FROM calendar_events
		WHERE deleted_at IS NULL
		  AND (
		    (owner_user_id = $1 AND couple_id IS NULL)
		    OR couple_id = (
		      SELECT couple_id
		      FROM couple_members
		      WHERE user_id = $1 AND left_at IS NULL
		      LIMIT 1
		    )
		  )
		ORDER BY starts_at, id
		LIMIT $2 OFFSET $3
	`, userID, page.Limit, page.Offset)
	if err != nil {
		return nil, fmt.Errorf("list calendar events: %w", err)
	}
	defer rows.Close()

	events := make([]CalendarEvent, 0)
	for rows.Next() {
		var event CalendarEvent
		var eventID uuid.UUID
		if err := rows.Scan(
			&eventID,
			&event.Scope,
			&event.Title,
			&event.Description,
			&event.EventType,
			&event.StartsAt,
			&event.EndsAt,
			&event.Color,
			&event.Version,
			&event.CreatedAt,
			&event.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan calendar event: %w", err)
		}
		event.ID = eventID.String()
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate calendar events: %w", err)
	}
	return events, nil
}

func (r *Repository) CreateCalendarEvent(
	ctx context.Context,
	userID uuid.UUID,
	request CreateCalendarEventRequest,
) (CalendarEvent, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return CalendarEvent{}, fmt.Errorf("begin calendar event creation: %w", err)
	}
	defer tx.Rollback(ctx)

	coupleID, err := r.resolveScope(ctx, tx, userID, request.Scope)
	if err != nil {
		return CalendarEvent{}, err
	}

	var (
		event   CalendarEvent
		eventID uuid.UUID
	)
	err = tx.QueryRow(ctx, `
		INSERT INTO calendar_events (
			owner_user_id, couple_id, title, description, event_type,
			starts_at, ends_at, color
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, title, description, event_type, starts_at, ends_at,
		          color, version, created_at, updated_at
	`,
		userID,
		coupleID,
		request.Title,
		request.Description,
		request.EventType,
		request.StartsAt,
		request.EndsAt,
		request.Color,
	).Scan(
		&eventID,
		&event.Title,
		&event.Description,
		&event.EventType,
		&event.StartsAt,
		&event.EndsAt,
		&event.Color,
		&event.Version,
		&event.CreatedAt,
		&event.UpdatedAt,
	)
	if err != nil {
		return CalendarEvent{}, fmt.Errorf("insert calendar event: %w", err)
	}
	event.ID = eventID.String()
	event.Scope = request.Scope

	payload, err := json.Marshal(eventPayload{
		Title:       event.Title,
		Description: event.Description,
		EventType:   event.EventType,
		StartsAt:    event.StartsAt,
		EndsAt:      event.EndsAt,
		Color:       event.Color,
	})
	if err != nil {
		return CalendarEvent{}, fmt.Errorf("encode calendar event sync payload: %w", err)
	}
	if err := recordScopedChange(ctx, tx, userID, coupleID, "calendar_event", eventID, "upsert", payload); err != nil {
		return CalendarEvent{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return CalendarEvent{}, fmt.Errorf("commit calendar event: %w", err)
	}
	return event, nil
}

func (r *Repository) UpdateCalendarEvent(
	ctx context.Context,
	userID uuid.UUID,
	eventID uuid.UUID,
	request UpdateCalendarEventRequest,
) (CalendarEvent, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return CalendarEvent{}, fmt.Errorf("begin calendar event update: %w", err)
	}
	defer tx.Rollback(ctx)

	var (
		event           CalendarEvent
		eventIDReturned uuid.UUID
		coupleID        *uuid.UUID
	)
	err = tx.QueryRow(ctx, `
		UPDATE calendar_events e
		SET title = $4,
		    description = $5,
		    event_type = $6,
		    starts_at = $7,
		    ends_at = $8,
		    color = $9::VARCHAR(7),
		    version = e.version + 1,
		    updated_at = NOW()
		WHERE e.id = $1
		  AND e.version = $3
		  AND e.deleted_at IS NULL
		  AND (
		    (e.owner_user_id = $2 AND e.couple_id IS NULL)
		    OR e.couple_id = (
		      SELECT couple_id
		      FROM couple_members
		      WHERE user_id = $2 AND left_at IS NULL
		      LIMIT 1
		    )
		  )
		RETURNING e.id, e.title, e.description, e.event_type,
		          e.starts_at, e.ends_at, e.color, e.version,
		          e.created_at, e.updated_at, e.couple_id
	`,
		eventID,
		userID,
		request.Version,
		*request.Title,
		*request.Description,
		*request.EventType,
		*request.StartsAt,
		*request.EndsAt,
		request.Color.Value,
	).Scan(
		&eventIDReturned,
		&event.Title,
		&event.Description,
		&event.EventType,
		&event.StartsAt,
		&event.EndsAt,
		&event.Color,
		&event.Version,
		&event.CreatedAt,
		&event.UpdatedAt,
		&coupleID,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return CalendarEvent{}, r.calendarEventUpdateError(ctx, tx, eventID, userID)
	}
	if err != nil {
		if isConstraintViolation(err, "calendar_events_time_check") ||
			isConstraintViolation(err, "calendar_events_color_check") ||
			isConstraintViolation(err, "calendar_events_type_check") ||
			isConstraintViolation(err, "calendar_events_title_check") ||
			isConstraintViolation(err, "calendar_events_description_length_check") {
			return CalendarEvent{}, ErrInvalidCalendarEvent
		}
		return CalendarEvent{}, fmt.Errorf("update calendar event: %w", err)
	}
	event.ID = eventIDReturned.String()

	payload, err := json.Marshal(eventPayload{
		Title:       event.Title,
		Description: event.Description,
		EventType:   event.EventType,
		StartsAt:    event.StartsAt,
		EndsAt:      event.EndsAt,
		Color:       event.Color,
	})
	if err != nil {
		return CalendarEvent{}, fmt.Errorf("encode calendar event sync payload: %w", err)
	}
	if err := recordScopedChange(ctx, tx, userID, coupleID, "calendar_event", eventIDReturned, "upsert", payload); err != nil {
		return CalendarEvent{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return CalendarEvent{}, fmt.Errorf("commit calendar event update: %w", err)
	}
	if coupleID == nil {
		event.Scope = "personal"
	} else {
		event.Scope = "couple"
	}
	return event, nil
}

func (r *Repository) DeleteCalendarEvent(
	ctx context.Context,
	userID uuid.UUID,
	eventID uuid.UUID,
	version int64,
) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin calendar event deletion: %w", err)
	}
	defer tx.Rollback(ctx)

	var coupleID *uuid.UUID
	err = tx.QueryRow(ctx, `
		UPDATE calendar_events e
		SET deleted_at = NOW(), version = e.version + 1, updated_at = NOW()
		WHERE e.id = $1
		  AND e.version = $3
		  AND e.deleted_at IS NULL
		  AND (
		    (e.owner_user_id = $2 AND e.couple_id IS NULL)
		    OR e.couple_id = (
		      SELECT couple_id
		      FROM couple_members
		      WHERE user_id = $2 AND left_at IS NULL
		      LIMIT 1
		    )
		  )
		RETURNING e.couple_id
	`, eventID, userID, version).Scan(&coupleID)
	if errors.Is(err, pgx.ErrNoRows) {
		return r.calendarEventUpdateError(ctx, tx, eventID, userID)
	}
	if err != nil {
		return fmt.Errorf("delete calendar event: %w", err)
	}
	if err := recordScopedChange(ctx, tx, userID, coupleID, "calendar_event", eventID, "delete", json.RawMessage(`{}`)); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit calendar event deletion: %w", err)
	}
	return nil
}

func (r *Repository) ListImportantDates(
	ctx context.Context,
	userID uuid.UUID,
	page ListPage,
) ([]ImportantDate, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, CASE WHEN couple_id IS NULL THEN 'personal' ELSE 'couple' END,
		       title, event_date, repeats_yearly, reminder_days,
		       version, created_at, updated_at
		FROM important_dates
		WHERE deleted_at IS NULL
		  AND (
		    (owner_user_id = $1 AND couple_id IS NULL)
		    OR couple_id = (
		      SELECT couple_id
		      FROM couple_members
		      WHERE user_id = $1 AND left_at IS NULL
		      LIMIT 1
		    )
		  )
		ORDER BY event_date, id
		LIMIT $2 OFFSET $3
	`, userID, page.Limit, page.Offset)
	if err != nil {
		return nil, fmt.Errorf("list important dates: %w", err)
	}
	defer rows.Close()

	dates := make([]ImportantDate, 0)
	for rows.Next() {
		var (
			date         ImportantDate
			dateID       uuid.UUID
			eventDate    time.Time
			reminderDays []int16
		)
		if err := rows.Scan(
			&dateID,
			&date.Scope,
			&date.Title,
			&eventDate,
			&date.RepeatsYearly,
			&reminderDays,
			&date.Version,
			&date.CreatedAt,
			&date.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan important date: %w", err)
		}
		date.ID = dateID.String()
		date.Date = eventDate.Format("2006-01-02")
		date.ReminderDays = make([]int, 0, len(reminderDays))
		for _, day := range reminderDays {
			date.ReminderDays = append(date.ReminderDays, int(day))
		}
		dates = append(dates, date)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate important dates: %w", err)
	}
	return dates, nil
}

func (r *Repository) CreateImportantDate(
	ctx context.Context,
	userID uuid.UUID,
	request CreateImportantDateRequest,
) (ImportantDate, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return ImportantDate{}, fmt.Errorf("begin important date creation: %w", err)
	}
	defer tx.Rollback(ctx)

	coupleID, err := r.resolveScope(ctx, tx, userID, request.Scope)
	if err != nil {
		return ImportantDate{}, err
	}
	reminderDays := make([]int16, 0, len(request.ReminderDays))
	for _, day := range request.ReminderDays {
		reminderDays = append(reminderDays, int16(day))
	}

	var (
		date      ImportantDate
		dateID    uuid.UUID
		eventDate time.Time
	)
	err = tx.QueryRow(ctx, `
		INSERT INTO important_dates (
			owner_user_id, couple_id, title, event_date,
			repeats_yearly, reminder_days
		)
		VALUES ($1, $2, $3, $4::DATE, $5, $6)
		RETURNING id, title, event_date, repeats_yearly,
		          reminder_days, version, created_at, updated_at
	`,
		userID,
		coupleID,
		request.Title,
		request.Date,
		request.RepeatsYearly,
		reminderDays,
	).Scan(
		&dateID,
		&date.Title,
		&eventDate,
		&date.RepeatsYearly,
		&reminderDays,
		&date.Version,
		&date.CreatedAt,
		&date.UpdatedAt,
	)
	if err != nil {
		return ImportantDate{}, fmt.Errorf("insert important date: %w", err)
	}
	date.ID = dateID.String()
	date.Scope = request.Scope
	date.Date = eventDate.Format("2006-01-02")
	date.ReminderDays = append([]int(nil), request.ReminderDays...)

	payload, err := json.Marshal(importantDatePayload{
		Title:         date.Title,
		Date:          date.Date,
		RepeatsYearly: date.RepeatsYearly,
		ReminderDays:  date.ReminderDays,
	})
	if err != nil {
		return ImportantDate{}, fmt.Errorf("encode important date sync payload: %w", err)
	}
	if err := recordScopedChange(ctx, tx, userID, coupleID, "important_date", dateID, "upsert", payload); err != nil {
		return ImportantDate{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return ImportantDate{}, fmt.Errorf("commit important date: %w", err)
	}
	return date, nil
}

func (r *Repository) UpdateImportantDate(
	ctx context.Context,
	userID uuid.UUID,
	dateID uuid.UUID,
	request UpdateImportantDateRequest,
) (ImportantDate, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return ImportantDate{}, fmt.Errorf("begin important date update: %w", err)
	}
	defer tx.Rollback(ctx)

	reminderDays := make([]int16, 0, len(*request.ReminderDays))
	for _, day := range *request.ReminderDays {
		reminderDays = append(reminderDays, int16(day))
	}
	var (
		date       ImportantDate
		returnedID uuid.UUID
		eventDate  time.Time
		storedDays []int16
		coupleID   *uuid.UUID
	)
	err = tx.QueryRow(ctx, `
		UPDATE important_dates d
		SET title = $4,
		    event_date = $5::DATE,
		    repeats_yearly = $6,
		    reminder_days = $7,
		    version = d.version + 1,
		    updated_at = NOW()
		WHERE d.id = $1
		  AND d.version = $3
		  AND d.deleted_at IS NULL
		  AND (
		    (d.owner_user_id = $2 AND d.couple_id IS NULL)
		    OR d.couple_id = (
		      SELECT couple_id
		      FROM couple_members
		      WHERE user_id = $2 AND left_at IS NULL
		      LIMIT 1
		    )
		  )
		RETURNING d.id, d.title, d.event_date, d.repeats_yearly,
		          d.reminder_days, d.version, d.created_at, d.updated_at,
		          d.couple_id
	`,
		dateID,
		userID,
		request.Version,
		*request.Title,
		*request.Date,
		*request.RepeatsYearly,
		reminderDays,
	).Scan(
		&returnedID,
		&date.Title,
		&eventDate,
		&date.RepeatsYearly,
		&storedDays,
		&date.Version,
		&date.CreatedAt,
		&date.UpdatedAt,
		&coupleID,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return ImportantDate{}, r.importantDateUpdateError(ctx, tx, dateID, userID)
	}
	if err != nil {
		if isConstraintViolation(err, "important_dates_title_check") ||
			isConstraintViolation(err, "important_dates_reminder_days_check") {
			return ImportantDate{}, ErrInvalidImportantDate
		}
		return ImportantDate{}, fmt.Errorf("update important date: %w", err)
	}
	date.ID = returnedID.String()
	date.Date = eventDate.Format("2006-01-02")
	if coupleID == nil {
		date.Scope = "personal"
	} else {
		date.Scope = "couple"
	}
	date.ReminderDays = make([]int, 0, len(storedDays))
	for _, day := range storedDays {
		date.ReminderDays = append(date.ReminderDays, int(day))
	}

	payload, err := json.Marshal(importantDatePayload{
		Title:         date.Title,
		Date:          date.Date,
		RepeatsYearly: date.RepeatsYearly,
		ReminderDays:  date.ReminderDays,
	})
	if err != nil {
		return ImportantDate{}, fmt.Errorf("encode important date sync payload: %w", err)
	}
	if err := recordScopedChange(ctx, tx, userID, coupleID, "important_date", returnedID, "upsert", payload); err != nil {
		return ImportantDate{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return ImportantDate{}, fmt.Errorf("commit important date update: %w", err)
	}
	return date, nil
}

func (r *Repository) DeleteImportantDate(
	ctx context.Context,
	userID uuid.UUID,
	dateID uuid.UUID,
	version int64,
) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin important date deletion: %w", err)
	}
	defer tx.Rollback(ctx)

	var coupleID *uuid.UUID
	err = tx.QueryRow(ctx, `
		UPDATE important_dates d
		SET deleted_at = NOW(), version = d.version + 1, updated_at = NOW()
		WHERE d.id = $1
		  AND d.version = $3
		  AND d.deleted_at IS NULL
		  AND (
		    (d.owner_user_id = $2 AND d.couple_id IS NULL)
		    OR d.couple_id = (
		      SELECT couple_id
		      FROM couple_members
		      WHERE user_id = $2 AND left_at IS NULL
		      LIMIT 1
		    )
		  )
		RETURNING d.couple_id
	`, dateID, userID, version).Scan(&coupleID)
	if errors.Is(err, pgx.ErrNoRows) {
		return r.importantDateUpdateError(ctx, tx, dateID, userID)
	}
	if err != nil {
		return fmt.Errorf("delete important date: %w", err)
	}
	if err := recordScopedChange(ctx, tx, userID, coupleID, "important_date", dateID, "delete", json.RawMessage(`{}`)); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit important date deletion: %w", err)
	}
	return nil
}

func (r *Repository) calendarEventUpdateError(
	ctx context.Context,
	tx pgx.Tx,
	eventID uuid.UUID,
	userID uuid.UUID,
) error {
	var exists bool
	err := tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM calendar_events e
			WHERE e.id = $1 AND e.deleted_at IS NULL
			  AND (
			    (e.owner_user_id = $2 AND e.couple_id IS NULL)
			    OR e.couple_id = (
			      SELECT couple_id FROM couple_members
			      WHERE user_id = $2 AND left_at IS NULL LIMIT 1
			    )
			  )
		)
	`, eventID, userID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("check calendar event access: %w", err)
	}
	if exists {
		return ErrVersionConflict
	}
	return ErrCalendarEventNotFound
}

func (r *Repository) importantDateUpdateError(
	ctx context.Context,
	tx pgx.Tx,
	dateID uuid.UUID,
	userID uuid.UUID,
) error {
	var exists bool
	err := tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM important_dates d
			WHERE d.id = $1 AND d.deleted_at IS NULL
			  AND (
			    (d.owner_user_id = $2 AND d.couple_id IS NULL)
			    OR d.couple_id = (
			      SELECT couple_id FROM couple_members
			      WHERE user_id = $2 AND left_at IS NULL LIMIT 1
			    )
			  )
		)
	`, dateID, userID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("check important date access: %w", err)
	}
	if exists {
		return ErrVersionConflict
	}
	return ErrImportantDateNotFound
}

func (r *Repository) resolveScope(
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
		WHERE cm.user_id = $1
		  AND cm.left_at IS NULL
		  AND c.status = 'active'
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

type eventPayload struct {
	Title       string    `json:"title"`
	Description string    `json:"description"`
	EventType   string    `json:"event_type"`
	StartsAt    time.Time `json:"starts_at"`
	EndsAt      time.Time `json:"ends_at"`
	Color       *string   `json:"color"`
}

type importantDatePayload struct {
	Title         string `json:"title"`
	Date          string `json:"date"`
	RepeatsYearly bool   `json:"repeats_yearly"`
	ReminderDays  []int  `json:"reminder_days"`
}

func recordScopedChange(
	ctx context.Context,
	tx pgx.Tx,
	userID uuid.UUID,
	coupleID *uuid.UUID,
	entityType string,
	entityID uuid.UUID,
	operation string,
	payload json.RawMessage,
) error {
	change := sync.NewChange{
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
	if _, err := sync.RecordChange(ctx, tx, change); err != nil {
		return fmt.Errorf("record %s sync change: %w", entityType, err)
	}
	return nil
}

func (r *Repository) CreateTask(
	ctx context.Context,
	userID uuid.UUID,
	request CreateTaskRequest,
) (Task, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return Task{}, fmt.Errorf("begin task creation: %w", err)
	}
	defer tx.Rollback(ctx)

	var coupleID *uuid.UUID
	if request.Scope == "couple" {
		var activeCoupleID uuid.UUID
		err := tx.QueryRow(ctx, `
			SELECT cm.couple_id
			FROM couple_members cm
			JOIN couples c ON c.id = cm.couple_id
			WHERE cm.user_id = $1
			  AND cm.left_at IS NULL
			  AND c.status = 'active'
			LIMIT 1
		`, userID).Scan(&activeCoupleID)
		if errors.Is(err, pgx.ErrNoRows) {
			return Task{}, ErrNoActiveCouple
		}
		if err != nil {
			return Task{}, fmt.Errorf("find active couple for task: %w", err)
		}
		coupleID = &activeCoupleID
	}

	var record taskRecord
	err = tx.QueryRow(ctx, `
		INSERT INTO tasks (
			owner_user_id, couple_id, title, description,
			due_at, reminder_at, color
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, title, description, status, due_at, reminder_at,
		          color, version, created_at, updated_at
	`,
		userID,
		coupleID,
		request.Title,
		request.Description,
		request.DueAt,
		request.ReminderAt,
		request.Color,
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
	)
	if err != nil {
		return Task{}, fmt.Errorf("insert task: %w", err)
	}

	payload, err := json.Marshal(taskPayload{
		Title:       record.Title,
		Description: record.Description,
		Status:      record.Status,
		DueAt:       record.DueAt,
		ReminderAt:  record.ReminderAt,
		Color:       record.Color,
	})
	if err != nil {
		return Task{}, fmt.Errorf("encode task sync payload: %w", err)
	}

	change := sync.NewChange{
		ActorID:    userID,
		EntityType: "task",
		EntityID:   record.ID,
		Operation:  "upsert",
		Payload:    payload,
	}
	if coupleID == nil {
		change.Scope.UserID = &userID
	} else {
		change.Scope.CoupleID = coupleID
	}
	if _, err := sync.RecordChange(ctx, tx, change); err != nil {
		return Task{}, fmt.Errorf("record task sync change: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return Task{}, fmt.Errorf("commit task creation: %w", err)
	}

	record.Scope = request.Scope
	return toTask(record), nil
}

func (r *Repository) UpdateTask(
	ctx context.Context,
	userID uuid.UUID,
	taskID uuid.UUID,
	request UpdateTaskRequest,
) (Task, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return Task{}, fmt.Errorf("begin task update: %w", err)
	}
	defer tx.Rollback(ctx)

	var record taskRecord
	err = tx.QueryRow(ctx, `
		UPDATE tasks t
		SET title = COALESCE($4, t.title),
		    description = COALESCE($5, t.description),
		    status = COALESCE($6, t.status),
		    due_at = CASE WHEN $7 THEN $8::TIMESTAMPTZ ELSE t.due_at END,
		    reminder_at = CASE WHEN $9 THEN $10::TIMESTAMPTZ ELSE t.reminder_at END,
		    color = CASE WHEN $11 THEN $12::VARCHAR(7) ELSE t.color END,
		    version = t.version + 1,
		    updated_at = NOW()
		WHERE t.id = $1
		  AND t.version = $3
		  AND t.deleted_at IS NULL
		  AND (
		    (t.owner_user_id = $2 AND t.couple_id IS NULL)
		    OR t.couple_id = (
		      SELECT couple_id
		      FROM couple_members
		      WHERE user_id = $2 AND left_at IS NULL
		      LIMIT 1
		    )
		  )
		RETURNING t.id, t.title, t.description, t.status,
		          t.due_at, t.reminder_at, t.color, t.version,
		          t.created_at, t.updated_at, t.couple_id
	`,
		taskID,
		userID,
		request.Version,
		request.Title,
		request.Description,
		request.Status,
		request.DueAt.Set,
		request.DueAt.Value,
		request.ReminderAt.Set,
		request.ReminderAt.Value,
		request.Color.Set,
		request.Color.Value,
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
		return Task{}, r.taskUpdateError(ctx, tx, taskID, userID, request.Version)
	}
	if err != nil {
		if isConstraintViolation(err, "tasks_reminder_check") {
			return Task{}, ErrInvalidTask
		}
		return Task{}, fmt.Errorf("update task: %w", err)
	}

	if err := recordTaskChange(ctx, tx, userID, record, "upsert"); err != nil {
		return Task{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Task{}, fmt.Errorf("commit task update: %w", err)
	}
	return toTask(record), nil
}

func (r *Repository) DeleteTask(
	ctx context.Context,
	userID uuid.UUID,
	taskID uuid.UUID,
	version int64,
) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin task deletion: %w", err)
	}
	defer tx.Rollback(ctx)

	var record taskRecord
	err = tx.QueryRow(ctx, `
		UPDATE tasks t
		SET deleted_at = NOW(), version = t.version + 1, updated_at = NOW()
		WHERE t.id = $1
		  AND t.version = $3
		  AND t.deleted_at IS NULL
		  AND (
		    (t.owner_user_id = $2 AND t.couple_id IS NULL)
		    OR t.couple_id = (
		      SELECT couple_id
		      FROM couple_members
		      WHERE user_id = $2 AND left_at IS NULL
		      LIMIT 1
		    )
		  )
		RETURNING t.id, t.title, t.description, t.status,
		          t.due_at, t.reminder_at, t.color, t.version,
		          t.created_at, t.updated_at, t.couple_id
	`, taskID, userID, version).Scan(
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
		return r.taskDeleteError(ctx, tx, taskID, userID, version)
	}
	if err != nil {
		return fmt.Errorf("delete task: %w", err)
	}

	if err := recordTaskChange(ctx, tx, userID, record, "delete"); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit task deletion: %w", err)
	}
	return nil
}

func (r *Repository) taskUpdateError(
	ctx context.Context,
	tx pgx.Tx,
	taskID, userID uuid.UUID,
	version int64,
) error {
	var exists bool
	err := tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM tasks t
			WHERE t.id = $1 AND t.deleted_at IS NULL
			  AND (
			    (t.owner_user_id = $2 AND t.couple_id IS NULL)
			    OR t.couple_id = (
			      SELECT couple_id FROM couple_members
			      WHERE user_id = $2 AND left_at IS NULL LIMIT 1
			    )
			  )
		)
	`, taskID, userID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("check task update access: %w", err)
	}
	if exists {
		return ErrVersionConflict
	}
	return ErrTaskNotFound
}

func (r *Repository) taskDeleteError(
	ctx context.Context,
	tx pgx.Tx,
	taskID, userID uuid.UUID,
	version int64,
) error {
	var exists bool
	err := tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM tasks t
			WHERE t.id = $1 AND t.deleted_at IS NULL
			  AND (
			    (t.owner_user_id = $2 AND t.couple_id IS NULL)
			    OR t.couple_id = (
			      SELECT couple_id FROM couple_members
			      WHERE user_id = $2 AND left_at IS NULL LIMIT 1
			    )
			  )
		)
	`, taskID, userID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("check task deletion access: %w", err)
	}
	if exists {
		return ErrVersionConflict
	}
	return ErrTaskNotFound
}

type taskPayload struct {
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Status      string     `json:"status"`
	DueAt       *time.Time `json:"due_at"`
	ReminderAt  *time.Time `json:"reminder_at"`
	Color       *string    `json:"color"`
}

func recordTaskChange(
	ctx context.Context,
	tx pgx.Tx,
	userID uuid.UUID,
	record taskRecord,
	operation string,
) error {
	payload := json.RawMessage(`{}`)
	if operation == "upsert" {
		encoded, err := json.Marshal(taskPayload{
			Title:       record.Title,
			Description: record.Description,
			Status:      record.Status,
			DueAt:       record.DueAt,
			ReminderAt:  record.ReminderAt,
			Color:       record.Color,
		})
		if err != nil {
			return fmt.Errorf("encode task sync payload: %w", err)
		}
		payload = encoded
	}

	change := sync.NewChange{
		ActorID:    userID,
		EntityType: "task",
		EntityID:   record.ID,
		Operation:  operation,
		Payload:    payload,
	}
	if record.CoupleID == nil {
		change.Scope.UserID = &userID
	} else {
		change.Scope.CoupleID = record.CoupleID
	}
	if _, err := sync.RecordChange(ctx, tx, change); err != nil {
		return fmt.Errorf("record task sync change: %w", err)
	}
	return nil
}

func scanTask(rows pgx.Rows) (Task, error) {
	var record taskRecord
	if err := rows.Scan(
		&record.ID,
		&record.Scope,
		&record.Title,
		&record.Description,
		&record.Status,
		&record.DueAt,
		&record.ReminderAt,
		&record.Color,
		&record.Version,
		&record.CreatedAt,
		&record.UpdatedAt,
	); err != nil {
		return Task{}, err
	}
	return toTask(record), nil
}

func toTask(record taskRecord) Task {
	scope := record.Scope
	if scope == "" {
		if record.CoupleID == nil {
			scope = "personal"
		} else {
			scope = "couple"
		}
	}
	return Task{
		ID:          record.ID.String(),
		Scope:       scope,
		Title:       record.Title,
		Description: record.Description,
		Status:      record.Status,
		DueAt:       record.DueAt,
		ReminderAt:  record.ReminderAt,
		Color:       record.Color,
		Version:     record.Version,
		CreatedAt:   record.CreatedAt,
		UpdatedAt:   record.UpdatedAt,
	}
}

func isConstraintViolation(err error, constraint string) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) &&
		pgErr.Code == "23514" &&
		pgErr.ConstraintName == constraint
}
