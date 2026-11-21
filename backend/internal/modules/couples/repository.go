package couples

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	syncmodule "github.com/W0LIF/LoveTea/backend/internal/modules/sync"
)

var (
	ErrCoupleNotFound        = errors.New("couple not found")
	ErrAlreadyInCouple       = errors.New("user already belongs to a couple")
	ErrInvitationNotFound    = errors.New("invitation not found")
	ErrInvitationExpired     = errors.New("invitation expired")
	ErrInvitationUsed        = errors.New("invitation already used")
	ErrCannotJoinOwnCouple   = errors.New("cannot join your own couple")
	ErrCoupleIsFull          = errors.New("couple is full")
	ErrCoupleNotPending      = errors.New("couple is not pending")
	ErrInvitationAttempts    = errors.New("too many invitation attempts")
	ErrNoActiveCouple        = errors.New("user does not belong to a couple")
	ErrCoupleNotActive       = errors.New("couple is not active")
	ErrInvitationRateLimited = errors.New("too many invitation code attempts")
	ErrRateLimitUnavailable  = errors.New("rate limit service unavailable")
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) CreateCouple(
	ctx context.Context,
	userID uuid.UUID,
) (uuid.UUID, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return uuid.Nil, err
	}
	defer tx.Rollback(ctx)

	// Блокируем строку пользователя.
	// Параллельные запросы одного пользователя будут выполняться последовательно.
	var lockedUserID uuid.UUID

	err = tx.QueryRow(ctx, `
        SELECT id
        FROM users
        WHERE id = $1
        FOR UPDATE
    `, userID).Scan(&lockedUserID)

	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, ErrCoupleNotFound
	}
	if err != nil {
		return uuid.Nil, err
	}

	// Проверяем, есть ли у пользователя действующее членство.
	var existing uuid.UUID

	err = tx.QueryRow(ctx, `
        SELECT couple_id
        FROM couple_members
        WHERE user_id = $1
          AND left_at IS NULL
        LIMIT 1
    `, userID).Scan(&existing)

	if err == nil {
		return uuid.Nil, ErrAlreadyInCouple
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, err
	}

	var coupleID uuid.UUID

	err = tx.QueryRow(ctx, `
        INSERT INTO couples (created_by)
        VALUES ($1)
        RETURNING id
    `, userID).Scan(&coupleID)

	if err != nil {
		return uuid.Nil, err
	}

	_, err = tx.Exec(ctx, `
        INSERT INTO couple_members (
            couple_id,
            user_id,
            mascot_type
        )
        VALUES ($1, $2, 'kitten')
    `, coupleID, userID)

	if err != nil {
		if isUniqueViolation(err) {
			return uuid.Nil, ErrAlreadyInCouple
		}
		return uuid.Nil, err
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO couple_pet_state (couple_id)
		VALUES ($1)
	`, coupleID)
	if err != nil {
		return uuid.Nil, err
	}

	if _, err := syncmodule.RecordChange(ctx, tx, syncmodule.NewChange{
		Scope:      syncmodule.Scope{CoupleID: &coupleID},
		ActorID:    userID,
		EntityType: "couple",
		EntityID:   coupleID,
		Operation:  "upsert",
		Payload:    json.RawMessage(`{"status":"pending"}`),
	}); err != nil {
		return uuid.Nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		if isUniqueViolation(err) {
			return uuid.Nil, ErrAlreadyInCouple
		}
		return uuid.Nil, err
	}

	return coupleID, nil
}

func (r *Repository) CreateInvitation(
	ctx context.Context,
	coupleID uuid.UUID,
	userID uuid.UUID,
	codeHash []byte,
	tokenHash []byte,
	expiresAt time.Time,
) (uuid.UUID, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return uuid.Nil, err
	}
	defer tx.Rollback(ctx)

	var status string
	err = tx.QueryRow(ctx, `
        SELECT status
        FROM couples
        WHERE id = $1
        FOR UPDATE
    `, coupleID).Scan(&status)

	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, ErrCoupleNotFound
	}
	if err != nil {
		return uuid.Nil, err
	}
	if status != "pending" {
		return uuid.Nil, ErrCoupleNotPending
	}

	var isMember bool
	err = tx.QueryRow(ctx, `
        SELECT EXISTS (
            SELECT 1
            FROM couple_members
            WHERE couple_id = $1
              AND user_id = $2
              AND left_at IS NULL
        )
    `, coupleID, userID).Scan(&isMember)
	if err != nil {
		return uuid.Nil, err
	}
	if !isMember {
		return uuid.Nil, ErrCoupleNotFound
	}

	// Новое приглашение отзывает предыдущее, в том числе истёкшее.
	_, err = tx.Exec(ctx, `
        UPDATE couple_invitations
        SET revoked_at = NOW()
        WHERE couple_id = $1
          AND accepted_at IS NULL
          AND revoked_at IS NULL
    `, coupleID)
	if err != nil {
		return uuid.Nil, err
	}

	var invitationID uuid.UUID
	err = tx.QueryRow(ctx, `
        INSERT INTO couple_invitations (
            couple_id,
            created_by,
            code_hash,
            token_hash,
            expires_at
        )
        VALUES ($1, $2, $3, $4, $5)
        RETURNING id
    `, coupleID, userID, codeHash, tokenHash, expiresAt).
		Scan(&invitationID)
	if err != nil {
		return uuid.Nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return uuid.Nil, err
	}

	return invitationID, nil
}

func (r *Repository) AcceptInvitation(
	ctx context.Context,
	userID uuid.UUID,
	hash []byte,
	useToken bool,
) (uuid.UUID, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return uuid.Nil, err
	}
	defer tx.Rollback(ctx)

	// Сначала узнаём couple_id, затем блокируем строку пары.
	// Это помогает сериализовать одновременное принятие приглашений.
	var coupleID uuid.UUID

	query := `
        SELECT couple_id
        FROM couple_invitations
        WHERE code_hash = $1
    `
	if useToken {
		query = `
            SELECT couple_id
            FROM couple_invitations
            WHERE token_hash = $1
        `
	}

	err = tx.QueryRow(ctx, query, hash).Scan(&coupleID)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, ErrInvitationNotFound
	}
	if err != nil {
		return uuid.Nil, err
	}

	var status string
	err = tx.QueryRow(ctx, `
        SELECT status
        FROM couples
        WHERE id = $1
        FOR UPDATE
    `, coupleID).Scan(&status)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, ErrCoupleNotFound
	}
	if err != nil {
		return uuid.Nil, err
	}
	if status != "pending" {
		return uuid.Nil, ErrCoupleNotPending
	}

	var (
		invitationID uuid.UUID
		creatorID    uuid.UUID
		expiresAt    time.Time
		acceptedAt   *time.Time
		revokedAt    *time.Time
	)

	query = `
        SELECT id, created_by, expires_at, accepted_at,
               revoked_at
        FROM couple_invitations
        WHERE code_hash = $1
        FOR UPDATE
    `
	if useToken {
		query = `
            SELECT id, created_by, expires_at, accepted_at,
                   revoked_at
            FROM couple_invitations
            WHERE token_hash = $1
            FOR UPDATE
        `
	}

	err = tx.QueryRow(ctx, query, hash).Scan(
		&invitationID,
		&creatorID,
		&expiresAt,
		&acceptedAt,
		&revokedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, ErrInvitationNotFound
	}
	if err != nil {
		return uuid.Nil, err
	}

	if acceptedAt != nil {
		return uuid.Nil, ErrInvitationUsed
	}
	if revokedAt != nil {
		return uuid.Nil, ErrInvitationNotFound
	}
	if time.Now().After(expiresAt) {
		return uuid.Nil, ErrInvitationExpired
	}

	if creatorID == userID {
		return uuid.Nil, ErrCannotJoinOwnCouple
	}

	// Проверяем, что пользователь ещё не состоит в другой паре.
	var existing bool
	err = tx.QueryRow(ctx, `
        SELECT EXISTS (
            SELECT 1
            FROM couple_members
            WHERE user_id = $1 AND left_at IS NULL
        )
    `, userID).Scan(&existing)
	if err != nil {
		return uuid.Nil, err
	}
	if existing {
		return uuid.Nil, ErrAlreadyInCouple
	}

	var memberCount int
	err = tx.QueryRow(ctx, `
        SELECT COUNT(*)
        FROM couple_members
        WHERE couple_id = $1 AND left_at IS NULL
    `, coupleID).Scan(&memberCount)
	if err != nil {
		return uuid.Nil, err
	}
	if memberCount >= 2 {
		return uuid.Nil, ErrCoupleIsFull
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO couple_members (couple_id, user_id, mascot_type)
		VALUES ($1, $2, 'wolf')
	`, coupleID, userID)

	if err != nil {
		if isUniqueViolation(err) {
			return uuid.Nil, ErrAlreadyInCouple
		}

		return uuid.Nil, err
	}

	_, err = tx.Exec(ctx, `
        UPDATE couple_invitations
        SET accepted_at = NOW(),
            accepted_by = $2
        WHERE id = $1
    `, invitationID, userID)
	if err != nil {
		return uuid.Nil, err
	}

	_, err = tx.Exec(ctx, `
        UPDATE couples
        SET status = 'active',
            activated_at = NOW()
        WHERE id = $1
    `, coupleID)
	if err != nil {
		return uuid.Nil, err
	}

	if _, err := syncmodule.RecordChange(ctx, tx, syncmodule.NewChange{
		Scope:      syncmodule.Scope{CoupleID: &coupleID},
		ActorID:    userID,
		EntityType: "couple",
		EntityID:   coupleID,
		Operation:  "upsert",
		Payload:    json.RawMessage(`{"status":"active"}`),
	}); err != nil {
		return uuid.Nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return uuid.Nil, err
	}

	return coupleID, nil
}

type CoupleMember struct {
	UserID     uuid.UUID
	Email      string
	Nickname   *string
	MascotType string
	JoinedAt   time.Time
}

type CoupleDetails struct {
	ID          uuid.UUID
	Status      string
	CreatedAt   time.Time
	ActivatedAt *time.Time
	Members     []CoupleMember
}

func (r *Repository) GetMyCouple(
	ctx context.Context,
	userID uuid.UUID,
) (CoupleDetails, error) {
	var couple CoupleDetails

	err := r.db.QueryRow(ctx, `
        SELECT c.id, c.status, c.created_at, c.activated_at
        FROM couples c
        JOIN couple_members cm ON cm.couple_id = c.id
        WHERE cm.user_id = $1
          AND cm.left_at IS NULL
        LIMIT 1
    `, userID).Scan(
		&couple.ID,
		&couple.Status,
		&couple.CreatedAt,
		&couple.ActivatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return CoupleDetails{}, ErrNoActiveCouple
	}
	if err != nil {
		return CoupleDetails{}, err
	}

	rows, err := r.db.Query(ctx, `
        SELECT
            u.id,
            u.email,
            cm.nickname,
            cm.mascot_type,
            cm.joined_at
        FROM couple_members cm
        JOIN users u ON u.id = cm.user_id
        WHERE cm.couple_id = $1
          AND cm.left_at IS NULL
        ORDER BY cm.joined_at ASC
    `, couple.ID)
	if err != nil {
		return CoupleDetails{}, err
	}
	defer rows.Close()

	couple.Members = make([]CoupleMember, 0, 2)

	for rows.Next() {
		var member CoupleMember

		if err := rows.Scan(
			&member.UserID,
			&member.Email,
			&member.Nickname,
			&member.MascotType,
			&member.JoinedAt,
		); err != nil {
			return CoupleDetails{}, err
		}

		couple.Members = append(couple.Members, member)
	}

	if err := rows.Err(); err != nil {
		return CoupleDetails{}, err
	}

	return couple, nil
}

func (r *Repository) UpdateMemberProfile(
	ctx context.Context,
	coupleID uuid.UUID,
	userID uuid.UUID,
	nickname *string,
	mascotType *string,
) error {
	payload := make(map[string]any, 2)
	if nickname != nil {
		payload["nickname"] = *nickname
	}
	if mascotType != nil {
		payload["mascot_type"] = *mascotType
	}
	encodedPayload, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	commandTag, err := tx.Exec(ctx, `
        UPDATE couple_members
        SET
            nickname = COALESCE($3, nickname),
            mascot_type = COALESCE($4, mascot_type)
        WHERE couple_id = $1
          AND user_id = $2
          AND left_at IS NULL
    `, coupleID, userID, nickname, mascotType)

	if err != nil {
		return err
	}

	if commandTag.RowsAffected() == 0 {
		return ErrNoActiveCouple
	}

	if _, err := syncmodule.RecordChange(ctx, tx, syncmodule.NewChange{
		Scope:      syncmodule.Scope{CoupleID: &coupleID},
		ActorID:    userID,
		EntityType: "couple_member",
		EntityID:   userID,
		Operation:  "upsert",
		Payload:    encodedPayload,
	}); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (r *Repository) ConfirmDetach(
	ctx context.Context,
	userID uuid.UUID,
) (uuid.UUID, bool, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return uuid.Nil, false, err
	}
	defer tx.Rollback(ctx)

	var coupleID uuid.UUID

	err = tx.QueryRow(ctx, `
        SELECT couple_id
        FROM couple_members
        WHERE user_id = $1
          AND left_at IS NULL
    `, userID).Scan(&coupleID)

	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, false, ErrNoActiveCouple
	}
	if err != nil {
		return uuid.Nil, false, err
	}

	var status string

	err = tx.QueryRow(ctx, `
        SELECT status
        FROM couples
        WHERE id = $1
        FOR UPDATE
    `, coupleID).Scan(&status)

	if err != nil {
		return uuid.Nil, false, err
	}

	if status != "active" {
		return uuid.Nil, false, ErrCoupleNotActive
	}

	_, err = tx.Exec(ctx, `
        INSERT INTO couple_detach_confirmations (couple_id, user_id)
        VALUES ($1, $2)
        ON CONFLICT (couple_id, user_id) DO NOTHING
    `, coupleID, userID)
	if err != nil {
		return uuid.Nil, false, err
	}

	if _, err := syncmodule.RecordChange(ctx, tx, syncmodule.NewChange{
		Scope:      syncmodule.Scope{CoupleID: &coupleID},
		ActorID:    userID,
		EntityType: "couple_detach_confirmation",
		EntityID:   userID,
		Operation:  "upsert",
		Payload:    json.RawMessage(`{"confirmed":true}`),
	}); err != nil {
		return uuid.Nil, false, err
	}

	var confirmationCount int

	err = tx.QueryRow(ctx, `
        SELECT COUNT(*)
        FROM couple_detach_confirmations dc
        JOIN couple_members cm
          ON cm.couple_id = dc.couple_id
         AND cm.user_id = dc.user_id
        WHERE dc.couple_id = $1
          AND cm.left_at IS NULL
    `, coupleID).Scan(&confirmationCount)

	if err != nil {
		return uuid.Nil, false, err
	}

	if confirmationCount >= 2 {
		// Удаление всех совместных данных перед статусом 'detached'.
		// Порядок важен из-за внешних ключей.
		// Личные данные (owner_user_id) не затрагиваются.
		deleteQueries := []string{
			"DELETE FROM habit_completions hc USING habits h WHERE hc.habit_id = h.id AND h.couple_id = $1",
			"DELETE FROM habits WHERE couple_id = $1",
			"DELETE FROM tasks WHERE couple_id = $1",
			"DELETE FROM calendar_events WHERE couple_id = $1",
			"DELETE FROM important_dates WHERE couple_id = $1",
			"DELETE FROM mood_entries WHERE shared_couple_id = $1",
			"DELETE FROM sleep_entries WHERE shared_couple_id = $1",
			"DELETE FROM moments WHERE couple_id = $1",
			"DELETE FROM media_files WHERE couple_id = $1",
			"DELETE FROM couple_pet_state WHERE couple_id = $1",
			"DELETE FROM sync_entity_versions WHERE couple_id = $1",
			"DELETE FROM sync_changes WHERE couple_id = $1",
		}

		for _, q := range deleteQueries {
			if _, err := tx.Exec(ctx, q, coupleID); err != nil {
				return uuid.Nil, false, fmt.Errorf("detach cleanup: %w", err)
			}
		}

		_, err = tx.Exec(ctx, `
            UPDATE couples
            SET status = 'detached',
                detached_at = NOW()
            WHERE id = $1
        `, coupleID)
		if err != nil {
			return uuid.Nil, false, err
		}

		if _, err := syncmodule.RecordChange(ctx, tx, syncmodule.NewChange{
			Scope:      syncmodule.Scope{CoupleID: &coupleID},
			ActorID:    userID,
			EntityType: "couple_access_revoked",
			EntityID:   coupleID,
			Operation:  "delete",
			Payload:    json.RawMessage(`{}`),
		}); err != nil {
			return uuid.Nil, false, err
		}

		_, err = tx.Exec(ctx, `
            UPDATE couple_members
            SET left_at = NOW()
            WHERE couple_id = $1
              AND left_at IS NULL
        `, coupleID)
		if err != nil {
			return uuid.Nil, false, err
		}

		_, err = tx.Exec(ctx, `
            UPDATE couple_invitations
            SET revoked_at = NOW()
            WHERE couple_id = $1
              AND accepted_at IS NULL
              AND revoked_at IS NULL
        `, coupleID)
		if err != nil {
			return uuid.Nil, false, err
		}

		if err := tx.Commit(ctx); err != nil {
			return uuid.Nil, false, err
		}

		return coupleID, true, nil
	}

	if err := tx.Commit(ctx); err != nil {
		return uuid.Nil, false, err
	}

	return coupleID, false, nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError

	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
