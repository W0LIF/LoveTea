package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"
	"unicode/utf8"

	appmail "github.com/W0LIF/LoveTea/backend/internal/platform/mail"
	"github.com/W0LIF/LoveTea/backend/internal/platform/security"
	"github.com/google/uuid"
)

const (
	accessTokenTTL  = 15 * time.Minute
	refreshTokenTTL = 30 * 24 * time.Hour
)

var (
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrInvalidInput       = errors.New("invalid input")
)

type Service struct {
	repo   repository
	tokens *security.TokenManager
	mail   *appmail.Sender
}

type repository interface {
	CreateUser(context.Context, string, string) (*User, error)
	GetUserByEmail(context.Context, string) (*User, error)
	GetUserByID(context.Context, uuid.UUID) (*User, error)
	SaveRefreshToken(context.Context, uuid.UUID, []byte, time.Time) error
	RotateRefreshToken(context.Context, []byte, []byte, time.Time) (uuid.UUID, error)
	RevokeRefreshToken(context.Context, uuid.UUID, []byte) error
	CreatePasswordResetToken(context.Context, uuid.UUID, string, time.Time) error
	GetPasswordResetToken(context.Context, string) (uuid.UUID, time.Time, error)
	MarkPasswordResetTokenUsed(context.Context, string) error
	UpdateUserPassword(context.Context, uuid.UUID, string) error
}

func NewService(
	repo repository,
	tokens *security.TokenManager,
	mailSender *appmail.Sender,
) *Service {
	return &Service{
		repo:   repo,
		tokens: tokens,
		mail:   mailSender,
	}
}

func (s *Service) Register(
	ctx context.Context,
	email, password string,
) (*AuthResponse, error) {
	email = strings.ToLower(strings.TrimSpace(email))

	if err := validateEmail(email); err != nil {
		return nil, err
	}
	if err := validatePassword(password); err != nil {
		return nil, err
	}

	passwordHash, err := security.HashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	user, err := s.repo.CreateUser(ctx, email, passwordHash)
	if err != nil {
		return nil, err
	}

	return s.issueTokens(ctx, user)
}

func (s *Service) Login(
	ctx context.Context,
	email, password string,
) (*AuthResponse, error) {
	email = strings.ToLower(strings.TrimSpace(email))

	user, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	if !security.VerifyPassword(password, user.PasswordHash) {
		return nil, ErrInvalidCredentials
	}

	return s.issueTokens(ctx, user)
}

func (s *Service) Refresh(
	ctx context.Context,
	refreshToken string,
) (*AuthResponse, error) {
	if strings.TrimSpace(refreshToken) == "" {
		return nil, ErrInvalidRefreshToken
	}

	newToken, newHash, err := s.tokens.GenerateRefreshToken()
	if err != nil {
		return nil, err
	}

	userID, err := s.repo.RotateRefreshToken(
		ctx,
		security.HashRefreshToken(refreshToken),
		newHash,
		time.Now().Add(refreshTokenTTL),
	)
	if err != nil {
		return nil, err
	}

	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	accessToken, err := s.tokens.GenerateAccessToken(user.ID)
	if err != nil {
		return nil, err
	}

	return &AuthResponse{
		User:         toUserResponse(user),
		AccessToken:  accessToken,
		TokenType:    "Bearer",
		ExpiresIn:    int(accessTokenTTL.Seconds()),
		RefreshToken: newToken,
	}, nil
}

func (s *Service) Logout(
	ctx context.Context,
	userID uuid.UUID,
	refreshToken string,
) error {
	if strings.TrimSpace(refreshToken) == "" {
		return ErrInvalidRefreshToken
	}

	return s.repo.RevokeRefreshToken(
		ctx,
		userID,
		security.HashRefreshToken(refreshToken),
	)
}

func (s *Service) GetUser(
	ctx context.Context,
	userID uuid.UUID,
) (*UserResponse, error) {
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	response := toUserResponse(user)
	return &response, nil
}

func (s *Service) issueTokens(
	ctx context.Context,
	user *User,
) (*AuthResponse, error) {
	accessToken, err := s.tokens.GenerateAccessToken(user.ID)
	if err != nil {
		return nil, err
	}

	refreshToken, refreshHash, err := s.tokens.GenerateRefreshToken()
	if err != nil {
		return nil, err
	}

	err = s.repo.SaveRefreshToken(
		ctx,
		user.ID,
		refreshHash,
		time.Now().Add(refreshTokenTTL),
	)
	if err != nil {
		return nil, err
	}

	return &AuthResponse{
		User:         toUserResponse(user),
		AccessToken:  accessToken,
		TokenType:    "Bearer",
		ExpiresIn:    int(accessTokenTTL.Seconds()),
		RefreshToken: refreshToken,
	}, nil
}

func toUserResponse(user *User) UserResponse {
	return UserResponse{
		ID:        user.ID.String(),
		Email:     user.Email,
		CreatedAt: user.CreatedAt,
	}
}

func validateEmail(email string) error {
	if len(email) > 254 {
		return ErrInvalidInput
	}

	address, err := mail.ParseAddress(email)
	if err != nil || address.Address != email {
		return ErrInvalidInput
	}

	return nil
}

func validatePassword(password string) error {
	if !utf8.ValidString(password) {
		return ErrInvalidInput
	}

	if utf8.RuneCountInString(password) < 8 ||
		len(password) > 128 {
		return ErrInvalidInput
	}

	return nil
}

// ── Password reset ────────────────────────────────────────────────

func (s *Service) ForgotPassword(ctx context.Context, email string) error {
	email = strings.ToLower(strings.TrimSpace(email))
	if err := validateEmail(email); err != nil {
		return ErrInvalidInput
	}

	user, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		// Don't reveal whether the email exists
		return nil
	}

	rawToken := make([]byte, 32)
	if _, err := rand.Read(rawToken); err != nil {
		return fmt.Errorf("auth: generate reset token: %w", err)
	}

	tokenStr := hex.EncodeToString(rawToken)
	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(tokenStr)))
	expiresAt := time.Now().UTC().Add(1 * time.Hour)

	if err := s.repo.CreatePasswordResetToken(ctx, user.ID, hash, expiresAt); err != nil {
		return fmt.Errorf("auth: create reset token: %w", err)
	}

	// Send email — non-fatal, log and continue
	if s.mail != nil {
		go func() {
			_ = s.mail.Send(
				email,
				"Reset Your LoveTea Password",
				fmt.Sprintf(
					"<p>Click to reset your password:</p><p><a href='https://lovettea.app/reset/%s'>Reset Password</a></p>",
					tokenStr,
				),
			)
		}()
	}

	return nil
}

func (s *Service) ResetPassword(ctx context.Context, tokenStr string, newPassword string) error {
	if err := validatePassword(newPassword); err != nil {
		return ErrInvalidInput
	}

	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(tokenStr)))

	userID, expiresAt, err := s.repo.GetPasswordResetToken(ctx, hash)
	if err != nil {
		return ErrInvalidCredentials
	}

	if time.Now().UTC().After(expiresAt) {
		return ErrInvalidCredentials
	}

	if err := s.repo.MarkPasswordResetTokenUsed(ctx, hash); err != nil {
		return fmt.Errorf("auth: mark reset token used: %w", err)
	}

	passwordHash, err := security.HashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("auth: hash password: %w", err)
	}

	if err := s.repo.UpdateUserPassword(ctx, userID, passwordHash); err != nil {
		return fmt.Errorf("auth: update password: %w", err)
	}

	return nil
}
