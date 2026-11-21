package habits

import "time"

type Habit struct {
	ID            string    `json:"id"`
	Scope         string    `json:"scope"`
	Title         string    `json:"title"`
	Description   string    `json:"description"`
	TargetPerWeek int       `json:"target_per_week"`
	Version       int64     `json:"version"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type CreateHabitRequest struct {
	Scope         string `json:"scope"`
	Title         string `json:"title"`
	Description   string `json:"description"`
	TargetPerWeek *int   `json:"target_per_week"`
}

type UpdateHabitRequest struct {
	Version       int64   `json:"version"`
	Title         *string `json:"title"`
	Description   *string `json:"description"`
	TargetPerWeek *int    `json:"target_per_week"`
}

type Completion struct {
	ID          string    `json:"id"`
	HabitID     string    `json:"habit_id"`
	UserID      string    `json:"user_id"`
	CompletedOn string    `json:"completed_on"`
	IsMine      bool      `json:"is_mine"`
	CreatedAt   time.Time `json:"created_at"`
}

type CompleteHabitRequest struct {
	CompletedOn string `json:"completed_on"`
}

type HabitMemberStats struct {
	UserID             string `json:"user_id"`
	CompletionCount    int    `json:"completion_count"`
	SuccessfulWeeks    int    `json:"successful_weeks"`
	CurrentStreakWeeks int    `json:"current_streak_weeks"`
}

type HabitStats struct {
	HabitID          string             `json:"habit_id"`
	From             string             `json:"from"`
	To               string             `json:"to"`
	TargetPerWeek    int                `json:"target_per_week"`
	TotalCompletions int                `json:"total_completions"`
	Members          []HabitMemberStats `json:"members"`
}

type dateRange struct {
	From time.Time
	To   time.Time
}

type completionRecord struct {
	UserID string
	Date   time.Time
}
