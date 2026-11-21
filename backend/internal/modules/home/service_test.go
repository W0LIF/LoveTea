package home

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/W0LIF/LoveTea/backend/internal/modules/couples"
	"github.com/W0LIF/LoveTea/backend/internal/modules/users"
	"github.com/google/uuid"
)

type fakeProfiles struct {
	profile *users.ProfileResponse
	err     error
}

func (f fakeProfiles) Get(context.Context, uuid.UUID) (*users.ProfileResponse, error) {
	return f.profile, f.err
}

type fakeCouples struct {
	couple couples.MyCoupleResponse
	err    error
}

func (f fakeCouples) GetMyCouple(context.Context, uuid.UUID) (couples.MyCoupleResponse, error) {
	return f.couple, f.err
}

type fakePets struct {
	pet petRecord
	err error
}

func (f fakePets) GetPetState(context.Context, uuid.UUID) (petRecord, error) {
	return f.pet, f.err
}

func TestGetIncludesPetForCurrentCouple(t *testing.T) {
	userID := uuid.New()
	coupleID := uuid.New()
	profile := &users.ProfileResponse{UserID: userID.String()}
	now := time.Now()
	service := NewService(
		fakeProfiles{profile: profile},
		fakeCouples{couple: couples.MyCoupleResponse{
			CoupleID: coupleID.String(),
			Status:   "active",
		}},
		fakePets{pet: petRecord{
			Appearance: []byte(`{}`),
			Version:    1,
			UpdatedAt:  now,
		}},
	)

	response, err := service.Get(context.Background(), userID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if response.Profile != profile || response.Couple == nil || response.Pet == nil {
		t.Fatalf("Get() did not return the home snapshot: %+v", response)
	}
	if response.Pet.CoupleID != coupleID || response.Pet.Version != 1 {
		t.Fatalf("unexpected pet state: %+v", response.Pet)
	}
}

func TestGetReturnsProfileWhenUserHasNoCouple(t *testing.T) {
	userID := uuid.New()
	profile := &users.ProfileResponse{UserID: userID.String()}
	service := NewService(
		fakeProfiles{profile: profile},
		fakeCouples{err: couples.ErrNoActiveCouple},
		fakePets{},
	)

	response, err := service.Get(context.Background(), userID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if response.Profile != profile || response.Couple != nil || response.Pet != nil {
		t.Fatalf("unexpected single-user home snapshot: %+v", response)
	}
}

func TestGetPropagatesUnexpectedCoupleAndPetErrors(t *testing.T) {
	userID := uuid.New()
	profile := &users.ProfileResponse{UserID: userID.String()}
	unexpected := errors.New("database unavailable")

	_, err := NewService(
		fakeProfiles{profile: profile},
		fakeCouples{err: unexpected},
		fakePets{},
	).Get(context.Background(), userID)
	if !errors.Is(err, unexpected) {
		t.Fatalf("Get() error = %v, want couple error", err)
	}

	coupleID := uuid.New()
	_, err = NewService(
		fakeProfiles{profile: profile},
		fakeCouples{couple: couples.MyCoupleResponse{CoupleID: coupleID.String()}},
		fakePets{err: unexpected},
	).Get(context.Background(), userID)
	if !errors.Is(err, unexpected) {
		t.Fatalf("Get() error = %v, want pet error", err)
	}
}
