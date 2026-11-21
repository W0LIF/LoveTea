package dates

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Service struct {
	repo *Repository
	db   *pgxpool.Pool
}

func NewService(repo *Repository, db *pgxpool.Pool) *Service {
	return &Service{repo: repo, db: db}
}

func (s *Service) CreateIdea(ctx context.Context, userID uuid.UUID, req CreateIdeaRequest) (DateIdea, error) {
	coupleID, err := s.repo.GetActiveCouple(ctx, userID)
	if err != nil {
		return DateIdea{}, err
	}
	if req.Title == "" {
		return DateIdea{}, ErrInvalidInput
	}
	category := req.Category
	if category == "" {
		category = "general"
	}
	idea := DateIdea{
		ID:             uuid.New(),
		CoupleID:       coupleID,
		ProposerUserID: userID,
		Title:          req.Title,
		Details:        req.Details,
		Category:       category,
		BudgetMin:      req.BudgetMin,
		BudgetMax:      req.BudgetMax,
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return DateIdea{}, fmt.Errorf("dates.CreateIdea begin: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := s.repo.CreateIdea(ctx, tx, &idea); err != nil {
		return DateIdea{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return DateIdea{}, fmt.Errorf("dates.CreateIdea commit: %w", err)
	}
	return s.repo.GetIdea(ctx, idea.ID)
}

func (s *Service) ListIdeas(ctx context.Context, userID uuid.UUID, status string) ([]DateIdea, error) {
	coupleID, err := s.repo.GetActiveCouple(ctx, userID)
	if err != nil {
		return nil, err
	}
	return s.repo.ListIdeas(ctx, coupleID, status)
}

func (s *Service) GetIdea(ctx context.Context, userID, id uuid.UUID) (DateIdea, error) {
	coupleID, err := s.repo.GetActiveCouple(ctx, userID)
	if err != nil {
		return DateIdea{}, err
	}
	idea, err := s.repo.GetIdea(ctx, id)
	if err != nil {
		return DateIdea{}, err
	}
	if idea.CoupleID != coupleID {
		return DateIdea{}, ErrNotFound
	}
	return idea, nil
}

func (s *Service) UpdateIdeaStatus(ctx context.Context, userID, id uuid.UUID, status string) (DateIdea, error) {
	coupleID, err := s.repo.GetActiveCouple(ctx, userID)
	if err != nil {
		return DateIdea{}, err
	}
	idea, err := s.repo.GetIdea(ctx, id)
	if err != nil {
		return DateIdea{}, err
	}
	if idea.CoupleID != coupleID {
		return DateIdea{}, ErrNotFound
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return DateIdea{}, fmt.Errorf("dates.UpdateIdeaStatus begin: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := s.repo.UpdateIdeaStatus(ctx, tx, id, status); err != nil {
		return DateIdea{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return DateIdea{}, fmt.Errorf("dates.UpdateIdeaStatus commit: %w", err)
	}
	return s.repo.GetIdea(ctx, id)
}

func (s *Service) ListRecommendations(ctx context.Context, userID uuid.UUID, limit int) ([]DateRecommendation, error) {
	coupleID, err := s.repo.GetActiveCouple(ctx, userID)
	if err != nil {
		return nil, err
	}
	return s.repo.ListRecommendations(ctx, userID, coupleID, limit)
}
