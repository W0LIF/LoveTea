package wheel

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

func (s *Service) ListSectors(ctx context.Context, userID uuid.UUID) ([]Sector, error) {
	coupleID, err := s.repo.GetActiveCouple(ctx, userID)
	if err != nil {
		return nil, err
	}
	return s.repo.ListSectors(ctx, coupleID)
}

func (s *Service) CreateSector(ctx context.Context, userID uuid.UUID, req CreateSectorRequest) (Sector, error) {
	coupleID, err := s.repo.GetActiveCouple(ctx, userID)
	if err != nil {
		return Sector{}, err
	}
	req.Title = strings.TrimSpace(req.Title)
	if utf8.RuneCountInString(req.Title) < 1 || utf8.RuneCountInString(req.Title) > 100 {
		return Sector{}, ErrInvalidInput
	}
	return s.repo.CreateSector(ctx, coupleID, req)
}

func (s *Service) DeleteSector(ctx context.Context, userID, sectorID uuid.UUID) error {
	coupleID, err := s.repo.GetActiveCouple(ctx, userID)
	if err != nil {
		return err
	}
	return s.repo.DeleteSector(ctx, sectorID, coupleID)
}

func (s *Service) Spin(ctx context.Context, userID uuid.UUID) (Spin, error) {
	coupleID, err := s.repo.GetActiveCouple(ctx, userID)
	if err != nil {
		return Spin{}, err
	}
	return s.repo.Spin(ctx, coupleID, userID)
}

func (s *Service) Confirm(ctx context.Context, userID uuid.UUID, req ConfirmRequest) error {
	spinID, err := uuid.Parse(req.SpinID)
	if err != nil {
		return ErrInvalidInput
	}
	return s.repo.Confirm(ctx, spinID, userID)
}

func (s *Service) History(ctx context.Context, userID uuid.UUID) ([]Spin, error) {
	coupleID, err := s.repo.GetActiveCouple(ctx, userID)
	if err != nil {
		return nil, err
	}
	return s.repo.History(ctx, coupleID)
}
