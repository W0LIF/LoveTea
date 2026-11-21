package memories

import (
	"context"

	"github.com/google/uuid"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Create(ctx context.Context, userID uuid.UUID, req CreateMemoryRequest) (Memory, error) {
	if req.Title == "" || req.Content == "" {
		return Memory{}, ErrInvalidInput
	}
	coupleID, err := s.repo.GetActiveCouple(ctx, userID)
	if err != nil {
		return Memory{}, err
	}
	m := Memory{
		ID:          uuid.New(),
		OwnerUserID: userID,
		CoupleID:    &coupleID,
		Title:       req.Title,
		Content:     req.Content,
		CreatedAt:   now(),
	}
	if err := s.repo.Create(ctx, &m); err != nil {
		return Memory{}, err
	}
	return m, nil
}

func (s *Service) CreateEncrypted(ctx context.Context, userID uuid.UUID, req CreateEncryptedMemoryRequest) (Memory, error) {
	if req.Title == "" || req.Ciphertext == "" || req.CipherNonce == "" {
		return Memory{}, ErrInvalidInput
	}
	coupleID, err := s.repo.GetActiveCouple(ctx, userID)
	if err != nil {
		return Memory{}, err
	}
	m := Memory{
		ID:               uuid.New(),
		OwnerUserID:      userID,
		CoupleID:         &coupleID,
		Title:            req.Title,
		Content:          "",
		ContentEncrypted: true,
		Ciphertext:       &req.Ciphertext,
		CipherNonce:      &req.CipherNonce,
		CipherKeyRef:     &req.CipherKeyRef,
		CreatedAt:        now(),
	}
	if err := s.repo.Create(ctx, &m); err != nil {
		return Memory{}, err
	}
	return m, nil
}

func (s *Service) List(ctx context.Context, userID uuid.UUID, f ListFilter) ([]Memory, error) {
	if f.CoupleID == nil {
		coupleID, err := s.repo.GetActiveCouple(ctx, userID)
		if err != nil {
			// No couple — return personal memories only
			return s.repo.List(ctx, userID, ListFilter{})
		}
		f.CoupleID = &coupleID
	}
	return s.repo.List(ctx, userID, f)
}

func (s *Service) Get(ctx context.Context, userID, id uuid.UUID) (Memory, error) {
	m, err := s.repo.Get(ctx, id)
	if err != nil {
		return Memory{}, err
	}
	if m.OwnerUserID != userID {
		if m.CoupleID == nil {
			return Memory{}, ErrNotFound
		}
		coupleID, err := s.repo.GetActiveCouple(ctx, userID)
		if err != nil || *m.CoupleID != coupleID {
			return Memory{}, ErrNotFound
		}
	}
	return m, nil
}

func (s *Service) Update(ctx context.Context, userID, id uuid.UUID, req UpdateMemoryRequest) (Memory, error) {
	m, err := s.Get(ctx, userID, id)
	if err != nil {
		return Memory{}, err
	}
	title := req.Title
	if title == "" {
		title = m.Title
	}
	content := req.Content
	if content == "" {
		content = m.Content
	}
	if err := s.repo.Update(ctx, id, title, content); err != nil {
		return Memory{}, err
	}
	return s.repo.Get(ctx, id)
}

func (s *Service) Delete(ctx context.Context, userID, id uuid.UUID) error {
	if _, err := s.Get(ctx, userID, id); err != nil {
		return err
	}
	return s.repo.Delete(ctx, id)
}
