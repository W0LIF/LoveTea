package wheel

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNoActiveCouple = errors.New("no active couple")
	ErrInvalidInput   = errors.New("invalid input")
	ErrNotFound       = errors.New("not found")
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

func (r *Repository) ListSectors(ctx context.Context, coupleID uuid.UUID) ([]Sector, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, title, description, requires_confirmation, position, created_at
		FROM wheel_sectors
		WHERE couple_id = $1 AND deleted_at IS NULL
		ORDER BY position ASC
	`, coupleID)
	if err != nil {
		return nil, fmt.Errorf("list sectors: %w", err)
	}
	defer rows.Close()

	var sectors []Sector
	for rows.Next() {
		var s Sector
		if err := rows.Scan(&s.ID, &s.Title, &s.Description,
			&s.RequiresConfirmation, &s.Position, &s.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan sector: %w", err)
		}
		sectors = append(sectors, s)
	}
	return sectors, rows.Err()
}

func (r *Repository) CreateSector(ctx context.Context, coupleID uuid.UUID, req CreateSectorRequest) (Sector, error) {
	var s Sector
	var id uuid.UUID
	err := r.db.QueryRow(ctx, `
		INSERT INTO wheel_sectors
			(couple_id, title, description, requires_confirmation, position)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at
	`, coupleID, req.Title, req.Description, req.RequiresConfirmation, req.Position).Scan(&id, &s.CreatedAt)
	if err != nil {
		return Sector{}, fmt.Errorf("create sector: %w", err)
	}
	s.ID = id.String()
	s.Title = req.Title
	s.Description = req.Description
	s.RequiresConfirmation = req.RequiresConfirmation
	s.Position = req.Position
	return s, nil
}

func (r *Repository) DeleteSector(ctx context.Context, sectorID, coupleID uuid.UUID) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE wheel_sectors SET deleted_at = NOW()
		WHERE id = $1 AND couple_id = $2 AND deleted_at IS NULL
	`, sectorID, coupleID)
	if err != nil {
		return fmt.Errorf("delete sector: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repository) Spin(ctx context.Context, coupleID, userID uuid.UUID) (Spin, error) {
	// Выбираем случайный сектор.
	var sectors []Sector
	rows, err := r.db.Query(ctx, `
		SELECT id, title, description, requires_confirmation, position, created_at
		FROM wheel_sectors
		WHERE couple_id = $1 AND deleted_at IS NULL
	`, coupleID)
	if err != nil {
		return Spin{}, fmt.Errorf("query sectors for spin: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var s Sector
		if err := rows.Scan(&s.ID, &s.Title, &s.Description,
			&s.RequiresConfirmation, &s.Position, &s.CreatedAt); err != nil {
			return Spin{}, fmt.Errorf("scan sector: %w", err)
		}
		sectors = append(sectors, s)
	}
	if len(sectors) == 0 {
		return Spin{}, ErrNotFound
	}

	selected := sectors[rand.Intn(len(sectors))]
	var sectorID *uuid.UUID
	parsed, _ := uuid.Parse(selected.ID)
	sectorID = &parsed

	var spin Spin
	var id uuid.UUID
	err = r.db.QueryRow(ctx, `
		INSERT INTO wheel_spins (couple_id, spun_by_user_id, selected_sector_id)
		VALUES ($1, $2, $3)
		RETURNING id, created_at
	`, coupleID, userID, sectorID).Scan(&id, &spin.CreatedAt)
	if err != nil {
		return Spin{}, fmt.Errorf("insert spin: %w", err)
	}
	spin.ID = id.String()
	spin.SectorID = &selected.ID
	spin.SpunBy = userID.String()
	return spin, nil
}

func (r *Repository) Confirm(ctx context.Context, spinID, userID uuid.UUID) error {
	_, err := r.db.Exec(ctx, `
		UPDATE wheel_spins SET confirmed_by_both = true, confirmed_at = NOW()
		WHERE id = $1 AND couple_id = (
			SELECT cm.couple_id FROM couple_members cm
			JOIN couples c ON c.id = cm.couple_id
			WHERE cm.user_id = $2 AND cm.left_at IS NULL AND c.status = 'active'
			LIMIT 1
		) AND deleted_at IS NULL
	`, spinID, userID)
	return err
}

func (r *Repository) History(ctx context.Context, coupleID uuid.UUID) ([]Spin, error) {
	rows, err := r.db.Query(ctx, `
		SELECT ws.id, ws.selected_sector_id::TEXT, ws.spun_by_user_id,
		       ws.confirmed_by_both, ws.cancelled, ws.created_at, ws.confirmed_at
		FROM wheel_spins ws
		WHERE ws.couple_id = $1 AND ws.deleted_at IS NULL
		ORDER BY ws.created_at DESC
		LIMIT 50
	`, coupleID)
	if err != nil {
		return nil, fmt.Errorf("history: %w", err)
	}
	defer rows.Close()

	var spins []Spin
	for rows.Next() {
		var s Spin
		if err := rows.Scan(&s.ID, &s.SectorID, &s.SpunBy,
			&s.Confirmed, &s.Cancelled, &s.CreatedAt, &s.ConfirmedAt); err != nil {
			return nil, fmt.Errorf("scan spin: %w", err)
		}
		spins = append(spins, s)
	}
	return spins, rows.Err()
}

var _ = time.Now
