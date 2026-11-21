package memories

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrNotFound     = errors.New("not found")
	ErrInvalidInput = errors.New("invalid input")
	ErrNoCouple     = errors.New("no active couple")
)

type Memory struct {
	ID              uuid.UUID  `json:"id"`
	OwnerUserID     uuid.UUID  `json:"owner_user_id"`
	CoupleID        *uuid.UUID `json:"couple_id"`
	Title           string     `json:"title"`
	Content         string     `json:"content"`
	ContentEncrypted bool      `json:"content_encrypted"`
	Ciphertext      *string    `json:"ciphertext"`
	CipherNonce     *string    `json:"cipher_nonce"`
	CipherKeyRef    *string    `json:"cipher_key_ref"`
	Version         int64      `json:"version"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

type CreateMemoryRequest struct {
	Title  string `json:"title"`
	Content string `json:"content"`
}

type UpdateMemoryRequest struct {
	Title  string `json:"title"`
	Content string `json:"content"`
}

type CreateEncryptedMemoryRequest struct {
	Title          string `json:"title"`
	Ciphertext     string `json:"ciphertext"`
	CipherNonce    string `json:"cipher_nonce"`
	CipherKeyRef   string `json:"cipher_key_ref"`
}

type ListFilter struct {
	CoupleID *uuid.UUID
}
