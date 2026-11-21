package users

import "time"

type ProfileResponse struct {
	UserID      string    `json:"user_id"`
	Email       string    `json:"email"`
	DisplayName *string   `json:"display_name"`
	AvatarKey   *string   `json:"avatar_key"`
	Locale      string    `json:"locale"`
	Timezone    string    `json:"timezone"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type UpdateProfileRequest struct {
	DisplayName *string `json:"display_name"`
	Locale      *string `json:"locale"`
	Timezone    *string `json:"timezone"`
}

type RegisterDeviceRequest struct {
	DeviceID   string `json:"device_id"`
	Platform   string `json:"platform"`
	PushToken  string `json:"push_token"`
	AppVersion string `json:"app_version"`
}

type DeviceResponse struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	DeviceID  string    `json:"device_id"`
	Platform  string    `json:"platform"`
	AppVersion string   `json:"app_version"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
