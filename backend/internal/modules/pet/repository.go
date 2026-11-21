package pet

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// GetActiveCoupleID returns the active couple ID for a user, or uuid.Nil if none.
func (r *Repository) GetActiveCoupleID(ctx context.Context, userID uuid.UUID) (uuid.UUID, error) {
	var coupleID uuid.UUID
	err := r.db.QueryRow(ctx, `
		SELECT cm.couple_id
		FROM couple_members cm
		JOIN couples c ON c.id = cm.couple_id
		WHERE cm.user_id = $1 AND cm.left_at IS NULL AND c.status = 'active'
		LIMIT 1
	`, userID).Scan(&coupleID)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, nil
	}
	if err != nil {
		return uuid.Nil, fmt.Errorf("get active couple: %w", err)
	}
	return coupleID, nil
}

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) GetPetState(ctx context.Context, coupleID uuid.UUID) (petRecord, error) {
	var (
		pet        petRecord
		appearance []byte
	)
	err := r.db.QueryRow(ctx, `
		SELECT appearance, version, updated_at
		FROM couple_pet_state
		WHERE couple_id = $1
	`, coupleID).Scan(&appearance, &pet.Version, &pet.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return petRecord{}, sql.ErrNoRows
	}
	if err != nil {
		return petRecord{}, fmt.Errorf("get couple pet state: %w", err)
	}
	if !json.Valid(appearance) {
		return petRecord{}, errors.New("stored pet appearance is invalid JSON")
	}
	pet.Appearance = append(json.RawMessage(nil), appearance...)
	return pet, nil
}

func (r *Repository) UpsertPetState(
	ctx context.Context,
	coupleID uuid.UUID,
	appearance json.RawMessage,
) (petRecord, error) {
	var pet petRecord
	err := r.db.QueryRow(ctx, `
		INSERT INTO couple_pet_state (couple_id, appearance, version, updated_at)
		VALUES ($1, $2, 1, NOW())
		ON CONFLICT (couple_id)
		DO UPDATE SET
			appearance = EXCLUDED.appearance,
			version = couple_pet_state.version + 1,
			updated_at = NOW()
		RETURNING appearance, version, updated_at
	`, coupleID, []byte(appearance)).Scan(&pet.Appearance, &pet.Version, &pet.UpdatedAt)
	if err != nil {
		return petRecord{}, fmt.Errorf("upsert couple pet state: %w", err)
	}
	return pet, nil
}

func (r *Repository) EnsurePetState(ctx context.Context, coupleID uuid.UUID) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO couple_pet_state (couple_id, appearance, version, updated_at)
		VALUES ($1, '{}'::jsonb, 1, NOW())
		ON CONFLICT (couple_id) DO NOTHING
	`, coupleID)
	if err != nil {
		return fmt.Errorf("ensure couple pet state: %w", err)
	}
	return nil
}
