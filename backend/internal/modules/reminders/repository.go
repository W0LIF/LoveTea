package reminders

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

// UpcomingEvent is a calendar event that needs a reminder in the next window.
type UpcomingEvent struct {
	ID                 uuid.UUID
	OwnerUserID        uuid.UUID
	CoupleID           *uuid.UUID
	Title              string
	StartsAt           time.Time
	RecurrenceRule     *string
	ReminderMinutes    int
}

// FindUpcoming returns events whose reminder time falls within [now, now+window].
// It skips events that already have a reminder_log entry for the same start.
func (r *Repository) FindUpcoming(ctx context.Context, now time.Time, window time.Duration) ([]UpcomingEvent, error) {
	upper := now.Add(window)
	rows, err := r.db.Query(ctx, `
		SELECT e.id, e.owner_user_id, e.couple_id, e.title, e.starts_at, e.recurrence_rule,
		       COALESCE(re.reminder_minutes_before, 30) AS reminder_minutes
		FROM calendar_events e
		LEFT JOIN reminder_events re ON re.calendar_event_id = e.id AND re.sent_at IS NULL
		WHERE e.deleted_at IS NULL
		  AND e.starts_at > $1
		  AND e.starts_at <= $2
		  AND (e.recurrence_rule IS NULL OR e.recurrence_rule <> 'NONE')
		ORDER BY e.starts_at ASC
	`, now.Add(-24*time.Hour), upper)
	if err != nil {
		return nil, fmt.Errorf("reminders.FindUpcoming: %w", err)
	}
	defer rows.Close()

	var out []UpcomingEvent
	for rows.Next() {
		var ev UpcomingEvent
		if err := rows.Scan(
			&ev.ID, &ev.OwnerUserID, &ev.CoupleID, &ev.Title,
			&ev.StartsAt, &ev.RecurrenceRule, &ev.ReminderMinutes,
		); err != nil {
			return nil, fmt.Errorf("reminders.FindUpcoming scan: %w", err)
		}
		out = append(out, ev)
	}
	return out, rows.Err()
}

// AlreadySent reports whether a reminder for this event start was already logged.
func (r *Repository) AlreadySent(ctx context.Context, eventID uuid.UUID, startsAt time.Time) (bool, error) {
	var n int
	err := r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM reminder_log WHERE event_id = $1 AND sent_at >= $2 AND sent_at <= $2 + interval '1 minute'`,
		eventID, startsAt.Add(-time.Duration(5)*time.Minute),
	).Scan(&n)
	if err != nil {
		return false, fmt.Errorf("reminders.AlreadySent: %w", err)
	}
	return n > 0, nil
}

// Notify inserts an in-app notification for the user.
func (r *Repository) Notify(ctx context.Context, userID uuid.UUID, title, body string) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO notifications (user_id, category, title, body, data)
		 VALUES ($1, 'reminder', $2, $3, '{}'::jsonb)`,
		userID, title, body,
	)
	if err != nil {
		return fmt.Errorf("reminders.Notify: %w", err)
	}
	return nil
}

// Log records a reminder send so it is not repeated.
func (r *Repository) Log(ctx context.Context, eventID, userID uuid.UUID, channel string) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO reminder_log (event_id, user_id, channel) VALUES ($1, $2, $3)`,
		eventID, userID, channel,
	)
	if err != nil {
		return fmt.Errorf("reminders.Log: %w", err)
	}
	return nil
}

// MarkReminderSent updates reminder_events.sent_at.
func (r *Repository) MarkReminderSent(ctx context.Context, eventID uuid.UUID, at time.Time) error {
	_, err := r.db.Exec(ctx,
		`UPDATE reminder_events SET sent_at = $2 WHERE calendar_event_id = $1 AND sent_at IS NULL`,
		eventID, at,
	)
	if err != nil {
		return fmt.Errorf("reminders.MarkReminderSent: %w", err)
	}
	return nil
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
		return uuid.Nil, nil
	}
	if err != nil {
		return uuid.Nil, fmt.Errorf("reminders.GetActiveCouple: %w", err)
	}
	return coupleID, nil
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

// GetPartnerID returns the partner user ID in the user's active couple.
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
