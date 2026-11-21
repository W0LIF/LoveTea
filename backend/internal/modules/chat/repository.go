package chat

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	syncmodule "github.com/W0LIF/LoveTea/backend/internal/modules/sync"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNoActiveCouple = errors.New("no active couple")
	ErrNotFound       = errors.New("not found")
	ErrInvalidInput   = errors.New("invalid input")
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) GetOrCreateConversation(
	ctx context.Context,
	coupleID uuid.UUID,
) (Conversation, error) {
	var conv Conversation
	err := r.db.QueryRow(ctx, `
		SELECT id, created_at
		FROM chat_conversations
		WHERE couple_id = $1
	`, coupleID).Scan(&conv.ID, &conv.CreatedAt)
	if err == nil {
		conv.CoupleID = coupleID.String()
		return conv, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return Conversation{}, fmt.Errorf("query conversation: %w", err)
	}

	err = r.db.QueryRow(ctx, `
		INSERT INTO chat_conversations (couple_id)
		VALUES ($1)
		ON CONFLICT (couple_id) DO NOTHING
		RETURNING id, created_at
	`, coupleID).Scan(&conv.ID, &conv.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		err = r.db.QueryRow(ctx, `
			SELECT id, created_at
			FROM chat_conversations
			WHERE couple_id = $1
		`, coupleID).Scan(&conv.ID, &conv.CreatedAt)
	}
	if err != nil {
		return Conversation{}, fmt.Errorf("create conversation: %w", err)
	}
	conv.CoupleID = coupleID.String()
	return conv, nil
}

func (r *Repository) CreateMessage(
	ctx context.Context,
	convID, senderID, coupleID uuid.UUID,
	content, ciphertext string,
) (Message, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return Message{}, err
	}
	defer tx.Rollback(ctx)

	var (
		id      uuid.UUID
		createdAt time.Time
	)
	err = tx.QueryRow(ctx, `
		INSERT INTO chat_messages (conversation_id, sender_user_id, content, ciphertext)
		VALUES ($1, $2, $3, NULLIF($4, ''))
		RETURNING id, created_at
	`, convID, senderID, content, ciphertext).Scan(&id, &createdAt)
	if err != nil {
		return Message{}, fmt.Errorf("insert message: %w", err)
	}

	payload, err := json.Marshal(map[string]any{
		"conversation_id": convID.String(),
		"sender_id":       senderID.String(),
		"content":         content,
		"ciphertext":      ciphertext,
		"created_at":      createdAt.UTC().Format(time.RFC3339Nano),
	})
	if err != nil {
		return Message{}, fmt.Errorf("encode payload: %w", err)
	}

	if _, err := syncmodule.RecordChange(ctx, tx, syncmodule.NewChange{
		Scope:      syncmodule.Scope{CoupleID: &coupleID},
		ActorID:    senderID,
		EntityType: "chat_message",
		EntityID:   id,
		Operation:  "upsert",
		Payload:    payload,
	}); err != nil {
		return Message{}, fmt.Errorf("record sync change: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return Message{}, fmt.Errorf("commit: %w", err)
	}

	msg := Message{
		ID:             id.String(),
		ConversationID: convID.String(),
		SenderID:       senderID.String(),
		Content:        content,
		IsMine:         true,
		CreatedAt:      createdAt,
	}
	if ciphertext != "" {
		msg.Ciphertext = &ciphertext
	}
	return msg, nil
}

func (r *Repository) ListMessages(
	ctx context.Context,
	convID, userID uuid.UUID,
	limit int,
	before string,
) (MessageListResult, error) {
	var beforeTime *time.Time
	if before != "" {
		parsed, err := time.Parse(time.RFC3339Nano, before)
		if err != nil {
			return MessageListResult{}, ErrInvalidInput
		}
		beforeTime = &parsed
	}

	rows, err := r.db.Query(ctx, `
		SELECT m.id, m.sender_user_id, m.content, m.ciphertext, m.created_at
		FROM chat_messages m
		WHERE m.conversation_id = $1
		  AND m.deleted_at IS NULL
		  AND ($3::TIMESTAMPTZ IS NULL OR m.created_at < $3)
		ORDER BY m.created_at DESC, m.id DESC
		LIMIT $2
	`, convID, limit+1, beforeTime)
	if err != nil {
		return MessageListResult{}, fmt.Errorf("query messages: %w", err)
	}
	defer rows.Close()

	messages := make([]Message, 0, limit)
	for rows.Next() {
		var (
			id, senderID uuid.UUID
			content      string
			ciphertext   *string
			createdAt    time.Time
		)
		if err := rows.Scan(&id, &senderID, &content, &ciphertext, &createdAt); err != nil {
			return MessageListResult{}, fmt.Errorf("scan message: %w", err)
		}
		messages = append(messages, Message{
			ID:             id.String(),
			ConversationID: convID.String(),
			SenderID:       senderID.String(),
			Content:        content,
			IsMine:         senderID == userID,
			Ciphertext:     ciphertext,
			CreatedAt:      createdAt,
		})
	}
	if err := rows.Err(); err != nil {
		return MessageListResult{}, fmt.Errorf("iterate: %w", err)
	}

	result := MessageListResult{Messages: messages, HasMore: len(messages) > limit}
	if result.HasMore {
		result.Messages = messages[:limit]
	}
	for i, j := 0, len(result.Messages)-1; i < j; i, j = i+1, j-1 {
		result.Messages[i], result.Messages[j] = result.Messages[j], result.Messages[i]
	}
	if len(result.Messages) > 0 {
		result.NextID = result.Messages[0].CreatedAt.UTC().Format(time.RFC3339Nano)
	}
	return result, nil
}

func (r *Repository) DeleteMessage(
	ctx context.Context,
	messageID, userID uuid.UUID,
) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE chat_messages
		SET deleted_at = NOW(), content = '', ciphertext = NULL,
		    version = version + 1
		WHERE id = $1 AND sender_user_id = $2 AND deleted_at IS NULL
	`, messageID, userID)
	if err != nil {
		return fmt.Errorf("delete message: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repository) AddReaction(
	ctx context.Context,
	messageID, userID uuid.UUID,
	emoji string,
) (MessageReaction, error) {
	var reaction MessageReaction
	err := r.db.QueryRow(ctx, `
		INSERT INTO chat_message_reactions (message_id, user_id, emoji)
		VALUES ($1, $2, $3)
		ON CONFLICT (message_id, user_id, emoji) DO NOTHING
		RETURNING id, created_at
	`, messageID, userID, emoji).Scan(&reaction.ID, &reaction.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		err = r.db.QueryRow(ctx, `
			SELECT id, created_at
			FROM chat_message_reactions
			WHERE message_id = $1 AND user_id = $2 AND emoji = $3
		`, messageID, userID, emoji).Scan(&reaction.ID, &reaction.CreatedAt)
	}
	if err != nil {
		return MessageReaction{}, fmt.Errorf("add reaction: %w", err)
	}
	reaction.MessageID = messageID.String()
	reaction.UserID = userID.String()
	reaction.IsMine = true
	reaction.Emoji = emoji
	return reaction, nil
}

func (r *Repository) RemoveReaction(
	ctx context.Context,
	messageID, userID uuid.UUID,
	emoji string,
) error {
	_, err := r.db.Exec(ctx, `
		DELETE FROM chat_message_reactions
		WHERE message_id = $1 AND user_id = $2 AND emoji = $3
	`, messageID, userID, emoji)
	if err != nil {
		return fmt.Errorf("remove reaction: %w", err)
	}
	return nil
}

func (r *Repository) ListReactions(
	ctx context.Context,
	messageIDs []uuid.UUID,
) (map[string][]MessageReaction, error) {
	if len(messageIDs) == 0 {
		return map[string][]MessageReaction{}, nil
	}
	rows, err := r.db.Query(ctx, `
		SELECT id, message_id, user_id, emoji, created_at
		FROM chat_message_reactions
		WHERE message_id = ANY($1)
		ORDER BY created_at ASC
	`, messageIDs)
	if err != nil {
		return nil, fmt.Errorf("query reactions: %w", err)
	}
	defer rows.Close()

	byMessage := make(map[string][]MessageReaction)
	for rows.Next() {
		var (
			id, messageID, userID uuid.UUID
			reaction              MessageReaction
		)
		if err := rows.Scan(&id, &messageID, &userID, &reaction.Emoji, &reaction.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan reaction: %w", err)
		}
		reaction.ID = id.String()
		reaction.MessageID = messageID.String()
		reaction.UserID = userID.String()
		byMessage[messageID.String()] = append(byMessage[messageID.String()], reaction)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate: %w", err)
	}
	return byMessage, nil
}

func (r *Repository) GetActiveCouple(ctx context.Context, userID uuid.UUID) (uuid.UUID, error) {
	var coupleID uuid.UUID
	err := r.db.QueryRow(ctx, `
		SELECT cm.couple_id
		FROM couple_members cm
		JOIN couples c ON c.id = cm.couple_id
		WHERE cm.user_id = $1 AND cm.left_at IS NULL AND c.status = 'active'
		LIMIT 1
	`, userID).Scan(&coupleID)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, ErrNoActiveCouple
	}
	if err != nil {
		return uuid.Nil, fmt.Errorf("get active couple: %w", err)
	}
	return coupleID, nil
}
