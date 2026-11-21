package sync

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"
)

func TestValidatePage(t *testing.T) {
	tests := []struct {
		name   string
		cursor int64
		limit  int
		want   error
	}{
		{name: "valid", cursor: 15, limit: 100},
		{name: "negative cursor", cursor: -1, limit: 10, want: ErrInvalidCursor},
		{name: "zero limit", cursor: 0, limit: 0, want: ErrInvalidLimit},
		{name: "limit too large", cursor: 0, limit: 101, want: ErrInvalidLimit},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := validatePage(test.cursor, test.limit); !errors.Is(err, test.want) {
				t.Fatalf("validatePage() error = %v, want %v", err, test.want)
			}
		})
	}
}

func TestValidCursorText(t *testing.T) {
	for _, test := range []struct {
		value string
		want  bool
	}{
		{value: "0", want: true},
		{value: "9223372036854775807", want: true},
		{value: "", want: false},
		{value: "-1", want: false},
		{value: "1.5", want: false},
		{value: "9223372036854775808", want: false},
	} {
		if got := validCursorText(test.value); got != test.want {
			t.Errorf("validCursorText(%q) = %t, want %t", test.value, got, test.want)
		}
	}
}

func TestValidateChangeEnforcesScopeAndPayload(t *testing.T) {
	actorID := uuid.New()
	coupleID := uuid.New()
	entityID := uuid.New()
	otherUserID := uuid.New()
	tests := []struct {
		name   string
		change NewChange
		want   error
	}{
		{
			name: "valid couple change",
			change: NewChange{
				Scope:      Scope{CoupleID: &coupleID},
				ActorID:    actorID,
				EntityType: "calendar_event",
				EntityID:   entityID,
				Operation:  "upsert",
				Payload:    json.RawMessage(`{"title":"date"}`),
			},
		},
		{
			name: "user change cannot target another user",
			change: NewChange{
				Scope:      Scope{UserID: &otherUserID},
				ActorID:    actorID,
				EntityType: "mood_entry",
				EntityID:   entityID,
				Operation:  "upsert",
			},
			want: ErrInvalidChange,
		},
		{
			name: "scope must be exactly one",
			change: NewChange{
				Scope:      Scope{UserID: &actorID, CoupleID: &coupleID},
				ActorID:    actorID,
				EntityType: "mood_entry",
				EntityID:   entityID,
				Operation:  "upsert",
			},
			want: ErrInvalidChange,
		},
		{
			name: "payload must be an object",
			change: NewChange{
				Scope:      Scope{UserID: &actorID},
				ActorID:    actorID,
				EntityType: "mood_entry",
				EntityID:   entityID,
				Operation:  "upsert",
				Payload:    json.RawMessage(`"private"`),
			},
			want: ErrInvalidChange,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := validateChange(test.change); !errors.Is(err, test.want) {
				t.Fatalf("validateChange() error = %v, want %v", err, test.want)
			}
		})
	}
}
