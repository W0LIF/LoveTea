package pet

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type PetState struct {
	CoupleID   uuid.UUID       `json:"couple_id"`
	Appearance json.RawMessage `json:"appearance"`
	Version    int64           `json:"version"`
	UpdatedAt  time.Time       `json:"updated_at"`
}

type UpdatePetAppearanceRequest struct {
	Appearance json.RawMessage `json:"appearance"`
}

type petRecord struct {
	Appearance json.RawMessage
	Version    int64
	UpdatedAt  time.Time
}

var validAppearanceKeys = map[string]struct{}{
	"fur_color":   {},
	"accessory":   {},
	"eye_style":   {},
	"outfit":      {},
	"mood":        {},
	"size":        {},
	"personality": {},
}

func ValidateAppearance(appearance json.RawMessage) error {
	var raw map[string]any
	if err := json.Unmarshal(appearance, &raw); err != nil {
		return err
	}
	for key := range raw {
		if _, ok := validAppearanceKeys[key]; !ok {
			return errInvalidAppearanceKey(key)
		}
	}
	return nil
}

type errInvalidAppearanceKey string

func (e errInvalidAppearanceKey) Error() string {
	return "invalid appearance key: " + string(e)
}
