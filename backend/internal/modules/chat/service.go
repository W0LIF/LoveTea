package chat

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) GetConversation(
	ctx context.Context,
	userID uuid.UUID,
) (Conversation, error) {
	coupleID, err := s.repo.GetActiveCouple(ctx, userID)
	if err != nil {
		return Conversation{}, err
	}
	return s.repo.GetOrCreateConversation(ctx, coupleID)
}

func (s *Service) ListMessages(
	ctx context.Context,
	userID uuid.UUID,
	limit int,
	before string,
) (MessageListResult, error) {
	if limit < 1 || limit > 50 {
		return MessageListResult{}, ErrInvalidInput
	}
	coupleID, err := s.repo.GetActiveCouple(ctx, userID)
	if err != nil {
		return MessageListResult{}, err
	}
	conv, err := s.repo.GetOrCreateConversation(ctx, coupleID)
	if err != nil {
		return MessageListResult{}, err
	}
	convUUID, _ := uuid.Parse(conv.ID)
	return s.repo.ListMessages(ctx, convUUID, userID, limit, before)
}

func (s *Service) SendMessage(
	ctx context.Context,
	userID uuid.UUID,
	content, ciphertext string,
) (Message, error) {
	content = strings.TrimSpace(content)
	if utf8.RuneCountInString(content) < 1 || utf8.RuneCountInString(content) > 5000 {
		return Message{}, ErrInvalidInput
	}

	coupleID, err := s.repo.GetActiveCouple(ctx, userID)
	if err != nil {
		return Message{}, err
	}
	conv, err := s.repo.GetOrCreateConversation(ctx, coupleID)
	if err != nil {
		return Message{}, err
	}
	convUUID, _ := uuid.Parse(conv.ID)
	return s.repo.CreateMessage(ctx, convUUID, userID, coupleID, content, ciphertext)
}

func (s *Service) DeleteMessage(
	ctx context.Context,
	userID uuid.UUID,
	messageID uuid.UUID,
) error {
	return s.repo.DeleteMessage(ctx, messageID, userID)
}

func (s *Service) AddReaction(
	ctx context.Context,
	userID uuid.UUID,
	messageID uuid.UUID,
	emoji string,
) (MessageReaction, error) {
	emoji = strings.TrimSpace(emoji)
	if utf8.RuneCountInString(emoji) < 1 || utf8.RuneCountInString(emoji) > 20 {
		return MessageReaction{}, ErrInvalidInput
	}
	return s.repo.AddReaction(ctx, messageID, userID, emoji)
}

func (s *Service) RemoveReaction(
	ctx context.Context,
	userID uuid.UUID,
	messageID uuid.UUID,
	emoji string,
) error {
	return s.repo.RemoveReaction(ctx, messageID, userID, emoji)
}

func (s *Service) ListReactions(
	ctx context.Context,
	userID uuid.UUID,
	messageIDs []string,
) (map[string][]MessageReaction, error) {
	if len(messageIDs) == 0 {
		return map[string][]MessageReaction{}, nil
	}
	uuids := make([]uuid.UUID, len(messageIDs))
	for i, id := range messageIDs {
		parsed, err := uuid.Parse(id)
		if err != nil {
			return nil, ErrInvalidInput
		}
		uuids[i] = parsed
	}
	return s.repo.ListReactions(ctx, uuids)
}

func mustCoupleID(ctx context.Context, userID uuid.UUID, repo *Repository) uuid.UUID {
	coupleID, err := repo.GetActiveCouple(ctx, userID)
	if err != nil {
		panic(fmt.Sprintf("chat mustCoupleID: %v", err))
	}
	return coupleID
}
