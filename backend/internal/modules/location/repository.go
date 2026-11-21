package location

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNoActiveCouple = errors.New("no active couple")
	ErrInvalidInput   = errors.New("invalid input")
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) GetActiveCouple(ctx context.Context, userID uuid.UUID) (uuid.UUID, error) {
	var coupleID uuid.UUID
	err := r.db.QueryRow(ctx, `
		SELECT cm.couple_id
		FROM couple_members cm
		JOIN couples c ON c.id = cm.couple_id
		WHERE cm.user_id = $1 AND cm.left_at IS NULL AND c.status = 'active'
		LIMIT 1
	`, userID).Scan(&coupleID)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, ErrNoActiveCouple
	}
	return coupleID, err
}

func (r *Repository) GetPartnerID(ctx context.Context, userID uuid.UUID) (uuid.UUID, error) {
	var partnerID uuid.UUID
	err := r.db.QueryRow(ctx, `
		SELECT m2.user_id
		FROM couple_members cm
		JOIN couple_members m2 ON m2.couple_id = cm.couple_id AND m2.user_id != cm.user_id
		JOIN couples c ON c.id = cm.couple_id
		WHERE cm.user_id = $1 AND cm.left_at IS NULL AND c.status = 'active'
		LIMIT 1
	`, userID).Scan(&partnerID)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, ErrNoActiveCouple
	}
	return partnerID, err
}

func (r *Repository) GetSharingStatus(ctx context.Context, userID, coupleID uuid.UUID) (SharingStatus, error) {
	var s SharingStatus
	err := r.db.QueryRow(ctx, `
		SELECT is_active, last_sharing_at, updated_at
		FROM location_sharing
		WHERE user_id = $1 AND couple_id = $2
	`, userID, coupleID).Scan(&s.IsActive, &s.LastSharingAt, &s.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return SharingStatus{UpdatedAt: time.Now().UTC()}, nil
	}
	return s, err
}

func (r *Repository) UpsertSharing(ctx context.Context, userID, coupleID uuid.UUID, isActive bool) (SharingStatus, error) {
	var s SharingStatus
	var lastSharingAt *time.Time
	err := r.db.QueryRow(ctx, `
		INSERT INTO location_sharing (user_id, couple_id, is_active, last_sharing_at, updated_at)
		VALUES ($1, $2, $3, NOW(), NOW())
		ON CONFLICT (couple_id, user_id) DO UPDATE SET
			is_active = EXCLUDED.is_active,
			last_sharing_at = CASE WHEN EXCLUDED.is_active THEN NOW() ELSE location_sharing.last_sharing_at END,
			updated_at = NOW()
		RETURNING is_active, last_sharing_at, updated_at
	`, userID, coupleID, isActive).Scan(&s.IsActive, &lastSharingAt, &s.UpdatedAt)
	if err != nil {
		return SharingStatus{}, fmt.Errorf("upsert sharing: %w", err)
	}
	s.LastSharingAt = lastSharingAt
	return s, nil
}

func (r *Repository) InsertPosition(ctx context.Context, userID, coupleID uuid.UUID, lat, lon float64, accuracy *float64) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO location_positions (user_id, couple_id, latitude, longitude, accuracy)
		VALUES ($1, $2, $3, $4, $5)
	`, userID, coupleID, lat, lon, accuracy)
	return err
}

func (r *Repository) GetPartnerLastPosition(ctx context.Context, partnerID, coupleID uuid.UUID) (Position, error) {
	var pos Position
	var accuracy *float64
	err := r.db.QueryRow(ctx, `
		SELECT latitude, longitude, accuracy, captured_at
		FROM location_positions
		WHERE user_id = $1 AND couple_id = $2
		ORDER BY captured_at DESC
		LIMIT 1
	`, partnerID, coupleID).Scan(&pos.Latitude, &pos.Longitude, &accuracy, &pos.CapturedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Position{}, nil
	}
	if err != nil {
		return Position{}, fmt.Errorf("get partner position: %w", err)
	}
	pos.UserID = partnerID.String()
	pos.Accuracy = accuracy
	return pos, nil
}

func (r *Repository) ListGeofences(ctx context.Context, coupleID uuid.UUID) ([]Geofence, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, name, latitude, longitude, radius_meters, created_at
		FROM location_geofences
		WHERE couple_id = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC
	`, coupleID)
	if err != nil {
		return nil, fmt.Errorf("list geofences: %w", err)
	}
	defer rows.Close()

	var geofences []Geofence
	for rows.Next() {
		var g Geofence
		if err := rows.Scan(&g.ID, &g.Name, &g.Latitude, &g.Longitude,
			&g.RadiusMeters, &g.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan geofence: %w", err)
		}
		geofences = append(geofences, g)
	}
	return geofences, rows.Err()
}

func (r *Repository) CreateGeofence(ctx context.Context, coupleID, userID uuid.UUID, req CreateGeofenceRequest) (Geofence, error) {
	var g Geofence
	var id uuid.UUID
	err := r.db.QueryRow(ctx, `
		INSERT INTO location_geofences
			(couple_id, user_id, name, latitude, longitude, radius_meters)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at
	`, coupleID, userID, req.Name, req.Latitude, req.Longitude, req.RadiusMeters).Scan(&id, &g.CreatedAt)
	if err != nil {
		return Geofence{}, fmt.Errorf("create geofence: %w", err)
	}
	g.ID = id.String()
	g.Name = req.Name
	g.Latitude = req.Latitude
	g.Longitude = req.Longitude
	g.RadiusMeters = req.RadiusMeters
	return g, nil
}
