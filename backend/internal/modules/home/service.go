package home

import (
	"context"
	"errors"
	"fmt"

	"github.com/W0LIF/LoveTea/backend/internal/modules/couples"
	"github.com/W0LIF/LoveTea/backend/internal/modules/users"
	"github.com/google/uuid"
)

type profileReader interface {
	Get(context.Context, uuid.UUID) (*users.ProfileResponse, error)
}

type coupleReader interface {
	GetMyCouple(context.Context, uuid.UUID) (couples.MyCoupleResponse, error)
}

type petReader interface {
	GetPetState(context.Context, uuid.UUID) (petRecord, error)
}

type Service struct {
	profiles profileReader
	couples  coupleReader
	pets     petReader
}

func NewService(
	profiles profileReader,
	couples coupleReader,
	pets petReader,
) *Service {
	return &Service{
		profiles: profiles,
		couples:  couples,
		pets:     pets,
	}
}

func (s *Service) Get(ctx context.Context, userID uuid.UUID) (Response, error) {
	profile, err := s.profiles.Get(ctx, userID)
	if err != nil {
		return Response{}, err
	}

	result := Response{Profile: profile}
	couple, err := s.couples.GetMyCouple(ctx, userID)
	if errors.Is(err, couples.ErrNoActiveCouple) {
		return result, nil
	}
	if err != nil {
		return Response{}, err
	}
	result.Couple = &couple

	coupleID, err := uuid.Parse(couple.CoupleID)
	if err != nil {
		return Response{}, fmt.Errorf("parse home couple id: %w", err)
	}
	pet, err := s.pets.GetPetState(ctx, coupleID)
	if err != nil {
		return Response{}, fmt.Errorf("get home pet state: %w", err)
	}
	result.Pet = &PetState{
		CoupleID:   coupleID,
		Appearance: pet.Appearance,
		Version:    pet.Version,
		UpdatedAt:  pet.UpdatedAt,
	}

	return result, nil
}
