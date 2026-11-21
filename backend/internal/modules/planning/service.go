package planning

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

var (
	ErrInvalidTask           = errors.New("invalid task")
	ErrInvalidCalendarEvent  = errors.New("invalid calendar event")
	ErrInvalidImportantDate  = errors.New("invalid important date")
	ErrInvalidListPage       = errors.New("invalid list page")
	ErrTaskNotFound          = errors.New("task not found")
	ErrCalendarEventNotFound = errors.New("calendar event not found")
	ErrImportantDateNotFound = errors.New("important date not found")
	ErrVersionConflict       = errors.New("resource version conflict")
	ErrNoActiveCouple        = errors.New("user does not belong to an active couple")

	taskColorPattern = regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)
)

const (
	defaultListLimit = 200
	maxListLimit     = 200
	maxListOffset    = 1_000_000
)

type ListPage struct {
	Limit  int
	Offset int
}

type repository interface {
	ListTasks(context.Context, uuid.UUID, ListPage) ([]Task, error)
	PushSyncOperations(context.Context, uuid.UUID, []TaskSyncOperation) (SyncPushResponse, error)
	CreateTask(context.Context, uuid.UUID, CreateTaskRequest) (Task, error)
	UpdateTask(context.Context, uuid.UUID, uuid.UUID, UpdateTaskRequest) (Task, error)
	DeleteTask(context.Context, uuid.UUID, uuid.UUID, int64) error
	ListCalendarEvents(context.Context, uuid.UUID, ListPage) ([]CalendarEvent, error)
	CreateCalendarEvent(context.Context, uuid.UUID, CreateCalendarEventRequest) (CalendarEvent, error)
	UpdateCalendarEvent(context.Context, uuid.UUID, uuid.UUID, UpdateCalendarEventRequest) (CalendarEvent, error)
	DeleteCalendarEvent(context.Context, uuid.UUID, uuid.UUID, int64) error
	ListImportantDates(context.Context, uuid.UUID, ListPage) ([]ImportantDate, error)
	CreateImportantDate(context.Context, uuid.UUID, CreateImportantDateRequest) (ImportantDate, error)
	UpdateImportantDate(context.Context, uuid.UUID, uuid.UUID, UpdateImportantDateRequest) (ImportantDate, error)
	DeleteImportantDate(context.Context, uuid.UUID, uuid.UUID, int64) error
}

type Service struct {
	repo repository
}

func NewService(repo repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) ListTasks(ctx context.Context, userID uuid.UUID, page ListPage) ([]Task, error) {
	if err := validateListPage(page); err != nil {
		return nil, err
	}
	return s.repo.ListTasks(ctx, userID, page)
}

func (s *Service) ListCalendarEvents(
	ctx context.Context,
	userID uuid.UUID,
	page ListPage,
) ([]CalendarEvent, error) {
	if err := validateListPage(page); err != nil {
		return nil, err
	}
	return s.repo.ListCalendarEvents(ctx, userID, page)
}

func (s *Service) CreateCalendarEvent(
	ctx context.Context,
	userID uuid.UUID,
	request CreateCalendarEventRequest,
) (CalendarEvent, error) {
	request.Title = strings.TrimSpace(request.Title)
	request.Description = strings.TrimSpace(request.Description)
	if !validScope(request.Scope) ||
		!validTitle(request.Title) ||
		utf8.RuneCountInString(request.Description) > 5000 ||
		!validCalendarEventType(request.EventType) ||
		request.StartsAt.IsZero() ||
		!request.EndsAt.After(request.StartsAt) ||
		!validColor(request.Color) {
		return CalendarEvent{}, ErrInvalidCalendarEvent
	}
	return s.repo.CreateCalendarEvent(ctx, userID, request)
}

func (s *Service) UpdateCalendarEvent(
	ctx context.Context,
	userID uuid.UUID,
	eventID uuid.UUID,
	request UpdateCalendarEventRequest,
) (CalendarEvent, error) {
	if eventID == uuid.Nil ||
		request.Version < 1 ||
		request.Title == nil ||
		request.Description == nil ||
		request.EventType == nil ||
		request.StartsAt == nil ||
		request.EndsAt == nil ||
		!request.Color.Set {
		return CalendarEvent{}, ErrInvalidCalendarEvent
	}
	title := strings.TrimSpace(*request.Title)
	description := strings.TrimSpace(*request.Description)
	eventType := *request.EventType
	startsAt := *request.StartsAt
	endsAt := *request.EndsAt
	if !validTitle(title) ||
		utf8.RuneCountInString(description) > 5000 ||
		!validCalendarEventType(eventType) ||
		startsAt.IsZero() ||
		!endsAt.After(startsAt) ||
		(request.Color.Value != nil && !validColor(request.Color.Value)) {
		return CalendarEvent{}, ErrInvalidCalendarEvent
	}
	request.Title = &title
	request.Description = &description
	request.EventType = &eventType
	return s.repo.UpdateCalendarEvent(ctx, userID, eventID, request)
}

func (s *Service) DeleteCalendarEvent(
	ctx context.Context,
	userID uuid.UUID,
	eventID uuid.UUID,
	version int64,
) error {
	if eventID == uuid.Nil || version < 1 {
		return ErrInvalidCalendarEvent
	}
	return s.repo.DeleteCalendarEvent(ctx, userID, eventID, version)
}

func (s *Service) ListImportantDates(
	ctx context.Context,
	userID uuid.UUID,
	page ListPage,
) ([]ImportantDate, error) {
	if err := validateListPage(page); err != nil {
		return nil, err
	}
	return s.repo.ListImportantDates(ctx, userID, page)
}

func validateListPage(page ListPage) error {
	if page.Limit < 1 || page.Limit > maxListLimit ||
		page.Offset < 0 || page.Offset > maxListOffset {
		return ErrInvalidListPage
	}
	return nil
}

func (s *Service) CreateImportantDate(
	ctx context.Context,
	userID uuid.UUID,
	request CreateImportantDateRequest,
) (ImportantDate, error) {
	request.Title = strings.TrimSpace(request.Title)
	eventDate, err := time.Parse("2006-01-02", request.Date)
	if err != nil ||
		eventDate.Format("2006-01-02") != request.Date ||
		!validScope(request.Scope) ||
		!validTitle(request.Title) ||
		len(request.ReminderDays) > 10 {
		return ImportantDate{}, ErrInvalidImportantDate
	}
	request.Date = eventDate.Format("2006-01-02")
	seen := make(map[int]struct{}, len(request.ReminderDays))
	for _, day := range request.ReminderDays {
		if day < 0 || day > 365 {
			return ImportantDate{}, ErrInvalidImportantDate
		}
		if _, exists := seen[day]; exists {
			return ImportantDate{}, ErrInvalidImportantDate
		}
		seen[day] = struct{}{}
	}
	return s.repo.CreateImportantDate(ctx, userID, request)
}

func (s *Service) UpdateImportantDate(
	ctx context.Context,
	userID uuid.UUID,
	dateID uuid.UUID,
	request UpdateImportantDateRequest,
) (ImportantDate, error) {
	if dateID == uuid.Nil || request.Version < 1 ||
		request.Title == nil || request.Date == nil ||
		request.RepeatsYearly == nil || request.ReminderDays == nil {
		return ImportantDate{}, ErrInvalidImportantDate
	}
	title := strings.TrimSpace(*request.Title)
	eventDate, err := time.Parse("2006-01-02", *request.Date)
	if err != nil ||
		eventDate.Format("2006-01-02") != *request.Date ||
		!validTitle(title) ||
		len(*request.ReminderDays) > 10 {
		return ImportantDate{}, ErrInvalidImportantDate
	}
	normalizedDate := eventDate.Format("2006-01-02")
	request.Title = &title
	request.Date = &normalizedDate
	seen := make(map[int]struct{}, len(*request.ReminderDays))
	for _, day := range *request.ReminderDays {
		if day < 0 || day > 365 {
			return ImportantDate{}, ErrInvalidImportantDate
		}
		if _, exists := seen[day]; exists {
			return ImportantDate{}, ErrInvalidImportantDate
		}
		seen[day] = struct{}{}
	}
	return s.repo.UpdateImportantDate(ctx, userID, dateID, request)
}

func (s *Service) DeleteImportantDate(
	ctx context.Context,
	userID uuid.UUID,
	dateID uuid.UUID,
	version int64,
) error {
	if dateID == uuid.Nil || version < 1 {
		return ErrInvalidImportantDate
	}
	return s.repo.DeleteImportantDate(ctx, userID, dateID, version)
}

func (s *Service) CreateTask(
	ctx context.Context,
	userID uuid.UUID,
	request CreateTaskRequest,
) (Task, error) {
	if !validScope(request.Scope) {
		return Task{}, ErrInvalidTask
	}
	request.Title = strings.TrimSpace(request.Title)
	request.Description = strings.TrimSpace(request.Description)
	if !validTitle(request.Title) ||
		utf8.RuneCountInString(request.Description) > 5000 ||
		!validColor(request.Color) ||
		!validReminder(request.DueAt, request.ReminderAt) {
		return Task{}, ErrInvalidTask
	}
	return s.repo.CreateTask(ctx, userID, request)
}

func (s *Service) UpdateTask(
	ctx context.Context,
	userID uuid.UUID,
	taskID uuid.UUID,
	request UpdateTaskRequest,
) (Task, error) {
	if taskID == uuid.Nil || request.Version < 1 ||
		(request.Title == nil && request.Description == nil &&
			request.Status == nil && !request.DueAt.Set &&
			!request.ReminderAt.Set && !request.Color.Set) {
		return Task{}, ErrInvalidTask
	}
	if request.Title != nil {
		title := strings.TrimSpace(*request.Title)
		if !validTitle(title) {
			return Task{}, ErrInvalidTask
		}
		request.Title = &title
	}
	if request.Description != nil {
		description := strings.TrimSpace(*request.Description)
		if utf8.RuneCountInString(description) > 5000 {
			return Task{}, ErrInvalidTask
		}
		request.Description = &description
	}
	if request.Status != nil && !validTaskStatus(*request.Status) {
		return Task{}, ErrInvalidTask
	}
	if request.Color.Set && request.Color.Value != nil &&
		!validColor(request.Color.Value) {
		return Task{}, ErrInvalidTask
	}
	if !validUpdateReminder(request.DueAt, request.ReminderAt) {
		return Task{}, ErrInvalidTask
	}
	return s.repo.UpdateTask(ctx, userID, taskID, request)
}

func (s *Service) DeleteTask(
	ctx context.Context,
	userID uuid.UUID,
	taskID uuid.UUID,
	version int64,
) error {
	if taskID == uuid.Nil || version < 1 {
		return ErrInvalidTask
	}
	return s.repo.DeleteTask(ctx, userID, taskID, version)
}

func validScope(scope string) bool {
	return scope == "personal" || scope == "couple"
}

func validTitle(title string) bool {
	return utf8.RuneCountInString(title) >= 1 &&
		utf8.RuneCountInString(title) <= 200
}

func validColor(color *string) bool {
	return color == nil || taskColorPattern.MatchString(*color)
}

func validReminder(dueAt, reminderAt *time.Time) bool {
	return reminderAt == nil ||
		(dueAt != nil && !reminderAt.After(*dueAt))
}

func validUpdateReminder(dueAt, reminderAt OptionalTime) bool {
	if !reminderAt.Set || reminderAt.Value == nil {
		return true
	}
	return !dueAt.Set ||
		(dueAt.Value != nil && !reminderAt.Value.After(*dueAt.Value))
}

func validTaskStatus(status string) bool {
	switch status {
	case "open", "in_progress", "done", "cancelled":
		return true
	default:
		return false
	}
}

func validCalendarEventType(eventType string) bool {
	switch eventType {
	case "free", "busy", "date", "meeting", "task":
		return true
	default:
		return false
	}
}
