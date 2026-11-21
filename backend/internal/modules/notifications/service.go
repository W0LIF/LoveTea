package notifications

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	"github.com/W0LIF/LoveTea/backend/internal/platform/push"
	"github.com/google/uuid"
)

type Service struct {
	repo    *Repository
	pushMgr *push.Manager
	logger  *slog.Logger
}

func NewService(repo *Repository, pushMgr *push.Manager, logger *slog.Logger) *Service {
	if logger == nil {
		logger = slog.Default()
	}
	return &Service{repo: repo, pushMgr: pushMgr, logger: logger}
}

func (s *Service) List(ctx context.Context, userID uuid.UUID, limit int, unreadOnly bool) ([]Notification, error) {
	return s.repo.List(ctx, userID, limit, unreadOnly)
}

func (s *Service) UnreadCount(ctx context.Context, userID uuid.UUID) (int, error) {
	return s.repo.UnreadCount(ctx, userID)
}

func (s *Service) MarkRead(ctx context.Context, userID uuid.UUID, id int64) error {
	return s.repo.MarkRead(ctx, userID, id)
}

func (s *Service) MarkAllRead(ctx context.Context, userID uuid.UUID) error {
	return s.repo.MarkAllRead(ctx, userID)
}

func (s *Service) GetSettings(ctx context.Context, userID uuid.UUID) (NotificationSettings, error) {
	settings, err := s.repo.GetSettings(ctx, userID)
	if err != nil {
		if isNotFound(err) {
			return NotificationSettings{
				UserID:            userID.String(),
				EnabledCategories: json.RawMessage(`["mood","task","event","date","habit","message","achievement"]`),
				QuietHoursStart:   22,
				QuietHoursEnd:     7,
			}, nil
		}
		return NotificationSettings{}, err
	}
	return settings, nil
}

func (s *Service) UpdateSettings(
	ctx context.Context,
	userID uuid.UUID,
	req UpdateSettingsRequest,
) (NotificationSettings, error) {
	settings, err := s.GetSettings(ctx, userID)
	if err != nil {
		return NotificationSettings{}, err
	}
	if req.EnabledCategories != nil {
		data, _ := json.Marshal(req.EnabledCategories)
		settings.EnabledCategories = data
	}
	if req.QuietHoursStart != nil {
		settings.QuietHoursStart = *req.QuietHoursStart
	}
	if req.QuietHoursEnd != nil {
		settings.QuietHoursEnd = *req.QuietHoursEnd
	}
	if err := s.repo.UpsertSettings(ctx, userID, settings); err != nil {
		return NotificationSettings{}, err
	}
	return s.GetSettings(ctx, userID)
}

func (s *Service) Create(ctx context.Context, userID uuid.UUID, req CreateNotificationRequest) (Notification, error) {
	notif := Notification{
		UserID:   userID.String(),
		Category: req.Category,
		Title:    req.Title,
		Body:     req.Body,
		Data:     req.Data,
	}
	if notif.Data == nil {
		notif.Data = json.RawMessage(`{}`)
	}
	created, err := s.repo.Create(ctx, notif)
	if err != nil {
		return Notification{}, err
	}

	// Send push notification if enabled and not in quiet hours
	settings, err := s.GetSettings(ctx, userID)
	if err == nil {
		enabled := false
		var cats []string
		_ = json.Unmarshal(settings.EnabledCategories, &cats)
		for _, c := range cats {
			if c == req.Category {
				enabled = true
				break
			}
		}
		if enabled {
			now := time.Now().UTC()
			if now.Hour() >= settings.QuietHoursStart || now.Hour() < settings.QuietHoursEnd {
				// In quiet hours, skip push
			} else {
				tokens, err := s.repo.DeviceTokens(ctx, userID)
				if err == nil && len(tokens) > 0 {
					dataMap := make(map[string]string)
					_ = json.Unmarshal(notif.Data, &dataMap)
					if err := s.pushMgr.Send(ctx, "android", tokens, req.Title, req.Body, dataMap); err != nil {
						s.logger.Error("push send failed", "user_id", userID, "error", err)
					}
				}
			}
		}
	}

	return created, nil
}

func isNotFound(err error) bool {
	return errors.Is(err, context.DeadlineExceeded) ||
		err.Error() == "pgx: no rows in result set"
}
