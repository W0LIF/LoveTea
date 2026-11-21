package home

import (
	"encoding/json"
	"time"

	"github.com/W0LIF/LoveTea/backend/internal/modules/couples"
	"github.com/W0LIF/LoveTea/backend/internal/modules/users"
	"github.com/google/uuid"
)

type Response struct {
	Profile *users.ProfileResponse    `json:"profile"`
	Couple  *couples.MyCoupleResponse `json:"couple"`
	Pet     *PetState                 `json:"pet"`
}

type PetState struct {
	CoupleID   uuid.UUID       `json:"couple_id"`
	Appearance json.RawMessage `json:"appearance"`
	Version    int64           `json:"version"`
	UpdatedAt  time.Time       `json:"updated_at"`
}

type petRecord struct {
	Appearance json.RawMessage
	Version    int64
	UpdatedAt  time.Time
}
