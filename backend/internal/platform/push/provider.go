package push

import (
	"context"

	"github.com/google/uuid"
)

// Provider sends push notifications to a specific platform.
type Provider interface {
	Platform() string
	Send(ctx context.Context, tokens []string, title, body string, data map[string]string) error
}

// Result of a send attempt.
type SendResult struct {
	Success int
	Failure int
	Tokens  []string // tokens that failed and should be removed
}

// NoopProvider is a provider that does nothing (used when push is not configured).
type NoopProvider struct{}

func (NoopProvider) Platform() string { return "noop" }

func (NoopProvider) Send(_ context.Context, _ []string, _, _ string, _ map[string]string) error {
	return nil
}

var _ = uuid.Nil
