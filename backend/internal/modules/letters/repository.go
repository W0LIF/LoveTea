package letters

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	syncmodule "github.com/W0LIF/LoveTea/backend/internal/modules/sync"
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

func (r *Repository) GetActiveCoupleAndPartner(
	ctx context.Context,
	userID uuid.UUID,
) (coupleID, partnerID uuid.UUID, err error) {
	err = r.db.QueryRow(ctx, `
		SELECT cm.couple_id,
		       CASE WHEN cm.user_id = $1 THEN m2.user_id ELSE cm.user_id END
		FROM couple_members cm
		JOIN couple_members m2 ON m2.couple_id = cm.couple_id AND m2.user_id != cm.user_id
		JOIN couples c ON c.id = cm.couple_id
		WHERE cm.user_id = $1 OR m2.user_id = $1
		  AND cm.left_at IS NULL AND c.status = 'active'
		LIMIT 1
	`, userID).Scan(&coupleID, &partnerID)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, uuid.Nil, ErrNoActiveCouple
	}
	if err != nil {
		return uuid.Nil, uuid.Nil, fmt.Errorf("get couple partner: %w", err)
	}
	return coupleID, partnerID, nil
}

func (r *Repository) Create(
	ctx context.Context,
	senderID, recipientID, coupleID uuid.UUID,
	content string,
	cooldown time.Duration,
) (Letter, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return Letter{}, err
	}
	defer tx.Rollback(ctx)

	cooldownUntil := time.Now().Add(cooldown)

	var letterID uuid.UUID
	err = tx.QueryRow(ctx, `
		INSERT INTO letters (sender_user_id, recipient_user_id, couple_id, content,
		                     cooldown_until, status, version)
		VALUES ($1, $2, $3, $4, $5, 'waiting', 1)
		RETURNING id
	`, senderID, recipientID, coupleID, content, cooldownUntil).Scan(&letterID)
	if err != nil {
		return Letter{}, fmt.Errorf("insert letter: %w", err)
	}

	// Sync event
	payload, _ := json.Marshal(map[string]any{
		"sender_id":       senderID.String(),
		"recipient_id":    recipientID.String(),
		"couple_id":       coupleID.String(),
		"content":         content,
		"cooldown_until":  cooldownUntil.UTC().Format(time.RFC3339),
	})
	if _, err := syncmodule.RecordChange(ctx, tx, syncmodule.NewChange{
		Scope:      syncmodule.Scope{CoupleID: &coupleID},
		ActorID:    senderID,
		EntityType: "letter",
		EntityID:   letterID,
		Operation:  "upsert",
		Payload:    payload,
	}); err != nil {
		return Letter{}, fmt.Errorf("record sync: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return Letter{}, fmt.Errorf("commit: %w", err)
	}

	return Letter{
		ID:            letterID.String(),
		SenderID:      senderID.String(),
		RecipientID:   recipientID.String(),
		Content:       content,
		Status:        "waiting",
		CooldownUntil: cooldownUntil,
		CreatedAt:     time.Now().UTC(),
	}, nil
}

func (r *Repository) ListBySender(ctx context.Context, userID uuid.UUID) ([]Letter, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, sender_user_id, recipient_user_id, content, status,
		       cooldown_until, version, created_at, sent_at, read_at
		FROM letters
		WHERE sender_user_id = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC
		LIMIT 50
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("query sent letters: %w", err)
	}
	defer rows.Close()

	var letters []Letter
	for rows.Next() {
		var letter Letter
		if err := rows.Scan(&letter.ID, &letter.SenderID, &letter.RecipientID,
			&letter.Content, &letter.Status, &letter.CooldownUntil,
			&letter.Version, &letter.CreatedAt, &letter.SentAt, &letter.ReadAt); err != nil {
			return nil, fmt.Errorf("scan letter: %w", err)
		}
		letters = append(letters, letter)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate: %w", err)
	}
	return letters, nil
}

func (r *Repository) ListByRecipient(ctx context.Context, userID uuid.UUID) ([]Letter, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, sender_user_id, recipient_user_id, content, status,
		       cooldown_until, version, created_at, sent_at, read_at
		FROM letters
		WHERE recipient_user_id = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC
		LIMIT 50
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("query received letters: %w", err)
	}
	defer rows.Close()

	var letters []Letter
	for rows.Next() {
		var letter Letter
		if err := rows.Scan(&letter.ID, &letter.SenderID, &letter.RecipientID,
			&letter.Content, &letter.Status, &letter.CooldownUntil,
			&letter.Version, &letter.CreatedAt, &letter.SentAt, &letter.ReadAt); err != nil {
			return nil, fmt.Errorf("scan letter: %w", err)
		}
		letters = append(letters, letter)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate: %w", err)
	}
	return letters, nil
}

func (r *Repository) GetByID(ctx context.Context, letterID, userID uuid.UUID) (Letter, error) {
	var letter Letter
	err := r.db.QueryRow(ctx, `
		SELECT id, sender_user_id, recipient_user_id, content, status,
		       cooldown_until, version, created_at, sent_at, read_at
		FROM letters
		WHERE id = $1 AND deleted_at IS NULL
		  AND (sender_user_id = $2 OR recipient_user_id = $2)
	`, letterID, userID).Scan(
		&letter.ID, &letter.SenderID, &letter.RecipientID,
		&letter.Content, &letter.Status, &letter.CooldownUntil,
		&letter.Version, &letter.CreatedAt, &letter.SentAt, &letter.ReadAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Letter{}, ErrNotFound
	}
	if err != nil {
		return Letter{}, fmt.Errorf("get letter: %w", err)
	}
	return letter, nil
}

func (r *Repository) MarkRead(ctx context.Context, letterID, recipientID uuid.UUID) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE letters
		SET status = 'read', read_at = NOW(), version = version + 1
		WHERE id = $1 AND recipient_user_id = $2 AND status = 'sent'
	`, letterID, recipientID)
	if err != nil {
		return fmt.Errorf("mark read: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repository) Archive(ctx context.Context, letterID, userID uuid.UUID, version int64) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE letters
		SET status = 'archived', version = $3, deleted_at = NOW()
		WHERE id = $1 AND (sender_user_id = $2 OR recipient_user_id = $2)
		  AND version = $3 AND deleted_at IS NULL
	`, letterID, userID, version)
	if err != nil {
		return fmt.Errorf("archive: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repository) GetLastCooldown(ctx context.Context, userID uuid.UUID) (time.Time, error) {
	var last time.Time
	err := r.db.QueryRow(ctx, `
		SELECT MAX(created_at) FROM letters
		WHERE sender_user_id = $1 AND status IN ('waiting', 'sent', 'read', 'archived')
	`, userID).Scan(&last)
	if errors.Is(err, pgx.ErrNoRows) {
		return time.Time{}, nil
	}
	if err != nil {
		return time.Time{}, fmt.Errorf("get last cooldown: %w", err)
	}
	return last, nil
}
