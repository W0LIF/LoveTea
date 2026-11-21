package push

import (
	"context"
	"log/slog"
)

// Manager routes push notifications to the correct platform provider.
type Manager struct {
	providers map[string]Provider
	logger    *slog.Logger
}

func NewManager(logger *slog.Logger) *Manager {
	if logger == nil {
		logger = slog.Default()
	}
	return &Manager{
		providers: make(map[string]Provider),
		logger:    logger,
	}
}

func (m *Manager) Register(provider Provider) {
	m.providers[provider.Platform()] = provider
	m.logger.Info("push provider registered", "platform", provider.Platform())
}

func (m *Manager) Send(ctx context.Context, platform string, tokens []string, title, body string, data map[string]string) error {
	provider, ok := m.providers[platform]
	if !ok {
		m.logger.Debug("no push provider for platform", "platform", platform)
		return nil
	}
	return provider.Send(ctx, tokens, title, body, data)
}

func (m *Manager) Platforms() []string {
	platforms := make([]string, 0, len(m.providers))
	for p := range m.providers {
		platforms = append(platforms, p)
	}
	return platforms
}
