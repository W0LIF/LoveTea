package letters

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

var (
	ErrInvalidInput   = errors.New("invalid letter input")
	ErrNoActiveCouple = errors.New("no active couple")
	ErrNotFound       = errors.New("letter not found")
	ErrForbidden      = errors.New("letter not yet ready")
	ErrCooldownActive = errors.New("cooldown active")
)

const (
	defaultCooldown = 30 * time.Minute
	minCooldown     = 1 * time.Minute
	maxCooldown     = 24 * time.Hour
)

type repository interface {
	GetActiveCoupleAndPartner(ctx context.Context, userID uuid.UUID) (coupleID, partnerID uuid.UUID, err error)
	Create(ctx context.Context, senderID, recipientID, coupleID uuid.UUID, content string, cooldown time.Duration) (Letter, error)
	ListBySender(ctx context.Context, userID uuid.UUID) ([]Letter, error)
	ListByRecipient(ctx context.Context, userID uuid.UUID) ([]Letter, error)
	GetByID(ctx context.Context, letterID, userID uuid.UUID) (Letter, error)
	MarkRead(ctx context.Context, letterID, recipientID uuid.UUID) error
	Archive(ctx context.Context, letterID, userID uuid.UUID, version int64) error
	GetLastCooldown(ctx context.Context, userID uuid.UUID) (time.Time, error)
}

type Service struct {
	repo     repository
	cooldown time.Duration
}

func NewService(repo repository) *Service {
	return &Service{repo: repo, cooldown: defaultCooldown}
}

func (s *Service) ListMyLetters(ctx context.Context, userID uuid.UUID) ([]Letter, error) {
	return s.repo.ListBySender(ctx, userID)
}

func (s *Service) ListForMe(ctx context.Context, userID uuid.UUID) ([]Letter, error) {
	return s.repo.ListByRecipient(ctx, userID)
}

func (s *Service) CreateLetter(
	ctx context.Context,
	userID uuid.UUID,
	req CreateLetterRequest,
) (Letter, error) {
	content := strings.TrimSpace(req.Content)
	if utf8.RuneCountInString(content) < 1 || utf8.RuneCountInString(content) > 10000 {
		return Letter{}, ErrInvalidInput
	}

	cooldown := s.cooldown
	if req.CooldownMinutes != nil {
		cooldown = time.Duration(*req.CooldownMinutes) * time.Minute
		if cooldown < minCooldown || cooldown > maxCooldown {
			return Letter{}, ErrInvalidInput
		}
	}

	// Проверяем, не активен ли уже cooldown.
	lastCooldown, err := s.repo.GetLastCooldown(ctx, userID)
	if err != nil {
		return Letter{}, err
	}
	if !lastCooldown.IsZero() && lastCooldown.After(time.Now()) {
		return Letter{}, ErrCooldownActive
	}

	coupleID, partnerID, err := s.repo.GetActiveCoupleAndPartner(ctx, userID)
	if err != nil {
		return Letter{}, err
	}

	return s.repo.Create(ctx, userID, partnerID, coupleID, content, cooldown)
}

func (s *Service) GetLetter(
	ctx context.Context,
	userID uuid.UUID,
	letterID uuid.UUID,
) (Letter, error) {
	letter, err := s.repo.GetByID(ctx, letterID, userID)
	if err != nil {
		return Letter{}, err
	}
	// Отправитель всегда видит. Получатель — только после cooldown.
	if letter.SenderID != userID.String() && letter.Status == "waiting" {
		return Letter{}, ErrForbidden
	}
	return letter, nil
}

func (s *Service) MarkRead(ctx context.Context, userID, letterID uuid.UUID) error {
	return s.repo.MarkRead(ctx, letterID, userID)
}

func (s *Service) Archive(
	ctx context.Context,
	userID uuid.UUID,
	letterID uuid.UUID,
	version int64,
) error {
	if version < 1 {
		return ErrInvalidInput
	}
	return s.repo.Archive(ctx, letterID, userID, version)
}

func (s *Service) GetCooldownRemaining(
	ctx context.Context,
	userID uuid.UUID,
) (time.Duration, error) {
	last, err := s.repo.GetLastCooldown(ctx, userID)
	if err != nil {
		return 0, err
	}
	if last.IsZero() || last.Before(time.Now()) {
		return 0, nil
	}
	return time.Until(last), nil
}
