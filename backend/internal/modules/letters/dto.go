package letters

import "time"

type Letter struct {
	ID              string     `json:"id"`
	SenderID        string     `json:"sender_id"`
	SenderName      string     `json:"sender_name"`
	RecipientID     string     `json:"recipient_id"`
	RecipientName   string     `json:"recipient_name"`
	Content         string     `json:"content"`
	Status          string     `json:"status"`
	CooldownUntil   time.Time  `json:"cooldown_until"`
	Version         int64      `json:"version"`
	CreatedAt       time.Time  `json:"created_at"`
	SentAt          *time.Time `json:"sent_at,omitempty"`
	ReadAt          *time.Time `json:"read_at,omitempty"`
}

type CreateLetterRequest struct {
	Content         string `json:"content"`
	CooldownMinutes *int   `json:"cooldown_minutes"`
}
