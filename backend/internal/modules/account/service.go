package account

import (
	"context"

	"github.com/google/uuid"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// CreateExport assembles a JSON snapshot of the user's personal data and
// stores it as a data URL on the export row.
func (s *Service) CreateExport(ctx context.Context, userID uuid.UUID) (Export, error) {
	e, err := s.repo.CreateExport(ctx, userID)
	if err != nil {
		return Export{}, err
	}

	user, err := s.repo.GetUser(ctx, userID)
	if err != nil {
		return Export{}, err
	}
	profile, err := s.repo.GetProfile(ctx, userID)
	if err != nil {
		return Export{}, err
	}
	personal, err := s.repo.ExportPersonalData(ctx, userID)
	if err != nil {
		return Export{}, err
	}

	payload := map[string]any{
		"exported_at": e.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		"user":        user,
		"profile":     profile,
		"data":        personal,
	}

	data, err := EncodeExport(payload)
	if err != nil {
		return Export{}, err
	}

	if _, err := s.repo.SaveExportJSON(ctx, e.ID, data); err != nil {
		return Export{}, err
	}

	return s.repo.GetExport(ctx, e.ID, userID)
}

func (s *Service) GetExport(ctx context.Context, userID, exportID uuid.UUID) (Export, error) {
	return s.repo.GetExport(ctx, exportID, userID)
}

func (s *Service) DeleteAccount(ctx context.Context, userID uuid.UUID, confirm string) error {
	if confirm != "DELETE" {
		return ErrInvalidConfirm
	}
	return s.repo.DeleteAccount(ctx, userID)
}
