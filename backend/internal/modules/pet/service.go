package pet

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

func (s *Service) GetPetState(ctx context.Context, coupleID uuid.UUID) (PetState, error) {
	pet, err := s.repo.GetPetState(ctx, coupleID)
	if err != nil {
		return PetState{}, err
	}
	return PetState{
		CoupleID:   coupleID,
		Appearance: pet.Appearance,
		Version:    pet.Version,
		UpdatedAt:  pet.UpdatedAt,
	}, nil
}

func (s *Service) UpdatePetAppearance(
	ctx context.Context,
	coupleID uuid.UUID,
	req UpdatePetAppearanceRequest,
) (PetState, error) {
	if err := ValidateAppearance(req.Appearance); err != nil {
		return PetState{}, err
	}
	pet, err := s.repo.UpsertPetState(ctx, coupleID, req.Appearance)
	if err != nil {
		return PetState{}, err
	}
	return PetState{
		CoupleID:   coupleID,
		Appearance: pet.Appearance,
		Version:    pet.Version,
		UpdatedAt:  pet.UpdatedAt,
	}, nil
}
