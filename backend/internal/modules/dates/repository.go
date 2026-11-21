package dates

import (
	"context"
	"errors"
	"fmt"

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

// ------------------------------------------------------------------
// Date ideas
// ------------------------------------------------------------------

func (r *Repository) CreateIdea(ctx context.Context, tx pgx.Tx, idea *DateIdea) error {
	_, err := tx.Exec(ctx,
		`INSERT INTO date_ideas
		   (id, couple_id, proposer_user_id, title, description, category, budget_min, budget_max, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 'suggested')`,
		idea.ID, idea.CoupleID, idea.ProposerUserID, idea.Title, idea.Details,
		idea.Category, idea.BudgetMin, idea.BudgetMax,
	)
	if err != nil {
		return fmt.Errorf("dates.CreateIdea: %w", err)
	}
	return nil
}

func (r *Repository) ListIdeas(ctx context.Context, coupleID uuid.UUID, status string) ([]DateIdea, error) {
	query := `SELECT id, couple_id, proposer_user_id, title, description, category,
		             budget_min, budget_max, status, created_at, accepted_at, completed_at
		      FROM date_ideas
		     WHERE couple_id = $1 AND deleted_at IS NULL`
	args := []any{coupleID}
	argIdx := 2

	if status != "" {
		query += fmt.Sprintf(" AND status = $%d", argIdx)
		args = append(args, status)
		argIdx++
	}

	query += ` ORDER BY created_at DESC`

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("dates.ListIdeas: %w", err)
	}
	defer rows.Close()

	return pgx.CollectRows(rows, pgx.RowToStructByName[DateIdea])
}

func (r *Repository) GetIdea(ctx context.Context, id uuid.UUID) (DateIdea, error) {
	var idea DateIdea
	err := r.db.QueryRow(ctx,
		`SELECT id, couple_id, proposer_user_id, title, description, category,
		        budget_min, budget_max, status, created_at, accepted_at, completed_at
		 FROM date_ideas
		WHERE id = $1 AND deleted_at IS NULL`,
		id,
	).Scan(&idea.ID, &idea.CoupleID, &idea.ProposerUserID, &idea.Title, &idea.Details,
		&idea.Category, &idea.BudgetMin, &idea.BudgetMax, &idea.Status,
		&idea.CreatedAt, &idea.AcceptedAt, &idea.CompletedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return DateIdea{}, ErrNotFound
	}
	if err != nil {
		return DateIdea{}, fmt.Errorf("dates.GetIdea: %w", err)
	}
	return idea, nil
}

func (r *Repository) UpdateIdeaStatus(ctx context.Context, tx pgx.Tx, id uuid.UUID, status string) error {
	_, err := tx.Exec(ctx,
		`UPDATE date_ideas SET status = $1, accepted_at = NOW() WHERE id = $2 AND deleted_at IS NULL`,
		status, id,
	)
	if err != nil {
		return fmt.Errorf("dates.UpdateIdeaStatus: %w", err)
	}
	return nil
}

// ------------------------------------------------------------------
// Date recommendations
// ------------------------------------------------------------------

func (r *Repository) UpsertRecommendation(ctx context.Context, tx pgx.Tx, rec *DateRecommendation) error {
	_, err := tx.Exec(ctx,
		`INSERT INTO date_recommendations
		   (id, user_id, couple_id, title, description, category, suggested_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT DO NOTHING`,
		rec.ID, rec.UserID, rec.CoupleID, rec.Title, rec.Description,
		rec.Category, rec.SuggestedAt,
	)
	if err != nil {
		return fmt.Errorf("dates.UpsertRecommendation: %w", err)
	}
	return nil
}

func (r *Repository) ListRecommendations(ctx context.Context, userID, coupleID uuid.UUID, limit int) ([]DateRecommendation, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	rows, err := r.db.Query(ctx,
		`SELECT id, user_id, couple_id, title, description, category, suggested_at
		 FROM date_recommendations
		WHERE user_id = $1 AND couple_id = $2
		ORDER BY suggested_at DESC
		   LIMIT $3`,
		userID, coupleID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("dates.ListRecommendations: %w", err)
	}
	defer rows.Close()

	return pgx.CollectRows(rows, pgx.RowToStructByName[DateRecommendation])
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
		return uuid.Nil, fmt.Errorf("dates.GetActiveCouple: %w", err)
	}
	return coupleID, nil
}
