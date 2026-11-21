package push

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

// APNsProvider sends push notifications via Apple Push Notification service.
type APNsProvider struct {
	httpClient *http.Client
	topic      string
	logger     *slog.Logger
}

func NewAPNsProvider(topic string, logger *slog.Logger) *APNsProvider {
	if logger == nil {
		logger = slog.Default()
	}
	return &APNsProvider{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		},
		topic: topic,
		logger: logger,
	}
}

func (p *APNsProvider) Platform() string { return "ios" }

func (p *APNsProvider) Send(ctx context.Context, tokens []string, title, body string, data map[string]string) error {
	if len(tokens) == 0 {
		return nil
	}
	for _, token := range tokens {
		if err := p.sendOne(ctx, token, title, body, data); err != nil {
			p.logger.Error("APNs send failed", "token", maskToken(token), "error", err)
		}
	}
	return nil
}

func (p *APNsProvider) sendOne(ctx context.Context, deviceToken, title, body string, data map[string]string) error {
	payload := map[string]any{
		"aps": map[string]any{
			"alert": map[string]string{
				"title": title,
				"body":  body,
			},
			"sound": "default",
			"badge": 1,
		},
	}
	if len(data) > 0 {
		payload["data"] = data
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal APNs payload: %w", err)
	}

	url := fmt.Sprintf("https://api.sandbox.push.apple.com/3/device/%s", deviceToken)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(string(bodyBytes)))
	if err != nil {
		return fmt.Errorf("create APNs request: %w", err)
	}
	req.Header.Set("apns-topic", p.topic)
	req.Header.Set("apns-push-type", "alert")
	req.Header.Set("apns-priority", "10")
	req.Header.Set("Content-Type", "application/json; UTF-8")
	// In production, a valid JWT bearer token must be set here:
	// req.Header.Set("authorization", "bearer "+authToken)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("APNs request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		io.Copy(io.Discard, resp.Body)
		return nil
	}

	respBody, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("APNs returned status %d: %s", resp.StatusCode, string(respBody))
}

var _ = strings.NewReader
