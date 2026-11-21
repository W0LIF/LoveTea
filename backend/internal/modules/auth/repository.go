package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrUserNotFound = errors.New("user not found")
var ErrEmailAlreadyExists = errors.New("email already exists")
var ErrInvalidRefreshToken = errors.New("invalid refresh token")

type User struct {
	ID           uuid.UUID
	Email        string
	PasswordHash string
	CreatedAt    time.Time
}

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) CreateUser(
	ctx context.Context,
	email, passwordHash string,
) (*User, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin create user transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	user := &User{}

	err = tx.QueryRow(ctx, `
		INSERT INTO users (email, password_hash)
		VALUES ($1, $2)
		RETURNING id, email, password_hash, created_at
	`, email, passwordHash).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.CreatedAt,
	)

	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrEmailAlreadyExists
		}
		return nil, fmt.Errorf("create user: %w", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO user_profiles (user_id)
		VALUES ($1)
	`, user.ID)

	if err != nil {
		return nil, fmt.Errorf("create user profile: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit create user transaction: %w", err)
	}

	return user, nil
}

func (r *Repository) GetUserByEmail(
	ctx context.Context,
	email string,
) (*User, error) {
	query := `
		SELECT id, email, password_hash, created_at
		FROM users
		WHERE LOWER(email) = LOWER($1)
	`

	user := &User{}

	err := r.db.QueryRow(ctx, query, email).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.CreatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get user by email: %w", err)
	}

	return user, nil
}

func (r *Repository) GetUserByID(
	ctx context.Context,
	id uuid.UUID,
) (*User, error) {
	query := `
		SELECT id, email, password_hash, created_at
		FROM users
		WHERE id = $1
	`

	user := &User{}

	err := r.db.QueryRow(ctx, query, id).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.CreatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get user by ID: %w", err)
	}

	return user, nil
}

func (r *Repository) SaveRefreshToken(
	ctx context.Context,
	userID uuid.UUID,
	tokenHash []byte,
	expiresAt time.Time,
) error {
	query := `
		INSERT INTO refresh_tokens (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)
	`

	if _, err := r.db.Exec(ctx, query, userID, tokenHash, expiresAt); err != nil {
		return fmt.Errorf("save refresh token: %w", err)
	}

	return nil
}

func (r *Repository) RotateRefreshToken(
	ctx context.Context,
	oldHash, newHash []byte,
	newExpiresAt time.Time,
) (uuid.UUID, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return uuid.Nil, fmt.Errorf("begin refresh transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var userID uuid.UUID
	var oldID uuid.UUID

	err = tx.QueryRow(ctx, `
		SELECT id, user_id
		FROM refresh_tokens
		WHERE token_hash = $1
		  AND revoked_at IS NULL
		  AND expires_at > NOW()
		FOR UPDATE
	`, oldHash).Scan(&oldID, &userID)

	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, ErrInvalidRefreshToken
	}
	if err != nil {
		return uuid.Nil, fmt.Errorf("find refresh token: %w", err)
	}

	var newID uuid.UUID

	err = tx.QueryRow(ctx, `
		INSERT INTO refresh_tokens (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)
		RETURNING id
	`, userID, newHash, newExpiresAt).Scan(&newID)

	if err != nil {
		return uuid.Nil, fmt.Errorf("create replacement token: %w", err)
	}

	_, err = tx.Exec(ctx, `
		UPDATE refresh_tokens
		SET revoked_at = NOW(), replaced_by = $2
		WHERE id = $1
	`, oldID, newID)

	if err != nil {
		return uuid.Nil, fmt.Errorf("revoke old token: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return uuid.Nil, fmt.Errorf("commit refresh transaction: %w", err)
	}

	return userID, nil
}

func (r *Repository) RevokeRefreshToken(
	ctx context.Context,
	userID uuid.UUID,
	tokenHash []byte,
) error {
	_, err := r.db.Exec(ctx, `
		UPDATE refresh_tokens
		SET revoked_at = NOW()
		WHERE user_id = $1
		  AND token_hash = $2
		  AND revoked_at IS NULL
	`, userID, tokenHash)

	if err != nil {
		return fmt.Errorf("revoke refresh token: %w", err)
	}

	return nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

// ── Password reset token methods ──────────────────────────────────

func (r *Repository) CreatePasswordResetToken(
	ctx context.Context,
	userID uuid.UUID,
	tokenHash string,
	expiresAt time.Time,
) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO password_reset_tokens (user_id, token_hash, expires_at)
		 VALUES ($1, $2, $3)`,
		userID, tokenHash, expiresAt,
	)
	if err != nil {
		return fmt.Errorf("auth: create password reset token: %w", err)
	}
	return nil
}

func (r *Repository) GetPasswordResetToken(
	ctx context.Context,
	tokenHash string,
) (uuid.UUID, time.Time, error) {
	var userID uuid.UUID
	var expiresAt time.Time
	var usedAt *time.Time

	err := r.db.QueryRow(ctx,
		`SELECT user_id, expires_at, used_at
		 FROM password_reset_tokens
		 WHERE token_hash = $1 AND used_at IS NULL`,
		tokenHash,
	).Scan(&userID, &expiresAt, &usedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, time.Time{}, ErrInvalidRefreshToken
	}
	if err != nil {
		return uuid.Nil, time.Time{}, fmt.Errorf("auth: get password reset token: %w", err)
	}

	return userID, expiresAt, nil
}

func (r *Repository) MarkPasswordResetTokenUsed(
	ctx context.Context,
	tokenHash string,
) error {
	_, err := r.db.Exec(ctx,
		`UPDATE password_reset_tokens SET used_at = NOW() WHERE token_hash = $1`,
		tokenHash,
	)
	if err != nil {
		return fmt.Errorf("auth: mark password reset token used: %w", err)
	}
	return nil
}

func (r *Repository) UpdateUserPassword(
	ctx context.Context,
	userID uuid.UUID,
	passwordHash string,
) error {
	_, err := r.db.Exec(ctx,
		`UPDATE users SET password_hash = $1 WHERE id = $2`,
		passwordHash, userID,
	)
	if err != nil {
		return fmt.Errorf("auth: update user password: %w", err)
	}
	return nil
}
