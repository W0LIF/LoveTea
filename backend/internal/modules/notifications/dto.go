package notifications

import (
	"encoding/json"
	"time"
)

type Notification struct {
	ID        int64           `json:"id"`
	UserID    string          `json:"user_id"`
	Category  string          `json:"category"`
	Title     string          `json:"title"`
	Body      string          `json:"body"`
	Data      json.RawMessage `json:"data"`
	IsRead    bool            `json:"is_read"`
	CreatedAt time.Time       `json:"created_at"`
}

type NotificationSettings struct {
	UserID            string          `json:"user_id"`
	EnabledCategories json.RawMessage `json:"enabled_categories"`
	QuietHoursStart   int             `json:"quiet_hours_start"`
	QuietHoursEnd     int             `json:"quiet_hours_end"`
	UpdatedAt         time.Time       `json:"updated_at"`
}

type UpdateSettingsRequest struct {
	EnabledCategories []string `json:"enabled_categories"`
	QuietHoursStart   *int     `json:"quiet_hours_start"`
	QuietHoursEnd     *int     `json:"quiet_hours_end"`
}

type CreateNotificationRequest struct {
	UserID   string          `json:"user_id"`
	Category string          `json:"category"`
	Title    string          `json:"title"`
	Body     string          `json:"body"`
	Data     json.RawMessage `json:"data"`
}
