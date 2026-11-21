package location

import (
	"context"
	"math"

	"github.com/google/uuid"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) GetSharingStatus(ctx context.Context, userID uuid.UUID) (SharingStatus, error) {
	coupleID, err := s.repo.GetActiveCouple(ctx, userID)
	if err != nil {
		return SharingStatus{}, err
	}
	return s.repo.GetSharingStatus(ctx, userID, coupleID)
}

func (s *Service) ToggleSharing(ctx context.Context, userID uuid.UUID, req ToggleSharingRequest) (SharingStatus, error) {
	coupleID, err := s.repo.GetActiveCouple(ctx, userID)
	if err != nil {
		return SharingStatus{}, err
	}
	return s.repo.UpsertSharing(ctx, userID, coupleID, req.IsActive)
}

func (s *Service) UpdatePosition(ctx context.Context, userID uuid.UUID, req UpdatePositionRequest) error {
	if math.Abs(req.Latitude) > 90 || math.Abs(req.Longitude) > 180 {
		return ErrInvalidInput
	}
	coupleID, err := s.repo.GetActiveCouple(ctx, userID)
	if err != nil {
		return err
	}
	status, err := s.repo.GetSharingStatus(ctx, userID, coupleID)
	if err != nil {
		return err
	}
	if !status.IsActive {
		return nil // Sharing is off — silently ignore.
	}
	return s.repo.InsertPosition(ctx, userID, coupleID, req.Latitude, req.Longitude, req.Accuracy)
}

func (s *Service) GetPartnerPosition(ctx context.Context, userID uuid.UUID) (Position, error) {
	coupleID, err := s.repo.GetActiveCouple(ctx, userID)
	if err != nil {
		return Position{}, err
	}
	partnerID, err := s.repo.GetPartnerID(ctx, userID)
	if err != nil {
		return Position{}, err
	}
	return s.repo.GetPartnerLastPosition(ctx, partnerID, coupleID)
}

func (s *Service) ListGeofences(ctx context.Context, userID uuid.UUID) ([]Geofence, error) {
	coupleID, err := s.repo.GetActiveCouple(ctx, userID)
	if err != nil {
		return nil, err
	}
	return s.repo.ListGeofences(ctx, coupleID)
}

func (s *Service) CreateGeofence(ctx context.Context, userID uuid.UUID, req CreateGeofenceRequest) (Geofence, error) {
	if req.RadiusMeters <= 0 || req.RadiusMeters > 100000 {
		return Geofence{}, ErrInvalidInput
	}
	if math.Abs(req.Latitude) > 90 || math.Abs(req.Longitude) > 180 {
		return Geofence{}, ErrInvalidInput
	}
	coupleID, err := s.repo.GetActiveCouple(ctx, userID)
	if err != nil {
		return Geofence{}, err
	}
	return s.repo.CreateGeofence(ctx, coupleID, userID, req)
}
