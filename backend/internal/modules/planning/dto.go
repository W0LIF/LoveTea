package planning

import (
	"bytes"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type Task struct {
	ID          string     `json:"id"`
	Scope       string     `json:"scope"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Status      string     `json:"status"`
	DueAt       *time.Time `json:"due_at"`
	ReminderAt  *time.Time `json:"reminder_at"`
	Color       *string    `json:"color"`
	Version     int64      `json:"version"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type CalendarEvent struct {
	ID          string    `json:"id"`
	Scope       string    `json:"scope"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	EventType   string    `json:"event_type"`
	StartsAt    time.Time `json:"starts_at"`
	EndsAt      time.Time `json:"ends_at"`
	Color       *string   `json:"color"`
	Version     int64     `json:"version"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type CreateCalendarEventRequest struct {
	Scope       string    `json:"scope"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	EventType   string    `json:"event_type"`
	StartsAt    time.Time `json:"starts_at"`
	EndsAt      time.Time `json:"ends_at"`
	Color       *string   `json:"color"`
}

type UpdateCalendarEventRequest struct {
	Version     int64          `json:"version"`
	Title       *string        `json:"title"`
	Description *string        `json:"description"`
	EventType   *string        `json:"event_type"`
	StartsAt    *time.Time     `json:"starts_at"`
	EndsAt      *time.Time     `json:"ends_at"`
	Color       OptionalString `json:"color"`
}

type ImportantDate struct {
	ID            string    `json:"id"`
	Scope         string    `json:"scope"`
	Title         string    `json:"title"`
	Date          string    `json:"date"`
	RepeatsYearly bool      `json:"repeats_yearly"`
	ReminderDays  []int     `json:"reminder_days"`
	Version       int64     `json:"version"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type CreateImportantDateRequest struct {
	Scope         string `json:"scope"`
	Title         string `json:"title"`
	Date          string `json:"date"`
	RepeatsYearly bool   `json:"repeats_yearly"`
	ReminderDays  []int  `json:"reminder_days"`
}

type UpdateImportantDateRequest struct {
	Version       int64   `json:"version"`
	Title         *string `json:"title"`
	Date          *string `json:"date"`
	RepeatsYearly *bool   `json:"repeats_yearly"`
	ReminderDays  *[]int  `json:"reminder_days"`
}

type CreateTaskRequest struct {
	Scope       string     `json:"scope"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	DueAt       *time.Time `json:"due_at"`
	ReminderAt  *time.Time `json:"reminder_at"`
	Color       *string    `json:"color"`
}

type UpdateTaskRequest struct {
	Version     int64          `json:"version"`
	Title       *string        `json:"title"`
	Description *string        `json:"description"`
	Status      *string        `json:"status"`
	DueAt       OptionalTime   `json:"due_at"`
	ReminderAt  OptionalTime   `json:"reminder_at"`
	Color       OptionalString `json:"color"`
}

type OptionalTime struct {
	Value *time.Time
	Set   bool
}

func (o *OptionalTime) UnmarshalJSON(data []byte) error {
	o.Set = true
	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		o.Value = nil
		return nil
	}
	var value time.Time
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	o.Value = &value
	return nil
}

type OptionalString struct {
	Value *string
	Set   bool
}

func (o *OptionalString) UnmarshalJSON(data []byte) error {
	o.Set = true
	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		o.Value = nil
		return nil
	}
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	o.Value = &value
	return nil
}

type taskRecord struct {
	ID          uuid.UUID
	Scope       string
	Title       string
	Description string
	Status      string
	DueAt       *time.Time
	ReminderAt  *time.Time
	Color       *string
	Version     int64
	CreatedAt   time.Time
	UpdatedAt   time.Time
	CoupleID    *uuid.UUID
}
