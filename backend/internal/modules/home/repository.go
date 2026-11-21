package home

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrPetStateNotFound = errors.New("couple pet state not found")

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) GetPetState(
	ctx context.Context,
	coupleID uuid.UUID,
) (petRecord, error) {
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
		return petRecord{}, ErrPetStateNotFound
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
