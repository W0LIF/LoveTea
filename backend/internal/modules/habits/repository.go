package habits

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	syncmodule "github.com/W0LIF/LoveTea/backend/internal/modules/sync"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) List(ctx context.Context, userID uuid.UUID) ([]Habit, error) {
	rows, err := r.db.Query(ctx, `
		SELECT h.id,
		       CASE WHEN h.couple_id IS NULL THEN 'personal' ELSE 'couple' END,
		       h.title, h.description, h.target_per_week,
		       h.version, h.created_at, h.updated_at
		FROM habits h
		WHERE h.deleted_at IS NULL
		  AND (
		    (h.owner_user_id = $1 AND h.couple_id IS NULL)
		    OR h.couple_id = (
		      SELECT cm.couple_id
		      FROM couple_members cm
		      JOIN couples c ON c.id = cm.couple_id
		      WHERE cm.user_id = $1 AND cm.left_at IS NULL
		        AND c.status = 'active'
		      LIMIT 1
		    )
		  )
		ORDER BY h.created_at DESC, h.id
		LIMIT 200
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("list habits: %w", err)
	}
	defer rows.Close()

	habits := make([]Habit, 0)
	for rows.Next() {
		habit, err := scanHabit(rows)
		if err != nil {
			return nil, fmt.Errorf("scan habit: %w", err)
		}
		habits = append(habits, habit)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate habits: %w", err)
	}
	return habits, nil
}

func (r *Repository) Create(
	ctx context.Context,
	userID uuid.UUID,
	request CreateHabitRequest,
) (Habit, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return Habit{}, fmt.Errorf("begin habit creation: %w", err)
	}
	defer tx.Rollback(ctx)

	var coupleID *uuid.UUID
	if request.Scope == "couple" {
		coupleID, err = activeCoupleID(ctx, tx, userID)
		if err != nil {
			return Habit{}, err
		}
	}
	var (
		habit   Habit
		habitID uuid.UUID
	)
	err = tx.QueryRow(ctx, `
		INSERT INTO habits (
			owner_user_id, couple_id, title, description, target_per_week
		)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, title, description, target_per_week,
		          version, created_at, updated_at
	`, userID, coupleID, request.Title, request.Description, *request.TargetPerWeek).Scan(
		&habitID, &habit.Title, &habit.Description, &habit.TargetPerWeek,
		&habit.Version, &habit.CreatedAt, &habit.UpdatedAt,
	)
	if err != nil {
		return Habit{}, fmt.Errorf("insert habit: %w", err)
	}
	habit.ID = habitID.String()
	habit.Scope = request.Scope
	payload, err := habitPayload(habit)
	if err != nil {
		return Habit{}, err
	}
	if err := recordHabitChange(ctx, tx, userID, coupleID, "habit", habitID, "upsert", payload); err != nil {
		return Habit{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Habit{}, fmt.Errorf("commit habit creation: %w", err)
	}
	return habit, nil
}

func (r *Repository) Update(
	ctx context.Context,
	userID uuid.UUID,
	habitID uuid.UUID,
	request UpdateHabitRequest,
) (Habit, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return Habit{}, fmt.Errorf("begin habit update: %w", err)
	}
	defer tx.Rollback(ctx)

	var (
		habit    Habit
		id       uuid.UUID
		coupleID *uuid.UUID
	)
	err = tx.QueryRow(ctx, `
		UPDATE habits h
		SET title = $4, description = $5, target_per_week = $6,
		    version = h.version + 1, updated_at = NOW()
		WHERE h.id = $1 AND h.version = $3 AND h.deleted_at IS NULL
		  AND (
		    (h.owner_user_id = $2 AND h.couple_id IS NULL)
		    OR h.couple_id = (
		      SELECT cm.couple_id
		      FROM couple_members cm
		      JOIN couples c ON c.id = cm.couple_id
		      WHERE cm.user_id = $2 AND cm.left_at IS NULL
		        AND c.status = 'active'
		      LIMIT 1
		    )
		  )
		RETURNING h.id, h.title, h.description, h.target_per_week,
		          h.version, h.created_at, h.updated_at, h.couple_id
	`, habitID, userID, request.Version, *request.Title, *request.Description, *request.TargetPerWeek).Scan(
		&id, &habit.Title, &habit.Description,
		&habit.TargetPerWeek, &habit.Version, &habit.CreatedAt,
		&habit.UpdatedAt, &coupleID,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Habit{}, r.updateError(ctx, tx, habitID, userID)
	}
	if err != nil {
		if isConstraintViolation(err, "habits_title_check") ||
			isConstraintViolation(err, "habits_description_length_check") ||
			isConstraintViolation(err, "habits_target_per_week_check") {
			return Habit{}, ErrInvalidRequest
		}
		return Habit{}, fmt.Errorf("update habit: %w", err)
	}
	if coupleID == nil {
		habit.Scope = "personal"
	} else {
		habit.Scope = "couple"
	}
	habit.ID = id.String()
	payload, err := habitPayload(habit)
	if err != nil {
		return Habit{}, err
	}
	if err := recordHabitChange(ctx, tx, userID, coupleID, "habit", id, "upsert", payload); err != nil {
		return Habit{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Habit{}, fmt.Errorf("commit habit update: %w", err)
	}
	return habit, nil
}

func (r *Repository) Delete(
	ctx context.Context,
	userID uuid.UUID,
	habitID uuid.UUID,
	version int64,
) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin habit deletion: %w", err)
	}
	defer tx.Rollback(ctx)

	var coupleID *uuid.UUID
	err = tx.QueryRow(ctx, `
		UPDATE habits h
		SET deleted_at = NOW(), version = h.version + 1, updated_at = NOW()
		WHERE h.id = $1 AND h.version = $3 AND h.deleted_at IS NULL
		  AND (
		    (h.owner_user_id = $2 AND h.couple_id IS NULL)
		    OR h.couple_id = (
		      SELECT cm.couple_id
		      FROM couple_members cm
		      JOIN couples c ON c.id = cm.couple_id
		      WHERE cm.user_id = $2 AND cm.left_at IS NULL
		        AND c.status = 'active'
		      LIMIT 1
		    )
		  )
		RETURNING h.couple_id
	`, habitID, userID, version).Scan(&coupleID)
	if errors.Is(err, pgx.ErrNoRows) {
		return r.updateError(ctx, tx, habitID, userID)
	}
	if err != nil {
		return fmt.Errorf("delete habit: %w", err)
	}
	if err := recordHabitChange(ctx, tx, userID, coupleID, "habit", habitID, "delete", json.RawMessage(`{}`)); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit habit deletion: %w", err)
	}
	return nil
}

func (r *Repository) ListCompletions(
	ctx context.Context,
	userID uuid.UUID,
	habitID uuid.UUID,
	rangeDates dateRange,
) ([]Completion, error) {
	if _, err := r.getScopedHabit(ctx, r.db, userID, habitID); err != nil {
		return nil, err
	}
	rows, err := r.db.Query(ctx, `
		SELECT hc.id, hc.habit_id, hc.user_id, hc.completed_on,
		       hc.user_id = $2, hc.created_at
		FROM habit_completions hc
		WHERE hc.habit_id = $1 AND hc.deleted_at IS NULL
		  AND hc.completed_on BETWEEN $3::DATE AND $4::DATE
		  AND EXISTS (
		    SELECT 1 FROM habits h
		    WHERE h.id = hc.habit_id AND h.deleted_at IS NULL
		      AND (
		        (h.owner_user_id = $2 AND h.couple_id IS NULL)
		        OR h.couple_id = (
		          SELECT cm.couple_id
		          FROM couple_members cm
		          JOIN couples c ON c.id = cm.couple_id
		          WHERE cm.user_id = $2 AND cm.left_at IS NULL
		            AND c.status = 'active'
		          LIMIT 1
		        )
		      )
		  )
		ORDER BY hc.completed_on DESC, hc.user_id, hc.id
		LIMIT 732
	`, habitID, userID, rangeDates.From, rangeDates.To)
	if err != nil {
		return nil, fmt.Errorf("list habit completions: %w", err)
	}
	defer rows.Close()

	completions := make([]Completion, 0)
	for rows.Next() {
		var completion Completion
		var id, completionHabitID, completedBy uuid.UUID
		var completedOn time.Time
		if err := rows.Scan(
			&id, &completionHabitID, &completedBy, &completedOn,
			&completion.IsMine, &completion.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan habit completion: %w", err)
		}
		completion.ID = id.String()
		completion.HabitID = completionHabitID.String()
		completion.UserID = completedBy.String()
		completion.CompletedOn = completedOn.Format("2006-01-02")
		completions = append(completions, completion)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate habit completions: %w", err)
	}
	return completions, nil
}

func (r *Repository) Complete(
	ctx context.Context,
	userID uuid.UUID,
	habitID uuid.UUID,
	completedOn time.Time,
) (Completion, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return Completion{}, fmt.Errorf("begin habit completion: %w", err)
	}
	defer tx.Rollback(ctx)

	habit, err := r.getScopedHabit(ctx, tx, userID, habitID)
	if err != nil {
		return Completion{}, err
	}
	var (
		completion Completion
		id         uuid.UUID
	)
	err = tx.QueryRow(ctx, `
		INSERT INTO habit_completions (habit_id, user_id, completed_on)
		VALUES ($1, $2, $3::DATE)
		ON CONFLICT (habit_id, user_id, completed_on)
			WHERE deleted_at IS NULL
		DO NOTHING
		RETURNING id, created_at
	`, habitID, userID, completedOn).Scan(&id, &completion.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		err = tx.QueryRow(ctx, `
			SELECT id, created_at
			FROM habit_completions
			WHERE habit_id = $1 AND user_id = $2
			  AND completed_on = $3::DATE AND deleted_at IS NULL
		`, habitID, userID, completedOn).Scan(&id, &completion.CreatedAt)
		if err != nil {
			return Completion{}, fmt.Errorf("read existing habit completion: %w", err)
		}
	} else if err != nil {
		return Completion{}, fmt.Errorf("insert habit completion: %w", err)
	} else {
		payload, err := completionPayload(habitID, userID, completedOn)
		if err != nil {
			return Completion{}, err
		}
		if err := recordHabitChange(
			ctx, tx, userID, habit.coupleID, "habit_completion", id, "upsert", payload,
		); err != nil {
			return Completion{}, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return Completion{}, fmt.Errorf("commit habit completion: %w", err)
	}
	completion.ID = id.String()
	completion.HabitID = habitID.String()
	completion.UserID = userID.String()
	completion.CompletedOn = completedOn.Format("2006-01-02")
	completion.IsMine = true
	return completion, nil
}

func (r *Repository) Uncomplete(
	ctx context.Context,
	userID uuid.UUID,
	habitID uuid.UUID,
	completedOn time.Time,
) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin habit uncompletion: %w", err)
	}
	defer tx.Rollback(ctx)

	habit, err := r.getScopedHabit(ctx, tx, userID, habitID)
	if err != nil {
		return err
	}
	var completionID uuid.UUID
	err = tx.QueryRow(ctx, `
		UPDATE habit_completions
		SET deleted_at = NOW()
		WHERE habit_id = $1 AND user_id = $2
		  AND completed_on = $3::DATE AND deleted_at IS NULL
		RETURNING id
	`, habitID, userID, completedOn).Scan(&completionID)
	if errors.Is(err, pgx.ErrNoRows) {
		return tx.Commit(ctx)
	}
	if err != nil {
		return fmt.Errorf("delete habit completion: %w", err)
	}
	if err := recordHabitChange(ctx, tx, userID, habit.coupleID, "habit_completion", completionID, "delete", json.RawMessage(`{}`)); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit habit uncompletion: %w", err)
	}
	return nil
}

func (r *Repository) Stats(
	ctx context.Context,
	userID uuid.UUID,
	habitID uuid.UUID,
	rangeDates dateRange,
) (Habit, []string, []completionRecord, error) {
	habit, err := r.getScopedHabit(ctx, r.db, userID, habitID)
	if err != nil {
		return Habit{}, nil, nil, err
	}
	memberIDs := make([]string, 0, 2)
	if habit.coupleID == nil {
		memberIDs = append(memberIDs, habit.ownerID.String())
	} else {
		rows, err := r.db.Query(ctx, `
			SELECT user_id
			FROM couple_members
			WHERE couple_id = $1 AND left_at IS NULL
			ORDER BY user_id
		`, *habit.coupleID)
		if err != nil {
			return Habit{}, nil, nil, fmt.Errorf("list habit members: %w", err)
		}
		for rows.Next() {
			var memberID uuid.UUID
			if err := rows.Scan(&memberID); err != nil {
				rows.Close()
				return Habit{}, nil, nil, fmt.Errorf("scan habit member: %w", err)
			}
			memberIDs = append(memberIDs, memberID.String())
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return Habit{}, nil, nil, fmt.Errorf("iterate habit members: %w", err)
		}
		rows.Close()
	}
	from := time.Now().UTC().Truncate(24*time.Hour).AddDate(0, 0, -365)
	if rangeDates.From.Before(from) {
		from = rangeDates.From
	}
	to := rangeDates.To
	today := time.Now().UTC().Truncate(24 * time.Hour)
	if today.After(to) {
		to = today
	}
	rows, err := r.db.Query(ctx, `
		SELECT hc.user_id, hc.completed_on
		FROM habit_completions hc
		WHERE hc.habit_id = $1 AND hc.deleted_at IS NULL
		  AND hc.completed_on BETWEEN $3::DATE AND $4::DATE
		  AND EXISTS (
		    SELECT 1 FROM habits h
		    WHERE h.id = hc.habit_id AND h.deleted_at IS NULL
		      AND (
		        (h.owner_user_id = $2 AND h.couple_id IS NULL)
		        OR h.couple_id = (
		          SELECT cm.couple_id
		          FROM couple_members cm
		          JOIN couples c ON c.id = cm.couple_id
		          WHERE cm.user_id = $2 AND cm.left_at IS NULL
		            AND c.status = 'active'
		          LIMIT 1
		        )
		      )
		  )
		ORDER BY hc.completed_on, hc.user_id
		LIMIT 1464
	`, habitID, userID, from, to)
	if err != nil {
		return Habit{}, nil, nil, fmt.Errorf("query habit statistics: %w", err)
	}
	defer rows.Close()

	records := make([]completionRecord, 0)
	for rows.Next() {
		var record completionRecord
		if err := rows.Scan(&record.UserID, &record.Date); err != nil {
			return Habit{}, nil, nil, fmt.Errorf("scan habit statistic: %w", err)
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return Habit{}, nil, nil, fmt.Errorf("iterate habit statistics: %w", err)
	}
	return habit.Habit, memberIDs, records, nil
}

type scopedHabit struct {
	Habit
	coupleID *uuid.UUID
	ownerID  uuid.UUID
}

func (r *Repository) getScopedHabit(
	ctx context.Context,
	queryer interface {
		QueryRow(context.Context, string, ...any) pgx.Row
	},
	userID uuid.UUID,
	habitID uuid.UUID,
) (scopedHabit, error) {
	var (
		habit    scopedHabit
		id       uuid.UUID
		coupleID *uuid.UUID
		ownerID  uuid.UUID
	)
	err := queryer.QueryRow(ctx, `
		SELECT h.id, h.owner_user_id,
		       CASE WHEN h.couple_id IS NULL THEN 'personal' ELSE 'couple' END,
		       h.title, h.description, h.target_per_week,
		       h.version, h.created_at, h.updated_at, h.couple_id
		FROM habits h
		WHERE h.id = $1 AND h.deleted_at IS NULL
		  AND (
		    (h.owner_user_id = $2 AND h.couple_id IS NULL)
		    OR h.couple_id = (
		      SELECT cm.couple_id
		      FROM couple_members cm
		      JOIN couples c ON c.id = cm.couple_id
		      WHERE cm.user_id = $2 AND cm.left_at IS NULL
		        AND c.status = 'active'
		      LIMIT 1
		    )
		  )
	`, habitID, userID).Scan(
		&id, &ownerID, &habit.Scope, &habit.Title, &habit.Description,
		&habit.TargetPerWeek, &habit.Version, &habit.CreatedAt,
		&habit.UpdatedAt, &coupleID,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return scopedHabit{}, ErrNotFound
	}
	if err != nil {
		return scopedHabit{}, fmt.Errorf("get habit access: %w", err)
	}
	habit.ID = id.String()
	habit.coupleID = coupleID
	habit.ownerID = ownerID
	return habit, nil
}

func (r *Repository) updateError(
	ctx context.Context,
	tx pgx.Tx,
	habitID uuid.UUID,
	userID uuid.UUID,
) error {
	var exists bool
	err := tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM habits h
			WHERE h.id = $1 AND h.deleted_at IS NULL
			  AND (
			    (h.owner_user_id = $2 AND h.couple_id IS NULL)
			    OR h.couple_id = (
			      SELECT cm.couple_id
			      FROM couple_members cm
			      JOIN couples c ON c.id = cm.couple_id
			      WHERE cm.user_id = $2 AND cm.left_at IS NULL
			        AND c.status = 'active'
			      LIMIT 1
			    )
			  )
		)
	`, habitID, userID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("check habit access: %w", err)
	}
	if exists {
		return ErrVersionConflict
	}
	return ErrNotFound
}

func activeCoupleID(ctx context.Context, tx pgx.Tx, userID uuid.UUID) (*uuid.UUID, error) {
	var coupleID uuid.UUID
	err := tx.QueryRow(ctx, `
		SELECT cm.couple_id
		FROM couple_members cm
		JOIN couples c ON c.id = cm.couple_id
		WHERE cm.user_id = $1 AND cm.left_at IS NULL AND c.status = 'active'
		LIMIT 1
	`, userID).Scan(&coupleID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNoActiveCouple
	}
	if err != nil {
		return nil, fmt.Errorf("find active couple: %w", err)
	}
	return &coupleID, nil
}

func scanHabit(row pgx.Row) (Habit, error) {
	var habit Habit
	var id uuid.UUID
	err := row.Scan(
		&id, &habit.Scope, &habit.Title, &habit.Description,
		&habit.TargetPerWeek, &habit.Version, &habit.CreatedAt, &habit.UpdatedAt,
	)
	if err != nil {
		return Habit{}, err
	}
	habit.ID = id.String()
	return habit, nil
}

func habitPayload(habit Habit) (json.RawMessage, error) {
	payload, err := json.Marshal(struct {
		Title         string `json:"title"`
		Description   string `json:"description"`
		TargetPerWeek int    `json:"target_per_week"`
		Scope         string `json:"scope"`
	}{
		Title: habit.Title, Description: habit.Description,
		TargetPerWeek: habit.TargetPerWeek, Scope: habit.Scope,
	})
	if err != nil {
		return nil, fmt.Errorf("encode habit sync payload: %w", err)
	}
	return payload, nil
}

func completionPayload(habitID, userID uuid.UUID, completedOn time.Time) (json.RawMessage, error) {
	payload, err := json.Marshal(struct {
		HabitID     string `json:"habit_id"`
		UserID      string `json:"user_id"`
		CompletedOn string `json:"completed_on"`
	}{
		HabitID: habitID.String(), UserID: userID.String(),
		CompletedOn: completedOn.Format("2006-01-02"),
	})
	if err != nil {
		return nil, fmt.Errorf("encode habit completion sync payload: %w", err)
	}
	return payload, nil
}

func recordHabitChange(
	ctx context.Context,
	tx pgx.Tx,
	userID uuid.UUID,
	coupleID *uuid.UUID,
	entityType string,
	entityID uuid.UUID,
	operation string,
	payload json.RawMessage,
) error {
	change := syncmodule.NewChange{
		ActorID: userID, EntityType: entityType, EntityID: entityID,
		Operation: operation, Payload: payload,
	}
	if coupleID == nil {
		change.Scope.UserID = &userID
	} else {
		change.Scope.CoupleID = coupleID
	}
	if _, err := syncmodule.RecordChange(ctx, tx, change); err != nil {
		return fmt.Errorf("record %s sync change: %w", entityType, err)
	}
	return nil
}

func isConstraintViolation(err error, constraint string) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) &&
		pgErr.Code == "23514" &&
		pgErr.ConstraintName == constraint
}
