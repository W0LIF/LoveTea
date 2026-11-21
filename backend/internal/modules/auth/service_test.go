package auth

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/W0LIF/LoveTea/backend/internal/platform/security"
	"github.com/google/uuid"
)

type fakeRepository struct {
	user          *User
	createdEmail  string
	createdHash   string
	refreshHash   []byte
	refreshExpiry time.Time
	err           error
}

func (r *fakeRepository) CreateUser(
	_ context.Context,
	email string,
	passwordHash string,
) (*User, error) {
	r.createdEmail = email
	r.createdHash = passwordHash
	if r.err != nil {
		return nil, r.err
	}
	r.user = &User{
		ID:           uuid.New(),
		Email:        email,
		PasswordHash: passwordHash,
		CreatedAt:    time.Now(),
	}
	return r.user, nil
}

func (r *fakeRepository) GetUserByEmail(_ context.Context, email string) (*User, error) {
	if r.err != nil {
		return nil, r.err
	}
	if r.user == nil || r.user.Email != email {
		return nil, ErrUserNotFound
	}
	return r.user, nil
}

func (r *fakeRepository) GetUserByID(_ context.Context, id uuid.UUID) (*User, error) {
	if r.user == nil || r.user.ID != id {
		return nil, ErrUserNotFound
	}
	return r.user, nil
}

func (r *fakeRepository) SaveRefreshToken(
	_ context.Context,
	_ uuid.UUID,
	hash []byte,
	expiry time.Time,
) error {
	r.refreshHash = append([]byte(nil), hash...)
	r.refreshExpiry = expiry
	return r.err
}

func (r *fakeRepository) RotateRefreshToken(
	context.Context,
	[]byte,
	[]byte,
	time.Time,
) (uuid.UUID, error) {
	return uuid.Nil, errors.New("not used")
}

func (r *fakeRepository) RevokeRefreshToken(context.Context, uuid.UUID, []byte) error {
	return nil
}

func (r *fakeRepository) CreatePasswordResetToken(_ context.Context, _ uuid.UUID, _ string, _ time.Time) error {
	return nil
}

func (r *fakeRepository) GetPasswordResetToken(_ context.Context, _ string) (uuid.UUID, time.Time, error) {
	return uuid.Nil, time.Time{}, nil
}

func (r *fakeRepository) MarkPasswordResetTokenUsed(_ context.Context, _ string) error {
	return nil
}

func (r *fakeRepository) UpdateUserPassword(_ context.Context, _ uuid.UUID, _ string) error {
	return nil
}

func TestRegisterCreatesAccountAndIssuesTokensWithoutEmailConfirmation(t *testing.T) {
	repo := &fakeRepository{}
	service := NewService(
		repo,
		security.NewTokenManager(strings.Repeat("s", 32)),
		nil,
	)

	response, err := service.Register(context.Background(), " USER@example.com ", "password-123")
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if repo.createdEmail != "user@example.com" || repo.createdHash == "" {
		t.Fatalf("Register() did not create normalized account: %+v", repo)
	}
	if response.User.Email != repo.createdEmail ||
		response.AccessToken == "" ||
		response.RefreshToken == "" ||
		response.TokenType != "Bearer" {
		t.Fatalf("Register() returned incomplete auth response: %+v", response)
	}
	if len(repo.refreshHash) == 0 || string(repo.refreshHash) == response.RefreshToken {
		t.Fatal("Register() did not store only a refresh-token hash")
	}
}

func TestLoginAcceptsRegisteredCredentials(t *testing.T) {
	passwordHash, err := security.HashPassword("password-123")
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}
	repo := &fakeRepository{user: &User{
		ID:           uuid.New(),
		Email:        "user@example.com",
		PasswordHash: passwordHash,
		CreatedAt:    time.Now(),
	}}
	response, err := NewService(repo, security.NewTokenManager(strings.Repeat("s", 32)), nil).
		Login(context.Background(), " USER@example.com ", "password-123")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if response.User.Email != "user@example.com" || response.AccessToken == "" {
		t.Fatalf("Login() returned incomplete response: %+v", response)
	}
}

func TestLoginRejectsWrongPassword(t *testing.T) {
	passwordHash, err := security.HashPassword("password-123")
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}
	repo := &fakeRepository{user: &User{
		ID:           uuid.New(),
		Email:        "user@example.com",
		PasswordHash: passwordHash,
	}}
	_, err = NewService(repo, security.NewTokenManager(strings.Repeat("s", 32)), nil).
		Login(context.Background(), "user@example.com", "wrong-password")
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("Login() error = %v, want invalid credentials", err)
	}
}

func TestValidatePassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{name: "minimum length", password: "пароль123"},
		{name: "too short", password: "1234567", wantErr: true},
		{name: "too many bytes", password: strings.Repeat("a", 129), wantErr: true},
		{name: "invalid UTF-8", password: string([]byte{0xff}), wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validatePassword(test.password)
			if (err != nil) != test.wantErr {
				t.Fatalf("validatePassword() error = %v, wantErr %t", err, test.wantErr)
			}
			if err == nil && !utf8.ValidString(test.password) {
				t.Fatal("invalid UTF-8 password was accepted")
			}
		})
	}
}
