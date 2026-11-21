package planning

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

const maxPushOperations = 50

var ErrInvalidSyncPush = errors.New("invalid sync push request")

type SyncPushRequest struct {
	Operations []TaskSyncOperation `json:"operations"`
}

type TaskSyncOperation struct {
	OperationID            uuid.UUID                   `json:"operation_id"`
	EntityType             string                      `json:"entity_type"`
	EntityID               uuid.UUID                   `json:"entity_id"`
	Operation              string                      `json:"operation"`
	BaseVersion            int64                       `json:"base_version"`
	Payload                json.RawMessage             `json:"payload,omitempty"`
	TaskPayload            *TaskSyncPayload            `json:"-"`
	CalendarEventPayload   *CalendarEventSyncPayload   `json:"-"`
	ImportantDatePayload   *ImportantDateSyncPayload   `json:"-"`
	HabitPayload           *HabitSyncPayload           `json:"-"`
	HabitCompletionPayload *HabitCompletionSyncPayload `json:"-"`
	MoodPayload            *MoodSyncPayload            `json:"-"`
	SleepPayload           *SleepSyncPayload           `json:"-"`
}

func (operation *TaskSyncOperation) UnmarshalJSON(data []byte) error {
	type operationAlias TaskSyncOperation
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	for _, field := range []string{
		"operation_id", "entity_type", "entity_id", "operation", "base_version",
	} {
		if _, ok := fields[field]; !ok {
			return ErrInvalidSyncPush
		}
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var decoded operationAlias
	if err := decoder.Decode(&decoded); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return errors.New("sync operation must be one JSON object")
	}
	*operation = TaskSyncOperation(decoded)
	return nil
}

type TaskSyncPayload struct {
	Scope       string     `json:"scope"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Status      string     `json:"status"`
	DueAt       *time.Time `json:"due_at"`
	ReminderAt  *time.Time `json:"reminder_at"`
	Color       *string    `json:"color"`
}

func (p *TaskSyncPayload) UnmarshalJSON(data []byte) error {
	type payloadAlias TaskSyncPayload
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	for _, field := range []string{
		"scope", "title", "description", "status",
		"due_at", "reminder_at", "color",
	} {
		if _, ok := fields[field]; !ok {
			return ErrInvalidSyncPush
		}
	}

	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var decoded payloadAlias
	if err := decoder.Decode(&decoded); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return errors.New("task sync payload must be one JSON object")
	}
	*p = TaskSyncPayload(decoded)
	return nil
}

type ImportantDateSyncPayload struct {
	Scope         string `json:"scope"`
	Title         string `json:"title"`
	Date          string `json:"date"`
	RepeatsYearly bool   `json:"repeats_yearly"`
	ReminderDays  []int  `json:"reminder_days"`
}

type HabitSyncPayload struct {
	Scope         string `json:"scope"`
	Title         string `json:"title"`
	Description   string `json:"description"`
	TargetPerWeek int    `json:"target_per_week"`
}

type HabitCompletionSyncPayload struct {
	HabitID     uuid.UUID `json:"habit_id"`
	CompletedOn string    `json:"completed_on"`
}

type MoodSyncPayload struct {
	EntryDate        string `json:"entry_date"`
	Mood             string `json:"mood"`
	Note             string `json:"note"`
	SharedWithCouple bool   `json:"shared_with_couple"`
}

type SleepSyncPayload struct {
	EntryDate        string    `json:"entry_date"`
	SleepStartedAt   time.Time `json:"sleep_started_at"`
	WokeAt           time.Time `json:"woke_at"`
	Quality          int       `json:"quality"`
	SharedWithCouple bool      `json:"shared_with_couple"`
}

type CalendarEventSyncPayload struct {
	Scope       string    `json:"scope"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	EventType   string    `json:"event_type"`
	StartsAt    time.Time `json:"starts_at"`
	EndsAt      time.Time `json:"ends_at"`
	Color       *string   `json:"color"`
}

func (p *MoodSyncPayload) UnmarshalJSON(data []byte) error {
	type payloadAlias MoodSyncPayload
	return decodeStrictSyncPayload(data, []string{
		"entry_date", "mood", "note", "shared_with_couple",
	}, (*payloadAlias)(p))
}

func (p *SleepSyncPayload) UnmarshalJSON(data []byte) error {
	type payloadAlias SleepSyncPayload
	return decodeStrictSyncPayload(data, []string{
		"entry_date", "sleep_started_at", "woke_at", "quality", "shared_with_couple",
	}, (*payloadAlias)(p))
}

func (p *HabitSyncPayload) UnmarshalJSON(data []byte) error {
	type payloadAlias HabitSyncPayload
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	for _, field := range []string{"scope", "title", "description", "target_per_week"} {
		if _, ok := fields[field]; !ok {
			return ErrInvalidSyncPush
		}
	}

	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var decoded payloadAlias
	if err := decoder.Decode(&decoded); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return errors.New("habit sync payload must be one JSON object")
	}
	*p = HabitSyncPayload(decoded)
	return nil
}

func (p *HabitCompletionSyncPayload) UnmarshalJSON(data []byte) error {
	type payloadAlias HabitCompletionSyncPayload
	return decodeStrictSyncPayload(data, []string{"habit_id", "completed_on"}, (*payloadAlias)(p))
}

func decodeStrictSyncPayload(data []byte, requiredFields []string, output any) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	for _, field := range requiredFields {
		if _, ok := fields[field]; !ok {
			return ErrInvalidSyncPush
		}
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(output); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return errors.New("sync payload must be one JSON object")
	}
	return nil
}

func (p *CalendarEventSyncPayload) UnmarshalJSON(data []byte) error {
	type payloadAlias CalendarEventSyncPayload
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	for _, field := range []string{
		"scope", "title", "description", "event_type",
		"starts_at", "ends_at", "color",
	} {
		if _, ok := fields[field]; !ok {
			return ErrInvalidSyncPush
		}
	}

	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var decoded payloadAlias
	if err := decoder.Decode(&decoded); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return errors.New("calendar event sync payload must be one JSON object")
	}
	*p = CalendarEventSyncPayload(decoded)
	return nil
}

func (p *ImportantDateSyncPayload) UnmarshalJSON(data []byte) error {
	type payloadAlias ImportantDateSyncPayload
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	for _, field := range []string{
		"scope", "title", "date", "repeats_yearly", "reminder_days",
	} {
		if _, ok := fields[field]; !ok {
			return ErrInvalidSyncPush
		}
	}

	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var decoded payloadAlias
	if err := decoder.Decode(&decoded); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return errors.New("important date sync payload must be one JSON object")
	}
	*p = ImportantDateSyncPayload(decoded)
	return nil
}

type SyncPushResponse struct {
	Results []TaskSyncResult `json:"results"`
}

type TaskSyncResult struct {
	OperationID     uuid.UUID         `json:"operation_id"`
	Status          string            `json:"status"`
	Replayed        bool              `json:"replayed"`
	Error           string            `json:"error,omitempty"`
	Task            *Task             `json:"task,omitempty"`
	CalendarEvent   *CalendarEvent    `json:"calendar_event,omitempty"`
	ImportantDate   *ImportantDate    `json:"important_date,omitempty"`
	Habit           *SyncedHabit      `json:"habit,omitempty"`
	HabitCompletion *SyncedCompletion `json:"habit_completion,omitempty"`
	MoodEntry       *SyncedMoodEntry  `json:"mood_entry,omitempty"`
	SleepEntry      *SyncedSleepEntry `json:"sleep_entry,omitempty"`
	Version         int64             `json:"version,omitempty"`
}

type SyncedHabit struct {
	ID            string    `json:"id"`
	Scope         string    `json:"scope"`
	Title         string    `json:"title"`
	Description   string    `json:"description"`
	TargetPerWeek int       `json:"target_per_week"`
	Version       int64     `json:"version"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type SyncedCompletion struct {
	ID          string    `json:"id"`
	HabitID     string    `json:"habit_id"`
	UserID      string    `json:"user_id"`
	CompletedOn string    `json:"completed_on"`
	IsMine      bool      `json:"is_mine"`
	CreatedAt   time.Time `json:"created_at"`
}

type SyncedMoodEntry struct {
	ID               string    `json:"id"`
	EntryDate        string    `json:"entry_date"`
	Mood             string    `json:"mood"`
	Note             string    `json:"note"`
	SharedWithCouple bool      `json:"shared_with_couple"`
	IsMine           bool      `json:"is_mine"`
	Version          int64     `json:"version"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type SyncedSleepEntry struct {
	ID               string    `json:"id"`
	EntryDate        string    `json:"entry_date"`
	SleepStartedAt   time.Time `json:"sleep_started_at"`
	WokeAt           time.Time `json:"woke_at"`
	Quality          int       `json:"quality"`
	SharedWithCouple bool      `json:"shared_with_couple"`
	IsMine           bool      `json:"is_mine"`
	Version          int64     `json:"version"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

func validateSyncPush(request *SyncPushRequest) error {
	if request == nil || len(request.Operations) == 0 ||
		len(request.Operations) > maxPushOperations {
		return ErrInvalidSyncPush
	}

	for index := range request.Operations {
		operation := &request.Operations[index]
		if operation.OperationID == uuid.Nil ||
			(operation.EntityType != "task" && operation.EntityType != "calendar_event" &&
				operation.EntityType != "important_date" && operation.EntityType != "habit" &&
				operation.EntityType != "habit_completion" &&
				operation.EntityType != "mood_entry" && operation.EntityType != "sleep_entry") ||
			operation.EntityID == uuid.Nil {
			return ErrInvalidSyncPush
		}

		switch operation.Operation {
		case "upsert":
			if len(operation.Payload) == 0 || operation.BaseVersion < 0 {
				return ErrInvalidSyncPush
			}
			if operation.EntityType == "task" {
				var payload TaskSyncPayload
				if err := json.Unmarshal(operation.Payload, &payload); err != nil {
					return ErrInvalidSyncPush
				}
				payload.Title = strings.TrimSpace(payload.Title)
				payload.Description = strings.TrimSpace(payload.Description)
				if !validScope(payload.Scope) || !validTitle(payload.Title) ||
					utf8.RuneCountInString(payload.Description) > 5000 ||
					!validTaskStatus(payload.Status) ||
					!validColor(payload.Color) ||
					!validReminder(payload.DueAt, payload.ReminderAt) {
					return ErrInvalidSyncPush
				}
				if payload.DueAt != nil {
					dueAt := payload.DueAt.UTC()
					payload.DueAt = &dueAt
				}
				if payload.ReminderAt != nil {
					reminderAt := payload.ReminderAt.UTC()
					payload.ReminderAt = &reminderAt
				}
				operation.TaskPayload = &payload
				normalized, err := json.Marshal(payload)
				if err != nil {
					return ErrInvalidSyncPush
				}
				operation.Payload = normalized
				continue
			}

			if operation.EntityType == "calendar_event" {
				var payload CalendarEventSyncPayload
				if err := json.Unmarshal(operation.Payload, &payload); err != nil {
					return ErrInvalidSyncPush
				}
				payload.Title = strings.TrimSpace(payload.Title)
				payload.Description = strings.TrimSpace(payload.Description)
				if !validScope(payload.Scope) || !validTitle(payload.Title) ||
					utf8.RuneCountInString(payload.Description) > 5000 ||
					!validCalendarEventType(payload.EventType) ||
					payload.StartsAt.IsZero() || payload.EndsAt.IsZero() ||
					!payload.EndsAt.After(payload.StartsAt) || !validColor(payload.Color) {
					return ErrInvalidSyncPush
				}
				payload.StartsAt = payload.StartsAt.UTC()
				payload.EndsAt = payload.EndsAt.UTC()
				operation.CalendarEventPayload = &payload
				normalized, err := json.Marshal(payload)
				if err != nil {
					return ErrInvalidSyncPush
				}
				operation.Payload = normalized
				continue
			}

			if operation.EntityType == "habit" {
				var payload HabitSyncPayload
				if err := json.Unmarshal(operation.Payload, &payload); err != nil {
					return ErrInvalidSyncPush
				}
				payload.Title = strings.TrimSpace(payload.Title)
				payload.Description = strings.TrimSpace(payload.Description)
				if !validScope(payload.Scope) || !validTitle(payload.Title) ||
					utf8.RuneCountInString(payload.Description) > 2000 ||
					payload.TargetPerWeek < 1 || payload.TargetPerWeek > 7 {
					return ErrInvalidSyncPush
				}
				operation.HabitPayload = &payload
				normalized, err := json.Marshal(payload)
				if err != nil {
					return ErrInvalidSyncPush
				}
				operation.Payload = normalized
				continue
			}

			if operation.EntityType == "mood_entry" {
				var payload MoodSyncPayload
				if err := json.Unmarshal(operation.Payload, &payload); err != nil {
					return ErrInvalidSyncPush
				}
				payload.Mood = strings.TrimSpace(payload.Mood)
				payload.Note = strings.TrimSpace(payload.Note)
				entryDate, err := time.Parse("2006-01-02", payload.EntryDate)
				if err != nil || entryDate.Format("2006-01-02") != payload.EntryDate ||
					utf8.RuneCountInString(payload.Mood) < 1 ||
					utf8.RuneCountInString(payload.Mood) > 32 ||
					utf8.RuneCountInString(payload.Note) > 2000 {
					return ErrInvalidSyncPush
				}
				operation.MoodPayload = &payload
				normalized, err := json.Marshal(payload)
				if err != nil {
					return ErrInvalidSyncPush
				}
				operation.Payload = normalized
				continue
			}

			if operation.EntityType == "habit_completion" {
				if operation.BaseVersion != 0 {
					return ErrInvalidSyncPush
				}
				if err := validateHabitCompletionSyncPayload(operation); err != nil {
					return err
				}
				continue
			}

			if operation.EntityType == "sleep_entry" {
				var payload SleepSyncPayload
				if err := json.Unmarshal(operation.Payload, &payload); err != nil {
					return ErrInvalidSyncPush
				}
				entryDate, err := time.Parse("2006-01-02", payload.EntryDate)
				if err != nil || entryDate.Format("2006-01-02") != payload.EntryDate ||
					payload.SleepStartedAt.IsZero() || payload.WokeAt.IsZero() ||
					!payload.WokeAt.After(payload.SleepStartedAt) ||
					payload.WokeAt.Sub(payload.SleepStartedAt) > 24*time.Hour ||
					payload.Quality < 1 || payload.Quality > 5 {
					return ErrInvalidSyncPush
				}
				payload.SleepStartedAt = payload.SleepStartedAt.UTC()
				payload.WokeAt = payload.WokeAt.UTC()
				operation.SleepPayload = &payload
				normalized, err := json.Marshal(payload)
				if err != nil {
					return ErrInvalidSyncPush
				}
				operation.Payload = normalized
				continue
			}

			var payload ImportantDateSyncPayload
			if err := json.Unmarshal(operation.Payload, &payload); err != nil {
				return ErrInvalidSyncPush
			}
			payload.Title = strings.TrimSpace(payload.Title)
			eventDate, err := time.Parse("2006-01-02", payload.Date)
			if err != nil || eventDate.Format("2006-01-02") != payload.Date ||
				!validScope(payload.Scope) || !validTitle(payload.Title) ||
				payload.ReminderDays == nil || !validReminderDays(payload.ReminderDays) {
				return ErrInvalidSyncPush
			}
			operation.ImportantDatePayload = &payload
			normalized, err := json.Marshal(payload)
			if err != nil {
				return ErrInvalidSyncPush
			}
			operation.Payload = normalized
		case "delete":
			if operation.EntityType == "habit_completion" {
				if operation.BaseVersion != 0 {
					return ErrInvalidSyncPush
				}
				if err := validateHabitCompletionSyncPayload(operation); err != nil {
					return err
				}
				continue
			}
			if len(operation.Payload) != 0 || operation.BaseVersion < 1 ||
				operation.EntityType == "mood_entry" || operation.EntityType == "sleep_entry" {
				return ErrInvalidSyncPush
			}
		default:
			return ErrInvalidSyncPush
		}
	}

	return nil
}

func validateHabitCompletionSyncPayload(operation *TaskSyncOperation) error {
	if len(operation.Payload) == 0 {
		return ErrInvalidSyncPush
	}
	var payload HabitCompletionSyncPayload
	if err := json.Unmarshal(operation.Payload, &payload); err != nil {
		return ErrInvalidSyncPush
	}
	if payload.HabitID == uuid.Nil {
		return ErrInvalidSyncPush
	}
	completedOn, err := time.Parse("2006-01-02", payload.CompletedOn)
	if err != nil || completedOn.Format("2006-01-02") != payload.CompletedOn {
		return ErrInvalidSyncPush
	}
	operation.HabitCompletionPayload = &payload
	normalized, err := json.Marshal(payload)
	if err != nil {
		return ErrInvalidSyncPush
	}
	operation.Payload = normalized
	return nil
}

func syncOperationHash(operation TaskSyncOperation) ([]byte, error) {
	encoded, err := json.Marshal(operation)
	if err != nil {
		return nil, err
	}
	hash := sha256.Sum256(encoded)
	return hash[:], nil
}

func (s *Service) PushSyncOperations(
	ctx context.Context,
	userID uuid.UUID,
	request SyncPushRequest,
) (SyncPushResponse, error) {
	if userID == uuid.Nil || validateSyncPush(&request) != nil {
		return SyncPushResponse{}, ErrInvalidSyncPush
	}
	return s.repo.PushSyncOperations(ctx, userID, request.Operations)
}

func validReminderDays(days []int) bool {
	if len(days) > 10 {
		return false
	}
	seen := make(map[int]struct{}, len(days))
	for _, day := range days {
		if day < 0 || day > 365 {
			return false
		}
		if _, exists := seen[day]; exists {
			return false
		}
		seen[day] = struct{}{}
	}
	return true
}
