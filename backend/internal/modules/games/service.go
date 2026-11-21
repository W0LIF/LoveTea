package games

import (
	"context"
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

func (s *Service) ListSessions(ctx context.Context, userID uuid.UUID) ([]GameSession, error) {
	return s.repo.ListSessions(ctx, userID)
}

func (s *Service) StartSession(ctx context.Context, userID uuid.UUID, req StartGameRequest) (GameSession, error) {
	coupleID, partnerID, err := s.repo.GetActiveCoupleAndPartner(ctx, userID)
	if err != nil {
		return GameSession{}, err
	}
	totalQuestions := req.TotalQuestions
	if totalQuestions < 1 || totalQuestions > 50 {
		totalQuestions = 10
	}
	gameType := strings.TrimSpace(req.Type)
	if gameType != "quiz" && gameType != "mini" {
		gameType = "quiz"
	}
	return s.repo.CreateSession(ctx, coupleID, userID, partnerID, gameType, totalQuestions)
}

func (s *Service) SubmitAnswer(ctx context.Context, userID uuid.UUID, req SubmitAnswerRequest) (AnswerResult, error) {
	id, err := uuid.Parse(req.SessionID)
	if err != nil {
		return AnswerResult{}, ErrInvalidInput
	}
	if utf8.RuneCountInString(req.Question) == 0 || utf8.RuneCountInString(req.Answer) == 0 {
		return AnswerResult{}, ErrInvalidInput
	}
	return s.repo.SubmitAnswer(ctx, id, userID, req.Question, req.Answer)
}

func (s *Service) FinishSession(ctx context.Context, userID uuid.UUID, sessionID uuid.UUID) error {
	return s.repo.FinishSession(ctx, sessionID, userID)
}
