package reminders

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/W0LIF/LoveTea/backend/internal/platform/push"
	"github.com/google/uuid"
)

// Worker runs the recurring reminder check loop.
type Worker struct {
	repo    *Repository
	logger  *slog.Logger
	interval time.Duration
	window   time.Duration
	pushMgr *push.Manager
}

// NewWorker creates a reminder worker. interval controls how often the check runs.
func NewWorker(repo *Repository, logger *slog.Logger, interval time.Duration, pushMgr *push.Manager) *Worker {
	if logger == nil {
		logger = slog.Default()
	}
	if interval <= 0 {
		interval = 60 * time.Second
	}
	return &Worker{
		repo:     repo,
		logger:   logger,
		interval: interval,
		window:   1 * time.Hour,
		pushMgr:  pushMgr,
	}
}

// Run blocks until ctx is cancelled, ticking every interval.
func (w *Worker) Run(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	w.logger.Info("reminder worker started", "interval", w.interval)
	for {
		select {
		case <-ctx.Done():
			w.logger.Info("reminder worker stopped")
			return
		case <-ticker.C:
			if err := w.tick(ctx); err != nil {
				w.logger.Error("reminder tick failed", "error", err)
			}
		}
	}
}

func (w *Worker) tick(ctx context.Context) error {
	now := time.Now().UTC()
	events, err := w.repo.FindUpcoming(ctx, now, w.window)
	if err != nil {
		return err
	}
	for _, ev := range events {
		reminderTime := ev.StartsAt.Add(-time.Duration(ev.ReminderMinutes) * time.Minute)
		if now.Before(reminderTime) {
			continue
		}
		sent, err := w.repo.AlreadySent(ctx, ev.ID, ev.StartsAt)
		if err != nil {
			return err
		}
		if sent {
			continue
		}
		title := "Upcoming event"
		body := fmt.Sprintf("%s starts at %s", ev.Title, ev.StartsAt.UTC().Format("15:04"))

		// In-app notification
		if err := w.repo.Notify(ctx, ev.OwnerUserID, title, body); err != nil {
			return err
		}

		// Push notification to partner if in a couple
		if ev.CoupleID != nil {
			partnerID, err := w.repo.GetPartnerID(ctx, ev.OwnerUserID)
			if err == nil && partnerID != uuid.Nil {
				tokens, err := w.repo.DeviceTokens(ctx, partnerID)
				if err == nil && len(tokens) > 0 {
					dataMap := map[string]string{
						"type":      "reminder",
						"event_id":  ev.ID.String(),
						"couple_id": ev.CoupleID.String(),
					}
					if err := w.pushMgr.Send(ctx, "android", tokens, title, body, dataMap); err != nil {
						w.logger.Error("push reminder failed", "user_id", partnerID, "error", err)
					}
				}
			}
		}

		if err := w.repo.Log(ctx, ev.ID, ev.OwnerUserID, "push"); err != nil {
			return err
		}
		if err := w.repo.MarkReminderSent(ctx, ev.ID, now); err != nil {
			return err
		}
		w.logger.Info("reminder sent", "event_id", ev.ID, "user_id", ev.OwnerUserID)
	}
	return nil
}

var _ = uuid.Nil
