package memories

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, m *Memory) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO memories
		   (id, owner_user_id, couple_id, title, content, content_encrypted,
		    ciphertext, cipher_nonce, cipher_key_ref, version, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, 1, $10, $10)`,
		m.ID, m.OwnerUserID, m.CoupleID, m.Title, m.Content, m.ContentEncrypted,
		m.Ciphertext, m.CipherNonce, m.CipherKeyRef, m.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("memories.Create: %w", err)
	}
	return nil
}

func (r *Repository) List(ctx context.Context, userID uuid.UUID, f ListFilter) ([]Memory, error) {
	query := `SELECT id, owner_user_id, couple_id, title, content, content_encrypted,
		             ciphertext, cipher_nonce, cipher_key_ref, version, created_at, updated_at
		      FROM memories
		     WHERE deleted_at IS NULL`
	args := []any{}
	argIdx := 1

	if f.CoupleID != nil {
		query += fmt.Sprintf(" AND couple_id = $%d", argIdx)
		args = append(args, *f.CoupleID)
		argIdx++
	} else {
		query += fmt.Sprintf(" AND owner_user_id = $%d AND couple_id IS NULL", argIdx)
		args = append(args, userID)
		argIdx++
	}

	query += " ORDER BY created_at DESC"

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("memories.List: %w", err)
	}
	defer rows.Close()

	return pgx.CollectRows(rows, pgx.RowToStructByName[Memory])
}

func (r *Repository) Get(ctx context.Context, id uuid.UUID) (Memory, error) {
	var m Memory
	err := r.db.QueryRow(ctx,
		`SELECT id, owner_user_id, couple_id, title, content, content_encrypted,
		        ciphertext, cipher_nonce, cipher_key_ref, version, created_at, updated_at
		 FROM memories
		WHERE id = $1 AND deleted_at IS NULL`,
		id,
	).Scan(&m.ID, &m.OwnerUserID, &m.CoupleID, &m.Title, &m.Content, &m.ContentEncrypted,
		&m.Ciphertext, &m.CipherNonce, &m.CipherKeyRef, &m.Version, &m.CreatedAt, &m.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Memory{}, ErrNotFound
	}
	if err != nil {
		return Memory{}, fmt.Errorf("memories.Get: %w", err)
	}
	return m, nil
}

func (r *Repository) Update(ctx context.Context, id uuid.UUID, title, content string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE memories SET title = $1, content = $2, version = version + 1, updated_at = NOW()
		 WHERE id = $3 AND deleted_at IS NULL`,
		title, content, id,
	)
	if err != nil {
		return fmt.Errorf("memories.Update: %w", err)
	}
	return nil
}

func (r *Repository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`UPDATE memories SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL`,
		id,
	)
	if err != nil {
		return fmt.Errorf("memories.Delete: %w", err)
	}
	return nil
}

func (r *Repository) GetActiveCouple(ctx context.Context, userID uuid.UUID) (uuid.UUID, error) {
	var coupleID uuid.UUID
	err := r.db.QueryRow(ctx,
		`SELECT cm.couple_id
		 FROM couple_members cm
		 JOIN couples c ON c.id = cm.couple_id
		WHERE cm.user_id = $1 AND cm.left_at IS NULL AND c.status = 'active'
		LIMIT 1`,
		userID,
	).Scan(&coupleID)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, ErrNoCouple
	}
	if err != nil {
		return uuid.Nil, fmt.Errorf("memories.GetActiveCouple: %w", err)
	}
	return coupleID, nil
}

func now() time.Time { return time.Now().UTC() }
