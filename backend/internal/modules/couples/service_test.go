package couples

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
)

func TestAcceptInvitationRejectsInvalidInputBeforeRateLimit(t *testing.T) {
	service := NewService(nil, "secret", nil)
	userID := uuid.New()

	tests := []struct {
		name    string
		request AcceptInvitationRequest
	}{
		{name: "missing credentials"},
		{name: "both credentials", request: AcceptInvitationRequest{Code: "123456", Token: "token"}},
		{name: "malformed code", request: AcceptInvitationRequest{Code: "12x"}},
		{name: "short token", request: AcceptInvitationRequest{Token: "short"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := service.AcceptInvitation(context.Background(), userID, test.request)
			if !errors.Is(err, ErrInvalidInvitationInput) {
				t.Fatalf("AcceptInvitation() error = %v, want invalid input", err)
			}
		})
	}
}

func TestUpdateMemberProfileRejectsInvalidValuesBeforeRepositoryAccess(t *testing.T) {
	service := NewService(nil, "secret", nil)
	blank := "  "
	invalidMascot := "fox"

	tests := []UpdateMemberProfileRequest{
		{},
		{Nickname: &blank},
		{MascotType: &invalidMascot},
	}

	for _, request := range tests {
		if err := service.UpdateMemberProfile(context.Background(), uuid.New(), request); !errors.Is(err, ErrInvalidMemberProfile) {
			t.Fatalf("UpdateMemberProfile(%+v) error = %v, want invalid profile", request, err)
		}
	}
}
