package content

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/W0LIF/LoveTea/backend/internal/platform/storage"
	"github.com/google/uuid"
)

var (
	ErrInvalidContent  = errors.New("invalid content")
	ErrNotFound        = errors.New("content not found")
	ErrNotReady        = errors.New("media is not ready")
	ErrVersionConflict = errors.New("content version conflict")
	ErrNoActiveCouple  = errors.New("user does not belong to an active couple")

	allowedTypes = map[string]struct{}{
		"image/jpeg": {},
		"image/png":  {},
	}
)

type repository interface {
	ListMedia(context.Context, uuid.UUID) ([]MediaFile, error)
	CreatePendingMedia(context.Context, uuid.UUID, uuid.UUID, string, string, string) (MediaFile, error)
	CompleteMedia(context.Context, uuid.UUID, uuid.UUID, int64) (MediaFile, string, error)
	GetMedia(context.Context, uuid.UUID, uuid.UUID) (MediaFile, string, error)
	DeleteMedia(context.Context, uuid.UUID, uuid.UUID, int64) (string, error)
	ListMoments(context.Context, uuid.UUID) ([]Moment, error)
	CreateMoment(context.Context, uuid.UUID, CreateMomentRequest, []uuid.UUID) (Moment, error)
}

type objectStore interface {
	CreateUploadForm(context.Context, string, string) (storage.UploadForm, error)
	CreateDownloadURL(context.Context, string) (string, time.Time, error)
	Inspect(context.Context, string) (storage.ObjectInfo, error)
	Delete(context.Context, string) error
}

type Service struct {
	repo  repository
	store objectStore
}

func NewService(repo repository, store objectStore) *Service {
	return &Service{repo: repo, store: store}
}

func (s *Service) ListMedia(ctx context.Context, userID uuid.UUID) ([]MediaFile, error) {
	return s.repo.ListMedia(ctx, userID)
}

func (s *Service) CreateUpload(
	ctx context.Context,
	userID uuid.UUID,
	request UploadRequest,
) (UploadResponse, error) {
	if !validScope(request.Scope) {
		return UploadResponse{}, ErrInvalidContent
	}
	if _, ok := allowedTypes[request.ContentType]; !ok {
		return UploadResponse{}, ErrInvalidContent
	}

	mediaID := uuid.New()
	objectKey := "media/" + mediaID.String()
	form, err := s.store.CreateUploadForm(ctx, objectKey, request.ContentType)
	if err != nil {
		return UploadResponse{}, err
	}
	media, err := s.repo.CreatePendingMedia(
		ctx,
		userID,
		mediaID,
		objectKey,
		request.ContentType,
		request.Scope,
	)
	if err != nil {
		return UploadResponse{}, err
	}
	return UploadResponse{
		Media:      media,
		UploadURL:  form.URL,
		FormFields: form.Fields,
		ExpiresAt:  form.ExpiresAt,
	}, nil
}

func (s *Service) CompleteUpload(
	ctx context.Context,
	userID uuid.UUID,
	mediaID uuid.UUID,
) (MediaFile, error) {
	if mediaID == uuid.Nil {
		return MediaFile{}, ErrInvalidContent
	}
	media, objectKey, err := s.repo.GetMedia(ctx, userID, mediaID)
	if err != nil {
		return MediaFile{}, err
	}
	if media.Status == "ready" {
		return media, nil
	}
	if media.Status != "pending" {
		return MediaFile{}, ErrInvalidContent
	}
	info, err := s.store.Inspect(ctx, objectKey)
	if err != nil {
		return MediaFile{}, err
	}
	if info.Size < 1 || info.Size > storage.MaxUploadBytes ||
		info.ContentType != media.ContentType ||
		(media.ContentType == "image/jpeg" && info.Format != "jpeg") ||
		(media.ContentType == "image/png" && info.Format != "png") {
		return MediaFile{}, ErrInvalidContent
	}
	media, _, err = s.repo.CompleteMedia(ctx, userID, mediaID, info.Size)
	return media, err
}

func (s *Service) CreateDownloadURL(
	ctx context.Context,
	userID uuid.UUID,
	mediaID uuid.UUID,
) (string, time.Time, error) {
	if mediaID == uuid.Nil {
		return "", time.Time{}, ErrInvalidContent
	}
	media, objectKey, err := s.repo.GetMedia(ctx, userID, mediaID)
	if err != nil {
		return "", time.Time{}, err
	}
	if media.Status != "ready" {
		return "", time.Time{}, ErrNotReady
	}
	return s.store.CreateDownloadURL(ctx, objectKey)
}

func (s *Service) DeleteMedia(
	ctx context.Context,
	userID uuid.UUID,
	mediaID uuid.UUID,
	version int64,
) error {
	if mediaID == uuid.Nil || version < 1 {
		return ErrInvalidContent
	}
	objectKey, err := s.repo.DeleteMedia(ctx, userID, mediaID, version)
	if err != nil {
		return err
	}
	return s.store.Delete(ctx, objectKey)
}

func (s *Service) ListMoments(ctx context.Context, userID uuid.UUID) ([]Moment, error) {
	return s.repo.ListMoments(ctx, userID)
}

func (s *Service) CreateMoment(
	ctx context.Context,
	userID uuid.UUID,
	request CreateMomentRequest,
) (Moment, error) {
	request.Title = strings.TrimSpace(request.Title)
	request.Caption = strings.TrimSpace(request.Caption)
	if !validScope(request.Scope) ||
		!validTitle(request.Title) ||
		utf8.RuneCountInString(request.Caption) > 5000 ||
		request.HappenedAt.IsZero() ||
		len(request.MediaIDs) > 20 {
		return Moment{}, ErrInvalidContent
	}
	mediaIDs := make([]uuid.UUID, 0, len(request.MediaIDs))
	seen := make(map[uuid.UUID]struct{}, len(request.MediaIDs))
	for _, rawID := range request.MediaIDs {
		id, err := uuid.Parse(rawID)
		if err != nil || id == uuid.Nil {
			return Moment{}, ErrInvalidContent
		}
		if _, exists := seen[id]; exists {
			return Moment{}, ErrInvalidContent
		}
		seen[id] = struct{}{}
		mediaIDs = append(mediaIDs, id)
	}
	return s.repo.CreateMoment(ctx, userID, request, mediaIDs)
}

func validScope(scope string) bool {
	return scope == "personal" || scope == "couple"
}

func validTitle(title string) bool {
	return utf8.RuneCountInString(title) >= 1 && utf8.RuneCountInString(title) <= 200
}
