package users

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrProfileNotFound = errors.New("profile not found")

type Profile struct {
	UserID      uuid.UUID
	Email       string
	DisplayName *string
	AvatarKey   *string
	Locale      string
	Timezone    string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) GetByUserID(
	ctx context.Context,
	userID uuid.UUID,
) (*Profile, error) {
	const query = `
		SELECT
			u.id,
			u.email,
			p.display_name,
			p.avatar_key,
			p.locale,
			p.timezone,
			p.created_at,
			p.updated_at
		FROM users u
		JOIN user_profiles p ON p.user_id = u.id
		WHERE u.id = $1
	`

	profile := &Profile{}

	err := r.db.QueryRow(ctx, query, userID).Scan(
		&profile.UserID,
		&profile.Email,
		&profile.DisplayName,
		&profile.AvatarKey,
		&profile.Locale,
		&profile.Timezone,
		&profile.CreatedAt,
		&profile.UpdatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrProfileNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get profile: %w", err)
	}

	return profile, nil
}

func (r *Repository) Update(
	ctx context.Context,
	userID uuid.UUID,
	displayName *string,
	locale *string,
	timezone *string,
) (*Profile, error) {
	const query = `
		UPDATE user_profiles
		SET
			display_name = COALESCE($2, display_name),
			locale = COALESCE($3, locale),
			timezone = COALESCE($4, timezone),
			updated_at = NOW()
		WHERE user_id = $1
		RETURNING user_id
	`

	var updatedID uuid.UUID

	err := r.db.QueryRow(
		ctx,
		query,
		userID,
		displayName,
		locale,
		timezone,
	).Scan(&updatedID)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrProfileNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("update profile: %w", err)
	}

	return r.GetByUserID(ctx, updatedID)
}

// RegisterDevice upserts a user's push device token.
func (r *Repository) RegisterDevice(
	ctx context.Context,
	userID uuid.UUID,
	deviceID, platform, pushToken, appVersion string,
) (*Device, error) {
	var d Device
	err := r.db.QueryRow(ctx, `
		INSERT INTO user_devices (user_id, device_id, platform, push_token, app_version, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
		ON CONFLICT (user_id, device_id, platform)
		DO UPDATE SET
			push_token = EXCLUDED.push_token,
			app_version = EXCLUDED.app_version,
			updated_at = NOW()
		RETURNING id, user_id, device_id, platform, app_version, created_at, updated_at
	`, userID, deviceID, platform, pushToken, appVersion).Scan(
		&d.ID, &d.UserID, &d.DeviceID, &d.Platform, &d.PushToken, &d.AppVersion, &d.CreatedAt, &d.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("register device: %w", err)
	}
	return &d, nil
}

// ListDevices returns all registered devices for a user.
func (r *Repository) ListDevices(
	ctx context.Context,
	userID uuid.UUID,
) ([]Device, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, user_id, device_id, platform, push_token, app_version, created_at, updated_at
		FROM user_devices
		WHERE user_id = $1
		ORDER BY updated_at DESC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("list devices: %w", err)
	}
	defer rows.Close()
	var devices []Device
	for rows.Next() {
		var d Device
		if err := rows.Scan(
			&d.ID, &d.UserID, &d.DeviceID, &d.Platform, &d.PushToken, &d.AppVersion, &d.CreatedAt, &d.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan device: %w", err)
		}
		devices = append(devices, d)
	}
	return devices, rows.Err()
}

// DeleteDevice removes a registered device.
func (r *Repository) DeleteDevice(
	ctx context.Context,
	userID uuid.UUID,
	deviceID string,
) error {
	_, err := r.db.Exec(ctx, `
		DELETE FROM user_devices
		WHERE user_id = $1 AND device_id = $2
	`, userID, deviceID)
	if err != nil {
		return fmt.Errorf("delete device: %w", err)
	}
	return nil
}

type Device struct {
	ID         uuid.UUID
	UserID     uuid.UUID
	DeviceID   string
	Platform   string
	PushToken  string
	AppVersion string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}
