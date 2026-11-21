package notifications

import (
	"context"
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

func (r *Repository) List(
	ctx context.Context,
	userID uuid.UUID,
	limit int,
	unreadOnly bool,
) ([]Notification, error) {
	query := `
		SELECT id, user_id, category, title, body, data, is_read, created_at
		FROM notifications
		WHERE user_id = $1
	`
	if unreadOnly {
		query += " AND is_read = false"
	}
	query += " ORDER BY created_at DESC LIMIT $2"

	rows, err := r.db.Query(ctx, query, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("list notifications: %w", err)
	}
	defer rows.Close()

	var notifications []Notification
	for rows.Next() {
		var n Notification
		var id int64
		var uid uuid.UUID
		var data []byte
		if err := rows.Scan(&id, &uid, &n.Category, &n.Title, &n.Body,
			&data, &n.IsRead, &n.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan notification: %w", err)
		}
		n.ID = id
		n.UserID = uid.String()
		n.Data = data
		notifications = append(notifications, n)
	}
	return notifications, rows.Err()
}

func (r *Repository) UnreadCount(ctx context.Context, userID uuid.UUID) (int, error) {
	var count int
	err := r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM notifications
		WHERE user_id = $1 AND is_read = false
	`, userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("unread count: %w", err)
	}
	return count, nil
}

func (r *Repository) MarkRead(ctx context.Context, userID uuid.UUID, id int64) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE notifications SET is_read = true
		WHERE id = $1 AND user_id = $2
	`, id, userID)
	if err != nil {
		return fmt.Errorf("mark read: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("notification %d not found for user %s", id, userID)
	}
	return nil
}

func (r *Repository) MarkAllRead(ctx context.Context, userID uuid.UUID) error {
	_, err := r.db.Exec(ctx, `
		UPDATE notifications SET is_read = true
		WHERE user_id = $1 AND is_read = false
	`, userID)
	if err != nil {
		return fmt.Errorf("mark all read: %w", err)
	}
	return nil
}

func (r *Repository) GetSettings(ctx context.Context, userID uuid.UUID) (NotificationSettings, error) {
	var s NotificationSettings
	var uid uuid.UUID
	var data []byte
	err := r.db.QueryRow(ctx, `
		SELECT user_id, enabled_categories, quiet_hours_start, quiet_hours_end, updated_at
		FROM notification_settings
		WHERE user_id = $1
	`, userID).Scan(&uid, &data, &s.QuietHoursStart, &s.QuietHoursEnd, &s.UpdatedAt)
	if err != nil {
		return NotificationSettings{}, err
	}
	s.UserID = uid.String()
	s.EnabledCategories = data
	return s, nil
}

func (r *Repository) UpsertSettings(ctx context.Context, userID uuid.UUID, settings NotificationSettings) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO notification_settings
			(user_id, enabled_categories, quiet_hours_start, quiet_hours_end, updated_at)
		VALUES ($1, $2, $3, $4, NOW())
		ON CONFLICT (user_id) DO UPDATE SET
			enabled_categories = COALESCE(EXCLUDED.enabled_categories, notification_settings.enabled_categories),
			quiet_hours_start = COALESCE(EXCLUDED.quiet_hours_start, notification_settings.quiet_hours_start),
			quiet_hours_end = COALESCE(EXCLUDED.quiet_hours_end, notification_settings.quiet_hours_end),
			updated_at = NOW()
	`, userID, settings.EnabledCategories, settings.QuietHoursStart, settings.QuietHoursEnd)
	return err
}

func (r *Repository) Create(ctx context.Context, n Notification) (Notification, error) {
	var id int64
	err := r.db.QueryRow(ctx, `
		INSERT INTO notifications (user_id, category, title, body, data)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`, n.UserID, n.Category, n.Title, n.Body, n.Data).Scan(&id)
	if err != nil {
		return Notification{}, fmt.Errorf("create notification: %w", err)
	}
	n.ID = id
	n.CreatedAt = time.Now().UTC()
	return n, nil
}

func (r *Repository) GetPartnerID(ctx context.Context, userID uuid.UUID) (uuid.UUID, error) {
	var partnerID uuid.UUID
	err := r.db.QueryRow(ctx, `
		SELECT CASE WHEN cm.user_id = $1 THEN m2.user_id ELSE cm.user_id END
		FROM couple_members cm
		JOIN couple_members m2 ON m2.couple_id = cm.couple_id AND m2.user_id != cm.user_id
		JOIN couples c ON c.id = cm.couple_id
		WHERE (cm.user_id = $1 OR m2.user_id = $1)
		  AND cm.left_at IS NULL AND c.status = 'active'
		LIMIT 1
	`, userID).Scan(&partnerID)
	if err == pgx.ErrNoRows {
		return uuid.Nil, nil
	}
	return partnerID, err
}

// DeviceTokens returns active push tokens for a user.
func (r *Repository) DeviceTokens(ctx context.Context, userID uuid.UUID) ([]string, error) {
	rows, err := r.db.Query(ctx, `
		SELECT push_token FROM user_devices
		WHERE user_id = $1
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("get device tokens: %w", err)
	}
	defer rows.Close()
	var tokens []string
	for rows.Next() {
		var token string
		if err := rows.Scan(&token); err != nil {
			return nil, fmt.Errorf("scan device token: %w", err)
		}
		tokens = append(tokens, token)
	}
	return tokens, rows.Err()
}
