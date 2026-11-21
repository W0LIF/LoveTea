package wellness

import (
	"time"

	"github.com/google/uuid"
)

type MoodEntry struct {
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

type SaveMoodRequest struct {
	EntryDate        string `json:"entry_date"`
	Mood             string `json:"mood"`
	Note             string `json:"note"`
	SharedWithCouple bool   `json:"shared_with_couple"`
	Version          *int64 `json:"version"`
}

type SleepEntry struct {
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

type SaveSleepRequest struct {
	EntryDate        string    `json:"entry_date"`
	SleepStartedAt   time.Time `json:"sleep_started_at"`
	WokeAt           time.Time `json:"woke_at"`
	Quality          int       `json:"quality"`
	SharedWithCouple bool      `json:"shared_with_couple"`
	Version          *int64    `json:"version"`
}

type syncPayloadMood struct {
	EntryDate string `json:"entry_date"`
	Mood      string `json:"mood"`
	Note      string `json:"note,omitempty"`
}

type syncPayloadSleep struct {
	EntryDate      string    `json:"entry_date"`
	SleepStartedAt time.Time `json:"sleep_started_at"`
	WokeAt         time.Time `json:"woke_at"`
	Quality        int       `json:"quality"`
}

type dateRange struct {
	From time.Time
	To   time.Time
}

func parseDateRange(from, to string) (dateRange, error) {
	now := time.Now().UTC()
	defaultTo := now.Truncate(24 * time.Hour)
	defaultFrom := defaultTo.AddDate(0, 0, -29)
	start, end := defaultFrom, defaultTo

	if from != "" {
		parsed, err := time.Parse("2006-01-02", from)
		if err != nil || parsed.Format("2006-01-02") != from {
			return dateRange{}, ErrInvalidDateRange
		}
		start = parsed
	}
	if to != "" {
		parsed, err := time.Parse("2006-01-02", to)
		if err != nil || parsed.Format("2006-01-02") != to {
			return dateRange{}, ErrInvalidDateRange
		}
		end = parsed
	}
	if start.After(end) || end.Sub(start) > 365*24*time.Hour {
		return dateRange{}, ErrInvalidDateRange
	}
	return dateRange{From: start, To: end}, nil
}

func parseEntityID(value string) (uuid.UUID, error) {
	id, err := uuid.Parse(value)
	if err != nil || id == uuid.Nil {
		return uuid.Nil, ErrInvalidEntry
	}
	return id, nil
}
