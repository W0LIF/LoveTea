package sync

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	outboxBatchSize = 100
	outboxLease     = 30 * time.Second
	outboxMaxRetry  = 30 * time.Second
)

type outboxEvent struct {
	ID     int64
	UserID uuid.UUID
	Cursor int64
}

type OutboxDispatcher struct {
	db     *pgxpool.Pool
	hub    *Hub
	logger *slog.Logger
}

func NewOutboxDispatcher(db *pgxpool.Pool, hub *Hub, logger *slog.Logger) *OutboxDispatcher {
	if logger == nil {
		logger = slog.Default()
	}
	return &OutboxDispatcher{db: db, hub: hub, logger: logger}
}

func (d *OutboxDispatcher) Run(ctx context.Context) {
	delay := time.Second
	for ctx.Err() == nil {
		if err := d.dispatch(ctx); err != nil {
			if ctx.Err() == nil {
				d.logger.Error("dispatch realtime outbox events", "error", err)
			}
			delay *= 2
			if delay > outboxMaxRetry {
				delay = outboxMaxRetry
			}
		} else {
			delay = time.Second
		}
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
	}
}

func (d *OutboxDispatcher) dispatch(ctx context.Context) error {
	events, err := claimOutboxEvents(ctx, d.db, outboxBatchSize, outboxLease)
	if err != nil {
		return err
	}
	if len(events) == 0 {
		return nil
	}

	ids := make([]int64, 0, len(events))
	for _, event := range events {
		d.hub.Publish(event.UserID, event.Cursor)
		ids = append(ids, event.ID)
	}
	return completeOutboxEvents(ctx, d.db, ids)
}

func claimOutboxEvents(
	ctx context.Context,
	db *pgxpool.Pool,
	limit int,
	lease time.Duration,
) ([]outboxEvent, error) {
	if limit < 1 || limit > 1000 || lease <= 0 {
		return nil, fmt.Errorf("invalid realtime outbox claim parameters")
	}
	rows, err := db.Query(ctx, `
		WITH claimed AS (
			SELECT id
			FROM realtime_outbox
			WHERE locked_until IS NULL OR locked_until < NOW()
			ORDER BY id
			LIMIT $1
			FOR UPDATE SKIP LOCKED
		)
		UPDATE realtime_outbox AS events
		SET locked_until = NOW() + $2::INTERVAL,
		    attempts = events.attempts + 1
		FROM claimed
		WHERE events.id = claimed.id
		RETURNING events.id, events.recipient_user_id, events.sync_cursor
	`, limit, lease.String())
	if err != nil {
		return nil, fmt.Errorf("claim realtime outbox rows: %w", err)
	}
	defer rows.Close()

	events := make([]outboxEvent, 0)
	for rows.Next() {
		var event outboxEvent
		if err := rows.Scan(&event.ID, &event.UserID, &event.Cursor); err != nil {
			return nil, fmt.Errorf("scan realtime outbox row: %w", err)
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate realtime outbox rows: %w", err)
	}
	return events, nil
}

func completeOutboxEvents(
	ctx context.Context,
	db *pgxpool.Pool,
	ids []int64,
) error {
	if len(ids) == 0 {
		return nil
	}
	if _, err := db.Exec(ctx, `
		DELETE FROM realtime_outbox
		WHERE id = ANY($1::BIGINT[])
	`, ids); err != nil {
		return fmt.Errorf("delete completed realtime outbox rows: %w", err)
	}
	return nil
}
