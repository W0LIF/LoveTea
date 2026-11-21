package account

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrNotFound       = errors.New("export not found")
	ErrNoCouple       = errors.New("no active couple")
	ErrInvalidConfirm = errors.New("confirmation phrase does not match")
)

type Export struct {
	ID          uuid.UUID  `json:"id"`
	UserID      uuid.UUID  `json:"user_id"`
	Status      string     `json:"status"`
	FileURL     *string    `json:"file_url,omitempty"`
	ExpiresAt   time.Time  `json:"expires_at"`
	CreatedAt   time.Time  `json:"created_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}

type DeleteAccountRequest struct {
	Confirm string `json:"confirm"`
}
