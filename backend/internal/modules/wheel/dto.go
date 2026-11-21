package wheel

import "time"

type Sector struct {
	ID                   string    `json:"id"`
	Title                string    `json:"title"`
	Description          string    `json:"description"`
	RequiresConfirmation bool      `json:"requires_confirmation"`
	Position             int       `json:"position"`
	CreatedAt            time.Time `json:"created_at"`
}

type Spin struct {
	ID                string     `json:"id"`
	SectorID          *string    `json:"sector_id,omitempty"`
	SpunBy            string     `json:"spun_by"`
	Confirmed         bool       `json:"confirmed"`
	Cancelled         bool       `json:"cancelled"`
	CreatedAt         time.Time  `json:"created_at"`
	ConfirmedAt       *time.Time `json:"confirmed_at,omitempty"`
}

type CreateSectorRequest struct {
	Title                string `json:"title"`
	Description          string `json:"description"`
	RequiresConfirmation bool   `json:"requires_confirmation"`
	Position             int    `json:"position"`
}

type ConfirmRequest struct {
	SpinID string `json:"spin_id"`
}
