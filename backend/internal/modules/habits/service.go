package habits

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

var (
	ErrInvalidRequest  = errors.New("invalid habit request")
	ErrNotFound        = errors.New("habit not found")
	ErrVersionConflict = errors.New("habit version conflict")
	ErrNoActiveCouple  = errors.New("user does not belong to an active couple")
)

type repository interface {
	List(context.Context, uuid.UUID) ([]Habit, error)
	Create(context.Context, uuid.UUID, CreateHabitRequest) (Habit, error)
	Update(context.Context, uuid.UUID, uuid.UUID, UpdateHabitRequest) (Habit, error)
	Delete(context.Context, uuid.UUID, uuid.UUID, int64) error
	ListCompletions(context.Context, uuid.UUID, uuid.UUID, dateRange) ([]Completion, error)
	Complete(context.Context, uuid.UUID, uuid.UUID, time.Time) (Completion, error)
	Uncomplete(context.Context, uuid.UUID, uuid.UUID, time.Time) error
	Stats(context.Context, uuid.UUID, uuid.UUID, dateRange) (Habit, []string, []completionRecord, error)
}

type Service struct {
	repo repository
}

func NewService(repo repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) List(ctx context.Context, userID uuid.UUID) ([]Habit, error) {
	return s.repo.List(ctx, userID)
}

func (s *Service) Create(
	ctx context.Context,
	userID uuid.UUID,
	request CreateHabitRequest,
) (Habit, error) {
	request.Title = strings.TrimSpace(request.Title)
	request.Description = strings.TrimSpace(request.Description)
	if request.TargetPerWeek == nil {
		defaultTarget := 7
		request.TargetPerWeek = &defaultTarget
	}
	if !validScope(request.Scope) ||
		!validTitle(request.Title) ||
		utf8.RuneCountInString(request.Description) > 2000 ||
		*request.TargetPerWeek < 1 || *request.TargetPerWeek > 7 {
		return Habit{}, ErrInvalidRequest
	}
	return s.repo.Create(ctx, userID, request)
}

func (s *Service) Update(
	ctx context.Context,
	userID uuid.UUID,
	habitID uuid.UUID,
	request UpdateHabitRequest,
) (Habit, error) {
	if habitID == uuid.Nil || request.Version < 1 ||
		request.Title == nil || request.Description == nil || request.TargetPerWeek == nil {
		return Habit{}, ErrInvalidRequest
	}
	title := strings.TrimSpace(*request.Title)
	description := strings.TrimSpace(*request.Description)
	if !validTitle(title) ||
		utf8.RuneCountInString(description) > 2000 ||
		*request.TargetPerWeek < 1 || *request.TargetPerWeek > 7 {
		return Habit{}, ErrInvalidRequest
	}
	request.Title = &title
	request.Description = &description
	return s.repo.Update(ctx, userID, habitID, request)
}

func (s *Service) Delete(
	ctx context.Context,
	userID uuid.UUID,
	habitID uuid.UUID,
	version int64,
) error {
	if habitID == uuid.Nil || version < 1 {
		return ErrInvalidRequest
	}
	return s.repo.Delete(ctx, userID, habitID, version)
}

func (s *Service) ListCompletions(
	ctx context.Context,
	userID uuid.UUID,
	habitID uuid.UUID,
	from string,
	to string,
) ([]Completion, error) {
	if habitID == uuid.Nil {
		return nil, ErrInvalidRequest
	}
	rangeDates, err := parseDateRange(from, to)
	if err != nil {
		return nil, err
	}
	return s.repo.ListCompletions(ctx, userID, habitID, rangeDates)
}

func (s *Service) Complete(
	ctx context.Context,
	userID uuid.UUID,
	habitID uuid.UUID,
	request CompleteHabitRequest,
) (Completion, error) {
	if habitID == uuid.Nil {
		return Completion{}, ErrInvalidRequest
	}
	date, err := parseDate(request.CompletedOn)
	if err != nil {
		return Completion{}, ErrInvalidRequest
	}
	return s.repo.Complete(ctx, userID, habitID, date)
}

func (s *Service) Uncomplete(
	ctx context.Context,
	userID uuid.UUID,
	habitID uuid.UUID,
	completedOn string,
) error {
	if habitID == uuid.Nil {
		return ErrInvalidRequest
	}
	date, err := parseDate(completedOn)
	if err != nil {
		return ErrInvalidRequest
	}
	return s.repo.Uncomplete(ctx, userID, habitID, date)
}

func (s *Service) Stats(
	ctx context.Context,
	userID uuid.UUID,
	habitID uuid.UUID,
	from string,
	to string,
) (HabitStats, error) {
	if habitID == uuid.Nil {
		return HabitStats{}, ErrInvalidRequest
	}
	rangeDates, err := parseDateRange(from, to)
	if err != nil {
		return HabitStats{}, err
	}
	habit, memberIDs, records, err := s.repo.Stats(ctx, userID, habitID, rangeDates)
	if err != nil {
		return HabitStats{}, err
	}
	return summarize(habit, memberIDs, records, rangeDates, time.Now().UTC()), nil
}

func summarize(habit Habit, memberIDs []string, records []completionRecord, requested dateRange, now time.Time) HabitStats {
	stats := HabitStats{
		HabitID:       habit.ID,
		From:          requested.From.Format("2006-01-02"),
		To:            requested.To.Format("2006-01-02"),
		TargetPerWeek: habit.TargetPerWeek,
		Members:       make([]HabitMemberStats, 0),
	}
	type memberData struct {
		stats      HabitMemberStats
		weeks      map[time.Time]int
		rangeWeeks map[time.Time]int
	}
	members := make(map[string]*memberData)
	for _, userID := range memberIDs {
		members[userID] = &memberData{
			stats:      HabitMemberStats{UserID: userID},
			weeks:      make(map[time.Time]int),
			rangeWeeks: make(map[time.Time]int),
		}
	}
	for _, record := range records {
		member := members[record.UserID]
		if member == nil {
			member = &memberData{
				stats:      HabitMemberStats{UserID: record.UserID},
				weeks:      make(map[time.Time]int),
				rangeWeeks: make(map[time.Time]int),
			}
			members[record.UserID] = member
		}
		week := weekStart(record.Date)
		member.weeks[week]++
		if !record.Date.Before(requested.From) && !record.Date.After(requested.To) {
			member.stats.CompletionCount++
			member.rangeWeeks[week]++
			stats.TotalCompletions++
		}
	}

	currentWeek := weekStart(now)
	for _, member := range members {
		for week, count := range member.rangeWeeks {
			if !week.Before(weekStart(requested.From)) &&
				!week.After(weekStart(requested.To)) &&
				count >= habit.TargetPerWeek {
				member.stats.SuccessfulWeeks++
			}
		}
		candidate := currentWeek
		if member.weeks[candidate] < habit.TargetPerWeek {
			candidate = candidate.AddDate(0, 0, -7)
		}
		for member.weeks[candidate] >= habit.TargetPerWeek {
			member.stats.CurrentStreakWeeks++
			candidate = candidate.AddDate(0, 0, -7)
		}
		stats.Members = append(stats.Members, member.stats)
	}
	sortMemberStats(stats.Members)
	return stats
}

func sortMemberStats(members []HabitMemberStats) {
	sort.Slice(members, func(i, j int) bool {
		return members[i].UserID < members[j].UserID
	})
}

func weekStart(date time.Time) time.Time {
	date = time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
	offset := (int(date.Weekday()) + 6) % 7
	return date.AddDate(0, 0, -offset)
}

func validScope(scope string) bool {
	return scope == "personal" || scope == "couple"
}

func validTitle(title string) bool {
	return utf8.RuneCountInString(title) >= 1 && utf8.RuneCountInString(title) <= 200
}

func parseDate(value string) (time.Time, error) {
	date, err := time.Parse("2006-01-02", value)
	if err != nil || date.Format("2006-01-02") != value {
		return time.Time{}, ErrInvalidRequest
	}
	return date, nil
}

func parseDateRange(from, to string) (dateRange, error) {
	now := time.Now().UTC().Truncate(24 * time.Hour)
	start, end := now.AddDate(0, 0, -29), now
	if from != "" {
		parsed, err := parseDate(from)
		if err != nil {
			return dateRange{}, ErrInvalidRequest
		}
		start = parsed
	}
	if to != "" {
		parsed, err := parseDate(to)
		if err != nil {
			return dateRange{}, ErrInvalidRequest
		}
		end = parsed
	}
	if start.After(end) || end.Sub(start) > 365*24*time.Hour {
		return dateRange{}, ErrInvalidRequest
	}
	return dateRange{From: start, To: end}, nil
}
