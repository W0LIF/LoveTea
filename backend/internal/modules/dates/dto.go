package dates

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrNotFound      = errors.New("not found")
	ErrInvalidInput  = errors.New("invalid input")
	ErrNoCouple      = errors.New("no active couple")
	ErrAlreadyExists = errors.New("idea already exists")
)

type DateIdea struct {
	ID             uuid.UUID  `json:"id"`
	CoupleID       uuid.UUID  `json:"couple_id"`
	ProposerUserID uuid.UUID  `json:"proposer_user_id"`
	Title          string     `json:"title"`
	Details        string     `json:"description"`
	Category       string     `json:"category"`
	BudgetMin      *int       `json:"budget_min"`
	BudgetMax      *int       `json:"budget_max"`
	Status         string     `json:"status"`
	CreatedAt      time.Time  `json:"created_at"`
	AcceptedAt     *time.Time `json:"accepted_at"`
	CompletedAt    *time.Time `json:"completed_at"`
}

type DateRecommendation struct {
	ID          uuid.UUID `json:"id"`
	UserID      uuid.UUID `json:"user_id"`
	CoupleID    uuid.UUID `json:"couple_id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Category    string    `json:"category"`
	SuggestedAt time.Time `json:"suggested_at"`
}

type CreateIdeaRequest struct {
	Title     string `json:"title"`
	Details   string `json:"description"`
	Category  string `json:"category"`
	BudgetMin *int   `json:"budget_min"`
	BudgetMax *int   `json:"budget_max"`
}

type UpdateIdeaStatusRequest struct {
	Status string `json:"status"`
}

type SuggestRequest struct {
	Category string `json:"category"`
}
