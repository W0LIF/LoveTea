package planning

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestValidateSyncPush(t *testing.T) {
	taskID := uuid.New()
	operationID := uuid.New()
	dueAt := time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC)

	valid := SyncPushRequest{Operations: []TaskSyncOperation{{
		OperationID: operationID,
		EntityType:  "task",
		EntityID:    taskID,
		Operation:   "upsert",
		BaseVersion: 0,
		Payload: encodeSyncPayload(t, TaskSyncPayload{
			Scope:  "personal",
			Title:  "  Walk  ",
			Status: "open",
			DueAt:  &dueAt,
		}),
	}}}
	if err := validateSyncPush(&valid); err != nil {
		t.Fatalf("validateSyncPush() error = %v", err)
	}
	if valid.Operations[0].TaskPayload.Title != "Walk" {
		t.Fatalf("title was not normalized: %q", valid.Operations[0].TaskPayload.Title)
	}

	lateReminder := dueAt.Add(time.Minute)
	tests := []struct {
		name    string
		request SyncPushRequest
	}{
		{name: "empty batch", request: SyncPushRequest{}},
		{
			name: "unsupported entity",
			request: SyncPushRequest{Operations: []TaskSyncOperation{{
				OperationID: operationID, EntityType: "mood_entry", EntityID: taskID,
				Operation: "upsert", Payload: valid.Operations[0].Payload,
			}}},
		},
		{
			name: "negative base version",
			request: SyncPushRequest{Operations: []TaskSyncOperation{{
				OperationID: operationID, EntityType: "task", EntityID: taskID,
				Operation: "upsert", BaseVersion: -1, Payload: valid.Operations[0].Payload,
			}}},
		},
		{
			name: "oversized batch",
			request: func() SyncPushRequest {
				operations := make([]TaskSyncOperation, maxPushOperations+1)
				for index := range operations {
					operations[index] = valid.Operations[0]
				}
				return SyncPushRequest{Operations: operations}
			}(),
		},
		{
			name: "delete with payload",
			request: SyncPushRequest{Operations: []TaskSyncOperation{{
				OperationID: operationID, EntityType: "task", EntityID: taskID,
				Operation: "delete", BaseVersion: 1, Payload: valid.Operations[0].Payload,
			}}},
		},
		{
			name: "reminder after due time",
			request: SyncPushRequest{Operations: []TaskSyncOperation{{
				OperationID: operationID, EntityType: "task", EntityID: taskID,
				Operation: "upsert", Payload: encodeSyncPayload(t, TaskSyncPayload{
					Scope: "personal", Title: "Walk", Status: "open",
					DueAt: &dueAt, ReminderAt: &lateReminder,
				}),
			}}},
		},
		{
			name: "duplicate important date reminder",
			request: SyncPushRequest{Operations: []TaskSyncOperation{{
				OperationID: operationID, EntityType: "important_date", EntityID: taskID,
				Operation: "upsert", Payload: encodeSyncPayload(t, ImportantDateSyncPayload{
					Scope: "personal", Title: "Anniversary", Date: "2026-11-20",
					ReminderDays: []int{1, 1},
				}),
			}}},
		},
		{
			name: "invalid important date",
			request: SyncPushRequest{Operations: []TaskSyncOperation{{
				OperationID: operationID, EntityType: "important_date", EntityID: taskID,
				Operation: "upsert", Payload: encodeSyncPayload(t, ImportantDateSyncPayload{
					Scope: "personal", Title: "Anniversary", Date: "2026-02-30",
				}),
			}}},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := validateSyncPush(&test.request); !errors.Is(err, ErrInvalidSyncPush) {
				t.Fatalf("validateSyncPush() error = %v, want invalid request", err)
			}
		})
	}
}

func TestValidateSyncPushImportantDate(t *testing.T) {
	request := SyncPushRequest{Operations: []TaskSyncOperation{{
		OperationID: uuid.New(),
		EntityType:  "important_date",
		EntityID:    uuid.New(),
		Operation:   "upsert",
		Payload: encodeSyncPayload(t, ImportantDateSyncPayload{
			Scope: "couple", Title: "  Anniversary  ", Date: "2026-11-20",
			RepeatsYearly: true, ReminderDays: []int{30, 7, 1},
		}),
	}}}
	if err := validateSyncPush(&request); err != nil {
		t.Fatalf("validateSyncPush() error = %v", err)
	}
	payload := request.Operations[0].ImportantDatePayload
	if payload.Title != "Anniversary" || !payload.RepeatsYearly ||
		len(payload.ReminderDays) != 3 {
		t.Fatalf("important date payload was not normalized: %+v", payload)
	}
}

func TestValidateSyncPushCalendarEvent(t *testing.T) {
	location := time.FixedZone("test", 2*60*60)
	startsAt := time.Date(2026, 11, 20, 12, 0, 0, 0, location)
	endsAt := startsAt.Add(time.Hour)
	request := SyncPushRequest{Operations: []TaskSyncOperation{{
		OperationID: uuid.New(),
		EntityType:  "calendar_event",
		EntityID:    uuid.New(),
		Operation:   "upsert",
		Payload: encodeSyncPayload(t, CalendarEventSyncPayload{
			Scope: "personal", Title: "  Coffee  ", Description: "  Meet up  ",
			EventType: "date", StartsAt: startsAt, EndsAt: endsAt,
		}),
	}}}
	if err := validateSyncPush(&request); err != nil {
		t.Fatalf("validateSyncPush() error = %v", err)
	}
	payload := request.Operations[0].CalendarEventPayload
	if payload.Title != "Coffee" || payload.Description != "Meet up" ||
		payload.StartsAt.Location() != time.UTC || payload.StartsAt.Hour() != 10 {
		t.Fatalf("calendar event was not normalized: %+v", payload)
	}
}

func TestValidateSyncPushHabit(t *testing.T) {
	request := SyncPushRequest{Operations: []TaskSyncOperation{{
		OperationID: uuid.New(),
		EntityType:  "habit",
		EntityID:    uuid.New(),
		Operation:   "upsert",
		Payload: encodeSyncPayload(t, HabitSyncPayload{
			Scope: "personal", Title: "  Walk  ", Description: "  Daily walk  ",
			TargetPerWeek: 5,
		}),
	}}}
	if err := validateSyncPush(&request); err != nil {
		t.Fatalf("validateSyncPush() error = %v", err)
	}
	payload := request.Operations[0].HabitPayload
	if payload.Title != "Walk" || payload.Description != "Daily walk" ||
		payload.TargetPerWeek != 5 {
		t.Fatalf("habit payload was not normalized: %+v", payload)
	}

	for _, invalid := range []HabitSyncPayload{
		{Scope: "personal", Title: "Walk", Description: "", TargetPerWeek: 0},
		{Scope: "personal", Title: "Walk", Description: "", TargetPerWeek: 8},
		{
			Scope: "personal", Title: "Walk", Description: strings.Repeat("x", 2001),
			TargetPerWeek: 5,
		},
	} {
		invalidRequest := SyncPushRequest{Operations: []TaskSyncOperation{{
			OperationID: uuid.New(),
			EntityType:  "habit",
			EntityID:    uuid.New(),
			Operation:   "upsert",
			Payload:     encodeSyncPayload(t, invalid),
		}}}
		if validateSyncPush(&invalidRequest) == nil {
			t.Fatalf("invalid habit payload was accepted: %+v", invalid)
		}
	}
	missingField := SyncPushRequest{Operations: []TaskSyncOperation{{
		OperationID: uuid.New(),
		EntityType:  "habit",
		EntityID:    uuid.New(),
		Operation:   "upsert",
		Payload: json.RawMessage(
			`{"scope":"personal","title":"Walk","target_per_week":5}`,
		),
	}}}
	if validateSyncPush(&missingField) == nil {
		t.Fatal("habit payload with a missing required field was accepted")
	}
}

func TestValidateSyncPushWellnessEntries(t *testing.T) {
	startedAt := time.Date(2026, 11, 19, 22, 0, 0, 0, time.FixedZone("test", 2*60*60))
	wokeAt := startedAt.Add(8 * time.Hour)
	request := SyncPushRequest{Operations: []TaskSyncOperation{
		{
			OperationID: uuid.New(),
			EntityType:  "mood_entry",
			EntityID:    uuid.New(),
			Operation:   "upsert",
			Payload: encodeSyncPayload(t, MoodSyncPayload{
				EntryDate: "2026-11-19", Mood: "  good  ", Note: "  private note  ",
			}),
		},
		{
			OperationID: uuid.New(),
			EntityType:  "sleep_entry",
			EntityID:    uuid.New(),
			Operation:   "upsert",
			Payload: encodeSyncPayload(t, SleepSyncPayload{
				EntryDate: "2026-11-19", SleepStartedAt: startedAt, WokeAt: wokeAt,
				Quality: 4, SharedWithCouple: true,
			}),
		},
	}}
	if err := validateSyncPush(&request); err != nil {
		t.Fatalf("validateSyncPush() error = %v", err)
	}
	if request.Operations[0].MoodPayload.Mood != "good" ||
		request.Operations[0].MoodPayload.Note != "private note" {
		t.Fatalf("mood payload was not normalized: %+v", request.Operations[0].MoodPayload)
	}
	if request.Operations[1].SleepPayload.SleepStartedAt.Location() != time.UTC {
		t.Fatalf("sleep times were not normalized: %+v", request.Operations[1].SleepPayload)
	}

	for _, invalid := range []TaskSyncOperation{
		{
			OperationID: uuid.New(), EntityType: "mood_entry",
			EntityID: uuid.New(), Operation: "upsert",
			Payload: encodeSyncPayload(t, MoodSyncPayload{
				EntryDate: "2026-02-30", Mood: "good",
			}),
		},
		{
			OperationID: uuid.New(), EntityType: "mood_entry",
			EntityID: uuid.New(), Operation: "delete", BaseVersion: 1,
		},
		{
			OperationID: uuid.New(), EntityType: "sleep_entry",
			EntityID: uuid.New(), Operation: "upsert",
			Payload: encodeSyncPayload(t, SleepSyncPayload{
				EntryDate: "2026-11-19", SleepStartedAt: wokeAt,
				WokeAt: startedAt, Quality: 4,
			}),
		},
	} {
		invalidRequest := SyncPushRequest{Operations: []TaskSyncOperation{invalid}}
		if validateSyncPush(&invalidRequest) == nil {
			t.Fatalf("invalid wellness operation was accepted: %+v", invalid)
		}
	}
}

func TestValidateSyncPushHabitCompletions(t *testing.T) {
	payload := encodeSyncPayload(t, HabitCompletionSyncPayload{
		HabitID: uuid.New(), CompletedOn: "2026-11-19",
	})
	for _, action := range []string{"upsert", "delete"} {
		request := SyncPushRequest{Operations: []TaskSyncOperation{{
			OperationID: uuid.New(),
			EntityType:  "habit_completion",
			EntityID:    uuid.New(),
			Operation:   action,
			Payload:     payload,
		}}}
		if err := validateSyncPush(&request); err != nil {
			t.Errorf("validateSyncPush(%s) error = %v", action, err)
		}
	}

	for _, operation := range []TaskSyncOperation{
		{
			OperationID: uuid.New(), EntityType: "habit_completion",
			EntityID: uuid.New(), Operation: "upsert", BaseVersion: 1,
			Payload: payload,
		},
		{
			OperationID: uuid.New(), EntityType: "habit_completion",
			EntityID: uuid.New(), Operation: "delete",
			Payload: encodeSyncPayload(t, HabitCompletionSyncPayload{
				HabitID: uuid.New(), CompletedOn: "2026-02-30",
			}),
		},
		{
			OperationID: uuid.New(), EntityType: "habit_completion",
			EntityID: uuid.New(), Operation: "upsert",
			Payload: json.RawMessage(`{"habit_id":"00000000-0000-0000-0000-000000000001"}`),
		},
	} {
		request := SyncPushRequest{Operations: []TaskSyncOperation{operation}}
		if validateSyncPush(&request) == nil {
			t.Errorf("invalid habit completion operation was accepted: %+v", operation)
		}
	}
}

func TestTaskSyncPayloadRequiresCompleteKnownShape(t *testing.T) {
	for _, test := range []struct {
		name string
		body string
	}{
		{
			name: "missing nullable field",
			body: `{"scope":"personal","title":"Walk","description":"","status":"open","due_at":null,"reminder_at":null}`,
		},
		{
			name: "unknown field",
			body: `{"scope":"personal","title":"Walk","description":"","status":"open","due_at":null,"reminder_at":null,"color":null,"owner_user_id":"other"}`,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			var payload TaskSyncPayload
			if err := json.Unmarshal([]byte(test.body), &payload); err == nil {
				t.Fatal("JSON with an incomplete or unknown task payload was accepted")
			}
		})
	}
}

func TestHabitSyncPayloadRejectsUnknownFields(t *testing.T) {
	request := SyncPushRequest{Operations: []TaskSyncOperation{{
		OperationID: uuid.New(),
		EntityType:  "habit",
		EntityID:    uuid.New(),
		Operation:   "upsert",
		Payload: json.RawMessage(
			`{"scope":"personal","title":"Walk","description":"","target_per_week":5,"owner_user_id":"other"}`,
		),
	}}}
	if validateSyncPush(&request) == nil {
		t.Fatal("habit payload with an unknown field was accepted")
	}
}

func TestImportantDateSyncPayloadRequiresCompleteKnownShape(t *testing.T) {
	for _, body := range []string{
		`{"scope":"personal","title":"Anniversary","date":"2026-11-20","repeats_yearly":false}`,
		`{"scope":"personal","title":"Anniversary","date":"2026-11-20","repeats_yearly":false,"reminder_days":[],"owner_user_id":"other"}`,
		`{"scope":"personal","title":"Anniversary","date":"2026-11-20","repeats_yearly":false,"reminder_days":null}`,
	} {
		request := SyncPushRequest{Operations: []TaskSyncOperation{{
			OperationID: uuid.New(),
			EntityType:  "important_date",
			EntityID:    uuid.New(),
			Operation:   "upsert",
			Payload:     json.RawMessage(body),
		}}}
		if validateSyncPush(&request) == nil {
			t.Fatalf("invalid important date payload was accepted: %s", body)
		}
	}
}

func TestSyncOperationRequiresVersionAndKnownFields(t *testing.T) {
	base := `"operation_id":"00000000-0000-0000-0000-000000000001","entity_type":"task","entity_id":"00000000-0000-0000-0000-000000000002","operation":"upsert"`
	for _, body := range []string{
		`{` + base + `,"payload":{}}`,
		`{` + base + `,"base_version":0,"unexpected":true}`,
	} {
		var operation TaskSyncOperation
		if err := json.Unmarshal([]byte(body), &operation); err == nil {
			t.Fatalf("invalid sync operation envelope was accepted: %s", body)
		}
	}
}

func TestCalendarEventSyncPayloadRequiresCompleteKnownShape(t *testing.T) {
	for _, body := range []string{
		`{"scope":"personal","title":"Coffee","description":"","event_type":"date","starts_at":"2026-11-20T12:00:00Z","ends_at":"2026-11-20T13:00:00Z"}`,
		`{"scope":"personal","title":"Coffee","description":"","event_type":"date","starts_at":"2026-11-20T12:00:00Z","ends_at":"2026-11-20T13:00:00Z","color":null,"owner_user_id":"other"}`,
	} {
		request := SyncPushRequest{Operations: []TaskSyncOperation{{
			OperationID: uuid.New(),
			EntityType:  "calendar_event",
			EntityID:    uuid.New(),
			Operation:   "upsert",
			Payload:     json.RawMessage(body),
		}}}
		if validateSyncPush(&request) == nil {
			t.Fatalf("invalid calendar-event payload was accepted: %s", body)
		}
	}
}

func encodeSyncPayload(t *testing.T, payload any) json.RawMessage {
	t.Helper()
	encoded, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	return encoded
}
