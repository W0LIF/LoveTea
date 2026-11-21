package account

import (
	"context"
	"encoding/base64"
	"encoding/json"
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

func (r *Repository) CreateExport(ctx context.Context, userID uuid.UUID) (Export, error) {
	e := Export{
		ID:        uuid.New(),
		UserID:    userID,
		Status:    "pending",
		ExpiresAt: time.Now().UTC().Add(24 * time.Hour),
		CreatedAt: time.Now().UTC(),
	}
	_, err := r.db.Exec(ctx,
		`INSERT INTO account_exports (id, user_id, status, expires_at, created_at)
		 VALUES ($1, $2, $3, $4, $5)`,
		e.ID, e.UserID, e.Status, e.ExpiresAt, e.CreatedAt,
	)
	if err != nil {
		return Export{}, fmt.Errorf("account.CreateExport: %w", err)
	}
	return e, nil
}

func (r *Repository) MarkExportReady(ctx context.Context, id uuid.UUID, fileURL string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE account_exports SET status = 'ready', file_url = $2, completed_at = NOW()
		 WHERE id = $1`,
		id, fileURL,
	)
	if err != nil {
		return fmt.Errorf("account.MarkExportReady: %w", err)
	}
	return nil
}

func (r *Repository) GetExport(ctx context.Context, id, userID uuid.UUID) (Export, error) {
	var e Export
	err := r.db.QueryRow(ctx,
		`SELECT id, user_id, status, file_url, expires_at, created_at, completed_at
		 FROM account_exports WHERE id = $1 AND user_id = $2`,
		id, userID,
	).Scan(&e.ID, &e.UserID, &e.Status, &e.FileURL, &e.ExpiresAt, &e.CreatedAt, &e.CompletedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Export{}, ErrNotFound
	}
	if err != nil {
		return Export{}, fmt.Errorf("account.GetExport: %w", err)
	}
	return e, nil
}

// GetActiveCouple returns the couple the user currently belongs to, if any.
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
		return uuid.Nil, fmt.Errorf("account.GetActiveCouple: %w", err)
	}
	return coupleID, nil
}

// GetProfile returns the user profile row as a map for export.
func (r *Repository) GetProfile(ctx context.Context, userID uuid.UUID) (map[string]any, error) {
	var (
		nickname  *string
		bio       *string
		avatarURL *string
		birthDate *time.Time
		timezone  *string
	)
	err := r.db.QueryRow(ctx,
		`SELECT nickname, bio, avatar_url, birth_date, timezone
		 FROM user_profiles WHERE user_id = $1`,
		userID,
	).Scan(&nickname, &bio, &avatarURL, &birthDate, &timezone)
	if errors.Is(err, pgx.ErrNoRows) {
		return map[string]any{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("account.GetProfile: %w", err)
	}
	out := map[string]any{}
	if nickname != nil {
		out["nickname"] = *nickname
	}
	if bio != nil {
		out["bio"] = *bio
	}
	if avatarURL != nil {
		out["avatar_url"] = *avatarURL
	}
	if birthDate != nil {
		out["birth_date"] = birthDate.UTC().Format(time.RFC3339)
	}
	if timezone != nil {
		out["timezone"] = *timezone
	}
	return out, nil
}

// GetUser returns the user email and created_at.
func (r *Repository) GetUser(ctx context.Context, userID uuid.UUID) (map[string]any, error) {
	var (
		email     string
		createdAt time.Time
	)
	err := r.db.QueryRow(ctx,
		`SELECT email, created_at FROM users WHERE id = $1`, userID,
	).Scan(&email, &createdAt)
	if err != nil {
		return nil, fmt.Errorf("account.GetUser: %w", err)
	}
	return map[string]any{
		"id":         userID.String(),
		"email":      email,
		"created_at": createdAt.UTC().Format(time.RFC3339),
	}, nil
}

// tableQuery describes a personal table to dump for export.
type tableQuery struct {
	key   string
	query string
	cols  []string
}

// ExportPersonalData dumps all personal (owner_user_id = userID) rows.
func (r *Repository) ExportPersonalData(ctx context.Context, userID uuid.UUID) (map[string]any, error) {
	queries := []tableQuery{
		{"habits", `SELECT id, title, frequency, created_at FROM habits WHERE owner_user_id = $1`, []string{"id", "title", "frequency", "created_at"}},
		{"mood_entries", `SELECT id, mood, energy_level, note, created_at FROM mood_entries WHERE user_id = $1 AND shared_couple_id IS NULL`, []string{"id", "mood", "energy_level", "note", "created_at"}},
		{"sleep_entries", `SELECT id, duration_minutes, quality, note, created_at FROM sleep_entries WHERE user_id = $1 AND shared_couple_id IS NULL`, []string{"id", "duration_minutes", "quality", "note", "created_at"}},
		{"tasks", `SELECT id, title, status, due_date, created_at FROM tasks WHERE owner_user_id = $1`, []string{"id", "title", "status", "due_date", "created_at"}},
		{"calendar_events", `SELECT id, title, starts_at, ends_at, created_at FROM calendar_events WHERE owner_user_id = $1`, []string{"id", "title", "starts_at", "ends_at", "created_at"}},
		{"memories", `SELECT id, title, content, content_encrypted, created_at FROM memories WHERE owner_user_id = $1 AND couple_id IS NULL`, []string{"id", "title", "content", "content_encrypted", "created_at"}},
		{"notifications", `SELECT id, category, title, body, is_read, created_at FROM notifications WHERE user_id = $1`, []string{"id", "category", "title", "body", "is_read", "created_at"}},
		{"date_recommendations", `SELECT id, title, description, category, suggested_at FROM date_recommendations WHERE user_id = $1`, []string{"id", "title", "description", "category", "suggested_at"}},
		{"wheel_spins", `SELECT id, sector_id, confirmed, created_at FROM wheel_spins WHERE spun_by_user_id = $1`, []string{"id", "sector_id", "confirmed", "created_at"}},
		{"location_positions", `SELECT id, latitude, longitude, recorded_at FROM location_positions WHERE user_id = $1`, []string{"id", "latitude", "longitude", "recorded_at"}},
	}

	out := map[string]any{}
	for _, q := range queries {
		rows, err := r.db.Query(ctx, q.query, userID)
		if err != nil {
			// Table may not exist in some environments; skip silently.
			continue
		}
		items, err := collectRows(rows, q.cols)
		rows.Close()
		if err != nil {
			return nil, fmt.Errorf("account.ExportPersonalData %s: %w", q.key, err)
		}
		out[q.key] = items
	}
	return out, nil
}

func collectRows(rows pgx.Rows, cols []string) ([]map[string]any, error) {
	var items []map[string]any
	for rows.Next() {
		values := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range values {
			ptrs[i] = &values[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, err
		}
		item := map[string]any{}
		for i, c := range cols {
			item[c] = normalizeValue(values[i])
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func normalizeValue(v any) any {
	switch x := v.(type) {
	case time.Time:
		return x.UTC().Format(time.RFC3339)
	case []byte:
		return string(x)
	default:
		return x
	}
}

// DeleteAccount atomically removes the user and all their data.
func (r *Repository) DeleteAccount(ctx context.Context, userID uuid.UUID) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("account.DeleteAccount begin: %w", err)
	}
	defer tx.Rollback(ctx)

	// Find couples the user belongs to and delete them.
	rows, err := tx.Query(ctx,
		`SELECT couple_id FROM couple_members WHERE user_id = $1`,
		userID,
	)
	if err != nil {
		return fmt.Errorf("account.DeleteAccount find couples: %w", err)
	}
	var coupleIDs []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return fmt.Errorf("account.DeleteAccount scan couple: %w", err)
		}
		coupleIDs = append(coupleIDs, id)
	}
	rows.Close()

	for _, cid := range coupleIDs {
		if _, err := tx.Exec(ctx, `DELETE FROM couples WHERE id = $1`, cid); err != nil {
			return fmt.Errorf("account.DeleteAccount delete couple: %w", err)
		}
	}

	if _, err := tx.Exec(ctx, `DELETE FROM users WHERE id = $1`, userID); err != nil {
		return fmt.Errorf("account.DeleteAccount delete user: %w", err)
	}

	return tx.Commit(ctx)
}

func (r *Repository) SaveExportJSON(ctx context.Context, id uuid.UUID, data []byte) (string, error) {
	// Store the JSON inline; the file_url column holds a data URL for simplicity.
	url := "data:application/json;base64," + base64.StdEncoding.EncodeToString(data)
	if err := r.MarkExportReady(ctx, id, url); err != nil {
		return "", err
	}
	return url, nil
}

// EncodeExport marshals the export payload to JSON.
func EncodeExport(v any) ([]byte, error) {
	return json.MarshalIndent(v, "", "  ")
}
