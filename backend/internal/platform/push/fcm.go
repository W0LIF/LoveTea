package push

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

// FCMProvider sends push notifications via Firebase Cloud Messaging HTTP v1 API.
type FCMProvider struct {
	httpClient *http.Client
	projectID  string
	accessToken string
	logger     *slog.Logger
}

func NewFCMProvider(projectID, accessToken string, logger *slog.Logger) *FCMProvider {
	if logger == nil {
		logger = slog.Default()
	}
	return &FCMProvider{
		httpClient:  &http.Client{Timeout: 10 * time.Second},
		projectID:   projectID,
		accessToken: accessToken,
		logger:      logger,
	}
}

func (p *FCMProvider) Platform() string { return "android" }

func (p *FCMProvider) Send(ctx context.Context, tokens []string, title, body string, data map[string]string) error {
	if len(tokens) == 0 {
		return nil
	}
	for _, token := range tokens {
		if err := p.sendOne(ctx, token, title, body, data); err != nil {
			p.logger.Error("FCM send failed", "token", maskToken(token), "error", err)
		}
	}
	return nil
}

func (p *FCMProvider) sendOne(ctx context.Context, token, title, body string, data map[string]string) error {
	payload := map[string]any{
		"message": map[string]any{
			"token": token,
			"notification": map[string]string{
				"title": title,
				"body":  body,
			},
			"data": data,
			"android": map[string]any{
				"priority": "high",
			},
			"apns": map[string]any{
				"headers": map[string]string{
					"apns-priority": "10",
				},
			},
		},
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal FCM payload: %w", err)
	}

	url := fmt.Sprintf("https://fcm.googleapis.com/v1/projects/%s/messages:send", p.projectID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(string(bodyBytes)))
	if err != nil {
		return fmt.Errorf("create FCM request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+p.accessToken)
	req.Header.Set("Content-Type", "application/json; UTF-8")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("FCM request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		io.Copy(io.Discard, resp.Body)
		return nil
	}

	respBody, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("FCM returned status %d: %s", resp.StatusCode, string(respBody))
}

func maskToken(token string) string {
	if len(token) <= 8 {
		return "***"
	}
	return token[:4] + "..." + token[len(token)-4:]
}
