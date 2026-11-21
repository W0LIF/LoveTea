package chat

import "time"

type Conversation struct {
	ID        string    `json:"id"`
	CoupleID  string    `json:"couple_id"`
	CreatedAt time.Time `json:"created_at"`
}

type Message struct {
	ID         string    `json:"id"`
	ConversationID string `json:"conversation_id"`
	SenderID   string    `json:"sender_id"`
	Content    string    `json:"content"`
	IsMine     bool      `json:"is_mine"`
	IsRead     bool      `json:"is_read"`
	Ciphertext *string   `json:"ciphertext,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

type MessageReaction struct {
	ID         string    `json:"id"`
	MessageID  string    `json:"message_id"`
	UserID     string    `json:"user_id"`
	IsMine     bool      `json:"is_mine"`
	Emoji      string    `json:"emoji"`
	CreatedAt  time.Time `json:"created_at"`
}

type CreateMessageRequest struct {
	Content    string `json:"content"`
	Ciphertext string `json:"ciphertext,omitempty"`
}

type AddReactionRequest struct {
	Emoji string `json:"emoji"`
}

type MessageListResult struct {
	Messages []Message     `json:"messages"`
	HasMore  bool          `json:"has_more"`
	NextID   string        `json:"next_id,omitempty"`
}

type ReactionListResult struct {
	Reactions []MessageReaction `json:"reactions"`
}
