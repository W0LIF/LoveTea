package planning

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

type fakeRepository struct {
	created       CreateTaskRequest
	updated       UpdateTaskRequest
	calendarEvent CreateCalendarEventRequest
	updatedEvent  UpdateCalendarEventRequest
	importantDate CreateImportantDateRequest
	updatedDate   UpdateImportantDateRequest
	listPage      ListPage
	err           error
}

func (r *fakeRepository) ListTasks(_ context.Context, _ uuid.UUID, page ListPage) ([]Task, error) {
	r.listPage = page
	return []Task{}, nil
}

func (r *fakeRepository) PushSyncOperations(
	context.Context,
	uuid.UUID,
	[]TaskSyncOperation,
) (SyncPushResponse, error) {
	return SyncPushResponse{}, nil
}

func (r *fakeRepository) CreateTask(
	_ context.Context,
	_ uuid.UUID,
	request CreateTaskRequest,
) (Task, error) {
	r.created = request
	return Task{Title: request.Title, Scope: request.Scope}, r.err
}

func (r *fakeRepository) UpdateTask(
	_ context.Context,
	_ uuid.UUID,
	_ uuid.UUID,
	request UpdateTaskRequest,
) (Task, error) {
	r.updated = request
	return Task{}, r.err
}

func (r *fakeRepository) DeleteTask(
	context.Context,
	uuid.UUID,
	uuid.UUID,
	int64,
) error {
	return r.err
}

func (r *fakeRepository) ListCalendarEvents(
	_ context.Context,
	_ uuid.UUID,
	page ListPage,
) ([]CalendarEvent, error) {
	r.listPage = page
	return []CalendarEvent{}, nil
}

func (r *fakeRepository) CreateCalendarEvent(
	_ context.Context,
	_ uuid.UUID,
	request CreateCalendarEventRequest,
) (CalendarEvent, error) {
	r.calendarEvent = request
	return CalendarEvent{Title: request.Title}, r.err
}

func (r *fakeRepository) UpdateCalendarEvent(
	_ context.Context,
	_ uuid.UUID,
	_ uuid.UUID,
	request UpdateCalendarEventRequest,
) (CalendarEvent, error) {
	r.updatedEvent = request
	return CalendarEvent{Title: *request.Title}, r.err
}

func (r *fakeRepository) DeleteCalendarEvent(
	context.Context,
	uuid.UUID,
	uuid.UUID,
	int64,
) error {
	return r.err
}

func (r *fakeRepository) ListImportantDates(
	_ context.Context,
	_ uuid.UUID,
	page ListPage,
) ([]ImportantDate, error) {
	r.listPage = page
	return []ImportantDate{}, nil
}

func TestListOperationsValidateAndPassPagination(t *testing.T) {
	repo := &fakeRepository{}
	service := NewService(repo)
	userID := uuid.New()
	page := ListPage{Limit: 25, Offset: 50}

	if _, err := service.ListTasks(context.Background(), userID, page); err != nil {
		t.Fatalf("ListTasks() error = %v", err)
	}
	if repo.listPage != page {
		t.Fatalf("ListTasks() page = %+v, want %+v", repo.listPage, page)
	}

	for _, list := range []func() error{
		func() error {
			_, err := service.ListTasks(context.Background(), userID, ListPage{Limit: 0})
			return err
		},
		func() error {
			_, err := service.ListCalendarEvents(context.Background(), userID, ListPage{Limit: 201})
			return err
		},
		func() error {
			_, err := service.ListImportantDates(
				context.Background(),
				userID,
				ListPage{Limit: 1, Offset: maxListOffset + 1},
			)
			return err
		},
	} {
		if err := list(); !errors.Is(err, ErrInvalidListPage) {
			t.Errorf("list operation error = %v, want invalid page", err)
		}
	}
}

func (r *fakeRepository) CreateImportantDate(
	_ context.Context,
	_ uuid.UUID,
	request CreateImportantDateRequest,
) (ImportantDate, error) {
	r.importantDate = request
	return ImportantDate{Title: request.Title, Date: request.Date}, r.err
}

func (r *fakeRepository) UpdateImportantDate(
	_ context.Context,
	_ uuid.UUID,
	_ uuid.UUID,
	request UpdateImportantDateRequest,
) (ImportantDate, error) {
	r.updatedDate = request
	return ImportantDate{Title: *request.Title, Date: *request.Date}, r.err
}

func (r *fakeRepository) DeleteImportantDate(
	context.Context,
	uuid.UUID,
	uuid.UUID,
	int64,
) error {
	return r.err
}

func TestCreateTaskValidatesAndNormalizesInput(t *testing.T) {
	repo := &fakeRepository{}
	service := NewService(repo)
	request := CreateTaskRequest{
		Scope:       "personal",
		Title:       "  Plan a walk  ",
		Description: "  Bring tea  ",
	}

	task, err := service.CreateTask(context.Background(), uuid.New(), request)
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}
	if task.Title != "Plan a walk" || repo.created.Description != "Bring tea" {
		t.Fatalf("CreateTask() did not normalize text: %+v", repo.created)
	}
}

func TestCreateTaskRejectsInvalidScopeAndReminder(t *testing.T) {
	service := NewService(&fakeRepository{})
	tests := []CreateTaskRequest{
		{Scope: "other", Title: "Valid title"},
		{Scope: "personal", Title: " ", Description: ""},
		{Scope: "personal", Title: "Valid title", Color: stringPointer("red")},
		{Scope: "personal", Title: "Valid title", ReminderAt: timePointer("2026-01-02T10:00:00Z")},
	}

	for _, request := range tests {
		if _, err := service.CreateTask(context.Background(), uuid.New(), request); !errors.Is(err, ErrInvalidTask) {
			t.Errorf("CreateTask(%+v) error = %v, want invalid task", request, err)
		}
	}
}

func TestUpdateTaskAcceptsExplicitClearsAndRequiresVersion(t *testing.T) {
	repo := &fakeRepository{}
	service := NewService(repo)
	request := UpdateTaskRequest{
		Version: 2,
		Color:   OptionalString{Set: true},
	}

	if _, err := service.UpdateTask(context.Background(), uuid.New(), uuid.New(), request); err != nil {
		t.Fatalf("UpdateTask() error = %v", err)
	}
	if !repo.updated.Color.Set || repo.updated.Color.Value != nil {
		t.Fatalf("explicit color clear was not preserved: %+v", repo.updated.Color)
	}

	request.Version = 0
	if _, err := service.UpdateTask(context.Background(), uuid.New(), uuid.New(), request); !errors.Is(err, ErrInvalidTask) {
		t.Fatalf("UpdateTask() error = %v, want invalid task", err)
	}
}

func TestUpdateTaskRejectsInvalidStateAndReminderRange(t *testing.T) {
	service := NewService(&fakeRepository{})
	laterReminder := timePointer("2026-01-02T11:00:00Z")
	tests := []UpdateTaskRequest{
		{Version: 1, Status: stringPointer("unknown")},
		{Version: 1, DueAt: OptionalTime{Set: true}, ReminderAt: OptionalTime{Value: laterReminder, Set: true}},
	}
	for _, request := range tests {
		if _, err := service.UpdateTask(context.Background(), uuid.New(), uuid.New(), request); !errors.Is(err, ErrInvalidTask) {
			t.Errorf("UpdateTask(%+v) error = %v, want invalid task", request, err)
		}
	}
}

func TestCreateCalendarEventValidatesTimeAndNormalizesText(t *testing.T) {
	repo := &fakeRepository{}
	start := timePointer("2026-01-02T10:00:00Z")
	end := timePointer("2026-01-02T11:00:00Z")
	request := CreateCalendarEventRequest{
		Scope:       "couple",
		Title:       "  Coffee  ",
		Description: "  Meet at noon  ",
		EventType:   "date",
		StartsAt:    *start,
		EndsAt:      *end,
	}

	event, err := NewService(repo).CreateCalendarEvent(context.Background(), uuid.New(), request)
	if err != nil {
		t.Fatalf("CreateCalendarEvent() error = %v", err)
	}
	if event.Title != "Coffee" || repo.calendarEvent.Description != "Meet at noon" {
		t.Fatalf("CreateCalendarEvent() did not normalize text: %+v", repo.calendarEvent)
	}
}

func TestCreateCalendarEventRejectsInvalidRange(t *testing.T) {
	start := timePointer("2026-01-02T11:00:00Z")
	end := timePointer("2026-01-02T10:00:00Z")
	_, err := NewService(&fakeRepository{}).CreateCalendarEvent(
		context.Background(),
		uuid.New(),
		CreateCalendarEventRequest{
			Scope:     "personal",
			Title:     "Invalid range",
			EventType: "busy",
			StartsAt:  *start,
			EndsAt:    *end,
		},
	)
	if !errors.Is(err, ErrInvalidCalendarEvent) {
		t.Fatalf("CreateCalendarEvent() error = %v, want invalid event", err)
	}
}

func TestUpdateCalendarEventValidatesVersionAndNullableColor(t *testing.T) {
	repo := &fakeRepository{}
	title := "  Dinner  "
	description := "  Reservation  "
	eventType := "date"
	startsAt := *timePointer("2026-01-02T18:00:00Z")
	endsAt := *timePointer("2026-01-02T20:00:00Z")
	request := UpdateCalendarEventRequest{
		Version:     2,
		Title:       &title,
		Description: &description,
		EventType:   &eventType,
		StartsAt:    &startsAt,
		EndsAt:      &endsAt,
		Color:       OptionalString{Set: true},
	}
	event, err := NewService(repo).UpdateCalendarEvent(
		context.Background(),
		uuid.New(),
		uuid.New(),
		request,
	)
	if err != nil {
		t.Fatalf("UpdateCalendarEvent() error = %v", err)
	}
	if event.Title != "Dinner" || *repo.updatedEvent.Description != "Reservation" ||
		!repo.updatedEvent.Color.Set || repo.updatedEvent.Color.Value != nil {
		t.Fatalf("UpdateCalendarEvent() did not normalize values or preserve null color: %+v", repo.updatedEvent)
	}
	request.Color.Set = false
	if _, err := NewService(&fakeRepository{}).UpdateCalendarEvent(
		context.Background(),
		uuid.New(),
		uuid.New(),
		request,
	); !errors.Is(err, ErrInvalidCalendarEvent) {
		t.Fatalf("UpdateCalendarEvent() without color field error = %v, want invalid event", err)
	}
}

func TestCreateImportantDateNormalizesDateAndValidatesReminders(t *testing.T) {
	repo := &fakeRepository{}
	service := NewService(repo)
	request := CreateImportantDateRequest{
		Scope:         "couple",
		Title:         "  Anniversary  ",
		Date:          "2026-02-14",
		RepeatsYearly: true,
		ReminderDays:  []int{30, 7, 0},
	}
	date, err := service.CreateImportantDate(context.Background(), uuid.New(), request)
	if err != nil {
		t.Fatalf("CreateImportantDate() error = %v", err)
	}
	if date.Title != "Anniversary" || repo.importantDate.Title != "Anniversary" ||
		repo.importantDate.Date != "2026-02-14" {
		t.Fatalf("CreateImportantDate() did not normalize values: %+v", repo.importantDate)
	}
}

func TestCreateImportantDateRejectsInvalidDateAndDuplicateReminder(t *testing.T) {
	service := NewService(&fakeRepository{})
	tests := []CreateImportantDateRequest{
		{Scope: "personal", Title: "Birthday", Date: "2026-02-30"},
		{Scope: "personal", Title: "Birthday", Date: "2026-02-14", ReminderDays: []int{7, 7}},
		{Scope: "personal", Title: "Birthday", Date: "2026-02-14", ReminderDays: []int{366}},
	}
	for _, request := range tests {
		if _, err := service.CreateImportantDate(context.Background(), uuid.New(), request); !errors.Is(err, ErrInvalidImportantDate) {
			t.Errorf("CreateImportantDate(%+v) error = %v, want invalid date", request, err)
		}
	}
}

func TestUpdateImportantDateValidatesVersionAndReminders(t *testing.T) {
	repo := &fakeRepository{}
	title := "  Anniversary  "
	dateValue := "2026-02-14"
	repeats := true
	reminderDays := []int{30, 7}
	request := UpdateImportantDateRequest{
		Version:       3,
		Title:         &title,
		Date:          &dateValue,
		RepeatsYearly: &repeats,
		ReminderDays:  &reminderDays,
	}
	date, err := NewService(repo).UpdateImportantDate(
		context.Background(),
		uuid.New(),
		uuid.New(),
		request,
	)
	if err != nil {
		t.Fatalf("UpdateImportantDate() error = %v", err)
	}
	if date.Title != "Anniversary" || *repo.updatedDate.Title != "Anniversary" {
		t.Fatalf("UpdateImportantDate() did not normalize title: %+v", repo.updatedDate)
	}
	request.Version = 0
	if _, err := NewService(&fakeRepository{}).UpdateImportantDate(
		context.Background(),
		uuid.New(),
		uuid.New(),
		request,
	); !errors.Is(err, ErrInvalidImportantDate) {
		t.Fatalf("UpdateImportantDate() error = %v, want invalid date", err)
	}
}

func stringPointer(value string) *string {
	return &value
}

func timePointer(value string) *time.Time {
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		panic(err)
	}
	return &parsed
}
