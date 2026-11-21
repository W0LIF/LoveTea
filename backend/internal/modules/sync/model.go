package sync

import (
	"encoding/json"
	"errors"
	"regexp"
	"strconv"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

var (
	ErrInvalidCursor = errors.New("invalid sync cursor")
	ErrInvalidLimit  = errors.New("invalid sync limit")
	ErrInvalidChange = errors.New("invalid sync change")

	entityTypePattern = regexp.MustCompile(`^[a-z][a-z0-9_]{0,79}$`)
)

type Change struct {
	Cursor     int64           `json:"cursor"`
	EntityType string          `json:"entity_type"`
	EntityID   uuid.UUID       `json:"entity_id"`
	Version    int64           `json:"version"`
	Operation  string          `json:"operation"`
	Payload    json.RawMessage `json:"payload"`
	CreatedAt  time.Time       `json:"created_at"`
}

type Page struct {
	Changes    []Change `json:"changes"`
	NextCursor int64    `json:"next_cursor"`
	HasMore    bool     `json:"has_more"`
}

type Scope struct {
	UserID   *uuid.UUID
	CoupleID *uuid.UUID
}

type NewChange struct {
	Scope      Scope
	ActorID    uuid.UUID
	EntityType string
	EntityID   uuid.UUID
	Operation  string
	Payload    json.RawMessage
}

func validatePage(cursor int64, limit int) error {
	if cursor < 0 {
		return ErrInvalidCursor
	}
	if limit < 1 || limit > 100 {
		return ErrInvalidLimit
	}
	return nil
}

func validateChange(change NewChange) error {
	if (change.Scope.UserID == nil) == (change.Scope.CoupleID == nil) ||
		change.ActorID == uuid.Nil ||
		change.EntityID == uuid.Nil ||
		!entityTypePattern.MatchString(change.EntityType) ||
		(change.Operation != "upsert" && change.Operation != "delete") {
		return ErrInvalidChange
	}
	if change.Scope.UserID != nil && *change.Scope.UserID == uuid.Nil ||
		change.Scope.CoupleID != nil && *change.Scope.CoupleID == uuid.Nil {
		return ErrInvalidChange
	}
	if change.Scope.UserID != nil && *change.Scope.UserID != change.ActorID {
		return ErrInvalidChange
	}
	if len(change.Payload) > 64*1024 ||
		(len(change.Payload) > 0 && !json.Valid(change.Payload)) {
		return ErrInvalidChange
	}
	if len(change.Payload) > 0 {
		var payload any
		if err := json.Unmarshal(change.Payload, &payload); err != nil {
			return ErrInvalidChange
		}
		switch payload.(type) {
		case map[string]any:
		default:
			return ErrInvalidChange
		}
	}
	return nil
}

func validCursorText(value string) bool {
	if value == "" || utf8.RuneCountInString(value) > 19 {
		return false
	}
	for _, char := range value {
		if char < '0' || char > '9' {
			return false
		}
	}
	_, err := strconv.ParseInt(value, 10, 64)
	return err == nil
}
