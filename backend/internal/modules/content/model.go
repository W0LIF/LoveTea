package content

import (
	"time"
)

type MediaFile struct {
	ID          string    `json:"id"`
	Scope       string    `json:"scope"`
	ContentType string    `json:"content_type"`
	SizeBytes   int64     `json:"size_bytes"`
	Status      string    `json:"status"`
	Version     int64     `json:"version"`
	CreatedAt   time.Time `json:"created_at"`
}

type MediaAttachment struct {
	ID          string `json:"id"`
	ContentType string `json:"content_type"`
	SizeBytes   int64  `json:"size_bytes"`
}

type UploadRequest struct {
	Scope       string `json:"scope"`
	ContentType string `json:"content_type"`
}

type UploadResponse struct {
	Media      MediaFile         `json:"media"`
	UploadURL  string            `json:"upload_url"`
	FormFields map[string]string `json:"form_fields"`
	ExpiresAt  time.Time         `json:"expires_at"`
}

type Moment struct {
	ID         string            `json:"id"`
	Scope      string            `json:"scope"`
	Title      string            `json:"title"`
	Caption    string            `json:"caption"`
	HappenedAt time.Time         `json:"happened_at"`
	Media      []MediaAttachment `json:"media"`
	Version    int64             `json:"version"`
	CreatedAt  time.Time         `json:"created_at"`
	UpdatedAt  time.Time         `json:"updated_at"`
}

type CreateMomentRequest struct {
	Scope      string    `json:"scope"`
	Title      string    `json:"title"`
	Caption    string    `json:"caption"`
	HappenedAt time.Time `json:"happened_at"`
	MediaIDs   []string  `json:"media_ids"`
}
