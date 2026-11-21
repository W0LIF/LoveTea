package users

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
)

func TestUpdateRejectsEmptyPatchBeforeRepositoryAccess(t *testing.T) {
	service := NewService(nil)

	_, err := service.Update(context.Background(), uuid.New(), UpdateProfileRequest{})
	if !errors.Is(err, ErrInvalidProfile) {
		t.Fatalf("Update() error = %v, want invalid profile", err)
	}
}
