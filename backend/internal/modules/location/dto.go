package location

import "time"

type SharingStatus struct {
	IsActive      bool       `json:"is_active"`
	LastSharingAt *time.Time `json:"last_sharing_at,omitempty"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

type Position struct {
	UserID     string    `json:"user_id"`
	Latitude   float64   `json:"latitude"`
	Longitude  float64   `json:"longitude"`
	Accuracy   *float64  `json:"accuracy,omitempty"`
	CapturedAt time.Time `json:"captured_at"`
}

type Geofence struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Latitude     float64   `json:"latitude"`
	Longitude    float64   `json:"longitude"`
	RadiusMeters float64   `json:"radius_meters"`
	CreatedAt    time.Time `json:"created_at"`
}

type ToggleSharingRequest struct {
	IsActive bool `json:"is_active"`
}

type UpdatePositionRequest struct {
	Latitude  float64  `json:"latitude"`
	Longitude float64  `json:"longitude"`
	Accuracy  *float64 `json:"accuracy"`
}

type CreateGeofenceRequest struct {
	Name         string  `json:"name"`
	Latitude     float64 `json:"latitude"`
	Longitude    float64 `json:"longitude"`
	RadiusMeters float64 `json:"radius_meters"`
}
