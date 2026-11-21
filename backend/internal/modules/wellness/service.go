package wellness

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

var (
	ErrInvalidEntry     = errors.New("invalid wellness entry")
	ErrInvalidDateRange = errors.New("invalid wellness date range")
	ErrVersionConflict  = errors.New("wellness entry version conflict")
	ErrNoActiveCouple   = errors.New("user does not belong to an active couple")
)

type repository interface {
	ListMood(context.Context, uuid.UUID, dateRange) ([]MoodEntry, error)
	SaveMood(context.Context, uuid.UUID, SaveMoodRequest) (MoodEntry, error)
	ListSleep(context.Context, uuid.UUID, dateRange) ([]SleepEntry, error)
	SaveSleep(context.Context, uuid.UUID, SaveSleepRequest) (SleepEntry, error)
}

type Service struct {
	repo repository
}

func NewService(repo repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) ListMood(
	ctx context.Context,
	userID uuid.UUID,
	from string,
	to string,
) ([]MoodEntry, error) {
	dateRange, err := parseDateRange(from, to)
	if err != nil {
		return nil, err
	}
	return s.repo.ListMood(ctx, userID, dateRange)
}

func (s *Service) SaveMood(
	ctx context.Context,
	userID uuid.UUID,
	request SaveMoodRequest,
) (MoodEntry, error) {
	date, err := time.Parse("2006-01-02", request.EntryDate)
	if err != nil || date.Format("2006-01-02") != request.EntryDate {
		return MoodEntry{}, ErrInvalidEntry
	}
	request.Mood = strings.TrimSpace(request.Mood)
	request.Note = strings.TrimSpace(request.Note)
	if utf8.RuneCountInString(request.Mood) < 1 ||
		utf8.RuneCountInString(request.Mood) > 32 ||
		utf8.RuneCountInString(request.Note) > 2000 ||
		(request.Version != nil && *request.Version < 1) {
		return MoodEntry{}, ErrInvalidEntry
	}
	return s.repo.SaveMood(ctx, userID, request)
}

func (s *Service) ListSleep(
	ctx context.Context,
	userID uuid.UUID,
	from string,
	to string,
) ([]SleepEntry, error) {
	dateRange, err := parseDateRange(from, to)
	if err != nil {
		return nil, err
	}
	return s.repo.ListSleep(ctx, userID, dateRange)
}

func (s *Service) SaveSleep(
	ctx context.Context,
	userID uuid.UUID,
	request SaveSleepRequest,
) (SleepEntry, error) {
	date, err := time.Parse("2006-01-02", request.EntryDate)
	if err != nil || date.Format("2006-01-02") != request.EntryDate {
		return SleepEntry{}, ErrInvalidEntry
	}
	if request.Quality < 1 || request.Quality > 5 ||
		request.SleepStartedAt.IsZero() ||
		!request.WokeAt.After(request.SleepStartedAt) ||
		request.WokeAt.Sub(request.SleepStartedAt) > 24*time.Hour ||
		(request.Version != nil && *request.Version < 1) {
		return SleepEntry{}, ErrInvalidEntry
	}
	return s.repo.SaveSleep(ctx, userID, request)
}
