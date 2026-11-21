package sync

import (
	"context"

	"github.com/google/uuid"
)

type reader interface {
	Pull(context.Context, uuid.UUID, int64, int) (Page, error)
}

type Service struct {
	repo reader
}

func NewService(repo reader) *Service {
	return &Service{repo: repo}
}

func (s *Service) Pull(
	ctx context.Context,
	userID uuid.UUID,
	cursor int64,
	limit int,
) (Page, error) {
	if err := validatePage(cursor, limit); err != nil {
		return Page{}, err
	}
	return s.repo.Pull(ctx, userID, cursor, limit)
}
