package users

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

var ErrInvalidProfile = errors.New("invalid profile data")

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Get(
	ctx context.Context,
	userID uuid.UUID,
) (*ProfileResponse, error) {
	profile, err := s.repo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	return toResponse(profile), nil
}

func (s *Service) Update(
	ctx context.Context,
	userID uuid.UUID,
	req UpdateProfileRequest,
) (*ProfileResponse, error) {
	if req.DisplayName == nil && req.Locale == nil && req.Timezone == nil {
		return nil, ErrInvalidProfile
	}

	if req.DisplayName != nil {
		value := strings.TrimSpace(*req.DisplayName)

		if utf8.RuneCountInString(value) < 1 ||
			utf8.RuneCountInString(value) > 50 {
			return nil, ErrInvalidProfile
		}

		req.DisplayName = &value
	}

	if req.Locale != nil {
		value := strings.TrimSpace(*req.Locale)
		if value != "ru" && value != "en" {
			return nil, ErrInvalidProfile
		}
		req.Locale = &value
	}

	if req.Timezone != nil {
		value := strings.TrimSpace(*req.Timezone)

		if len(value) > 64 {
			return nil, ErrInvalidProfile
		}

		if _, err := time.LoadLocation(value); err != nil {
			return nil, ErrInvalidProfile
		}

		req.Timezone = &value
	}

	profile, err := s.repo.Update(
		ctx,
		userID,
		req.DisplayName,
		req.Locale,
		req.Timezone,
	)
	if err != nil {
		return nil, err
	}

	return toResponse(profile), nil
}

func (s *Service) RegisterDevice(
	ctx context.Context,
	userID uuid.UUID,
	req RegisterDeviceRequest,
) (*DeviceResponse, error) {
	if req.DeviceID == "" || req.Platform == "" || req.PushToken == "" {
		return nil, ErrInvalidProfile
	}
	if req.Platform != "android" && req.Platform != "ios" {
		return nil, ErrInvalidProfile
	}

	d, err := s.repo.RegisterDevice(ctx, userID, req.DeviceID, req.Platform, req.PushToken, req.AppVersion)
	if err != nil {
		return nil, err
	}

	return toDeviceResponse(d), nil
}

func (s *Service) ListDevices(
	ctx context.Context,
	userID uuid.UUID,
) ([]DeviceResponse, error) {
	devices, err := s.repo.ListDevices(ctx, userID)
	if err != nil {
		return nil, err
	}
	responses := make([]DeviceResponse, 0, len(devices))
	for i := range devices {
		responses = append(responses, *toDeviceResponse(&devices[i]))
	}
	return responses, nil
}

func (s *Service) DeleteDevice(
	ctx context.Context,
	userID uuid.UUID,
	deviceID string,
) error {
	return s.repo.DeleteDevice(ctx, userID, deviceID)
}

func toDeviceResponse(d *Device) *DeviceResponse {
	return &DeviceResponse{
		ID:         d.ID.String(),
		UserID:     d.UserID.String(),
		DeviceID:   d.DeviceID,
		Platform:   d.Platform,
		AppVersion: d.AppVersion,
		CreatedAt:  d.CreatedAt,
		UpdatedAt:  d.UpdatedAt,
	}
}

func toResponse(profile *Profile) *ProfileResponse {
	return &ProfileResponse{
		UserID:      profile.UserID.String(),
		Email:       profile.Email,
		DisplayName: profile.DisplayName,
		AvatarKey:   profile.AvatarKey,
		Locale:      profile.Locale,
		Timezone:    profile.Timezone,
		CreatedAt:   profile.CreatedAt,
		UpdatedAt:   profile.UpdatedAt,
	}
}
