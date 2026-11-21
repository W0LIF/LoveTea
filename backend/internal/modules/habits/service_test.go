package habits

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

type fakeRepository struct {
	created CreateHabitRequest
	updated UpdateHabitRequest
	err     error
	habit   Habit
	records []completionRecord
}

func (r *fakeRepository) List(context.Context, uuid.UUID) ([]Habit, error) {
	return []Habit{}, r.err
}

func (r *fakeRepository) Create(
	_ context.Context,
	_ uuid.UUID,
	request CreateHabitRequest,
) (Habit, error) {
	r.created = request
	return Habit{Title: request.Title, Scope: request.Scope}, r.err
}

func (r *fakeRepository) Update(
	_ context.Context,
	_ uuid.UUID,
	_ uuid.UUID,
	request UpdateHabitRequest,
) (Habit, error) {
	r.updated = request
	return Habit{Title: *request.Title}, r.err
}

func (r *fakeRepository) Delete(context.Context, uuid.UUID, uuid.UUID, int64) error {
	return r.err
}

func (r *fakeRepository) ListCompletions(
	context.Context,
	uuid.UUID,
	uuid.UUID,
	dateRange,
) ([]Completion, error) {
	return []Completion{}, r.err
}

func (r *fakeRepository) Complete(
	_ context.Context,
	_ uuid.UUID,
	_ uuid.UUID,
	date time.Time,
) (Completion, error) {
	return Completion{CompletedOn: date.Format("2006-01-02")}, r.err
}

func (r *fakeRepository) Uncomplete(context.Context, uuid.UUID, uuid.UUID, time.Time) error {
	return r.err
}

func (r *fakeRepository) Stats(
	context.Context,
	uuid.UUID,
	uuid.UUID,
	dateRange,
) (Habit, []string, []completionRecord, error) {
	return r.habit, []string{}, r.records, r.err
}

func TestCreateNormalizesValuesAndValidatesWeeklyTarget(t *testing.T) {
	repo := &fakeRepository{}
	target := 3
	habit, err := NewService(repo).Create(context.Background(), uuid.New(), CreateHabitRequest{
		Scope:         "couple",
		Title:         "  Read together  ",
		Description:   "  Ten pages  ",
		TargetPerWeek: &target,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if habit.Title != "Read together" ||
		repo.created.Description != "Ten pages" ||
		*repo.created.TargetPerWeek != 3 {
		t.Fatalf("Create() did not normalize request: %+v", repo.created)
	}
	if _, err := NewService(&fakeRepository{}).Create(context.Background(), uuid.New(), CreateHabitRequest{
		Scope: "personal", Title: "Walk", TargetPerWeek: intPointer(8),
	}); !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("Create() error = %v, want invalid request", err)
	}
}

func intPointer(value int) *int {
	return &value
}

func TestUpdateRequiresCompleteVersionedRequest(t *testing.T) {
	repo := &fakeRepository{}
	title, description, target := "  Stretch  ", "", 4
	habit, err := NewService(repo).Update(context.Background(), uuid.New(), uuid.New(), UpdateHabitRequest{
		Version: 2, Title: &title, Description: &description, TargetPerWeek: &target,
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if habit.Title != "Stretch" || *repo.updated.Title != "Stretch" {
		t.Fatalf("Update() did not normalize title: %+v", repo.updated)
	}
	repo = &fakeRepository{}
	if _, err := NewService(repo).Update(context.Background(), uuid.New(), uuid.New(), UpdateHabitRequest{
		Version: 2, Title: &title, TargetPerWeek: &target,
	}); !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("Update() missing field error = %v, want invalid request", err)
	}
}

func TestCompleteRequiresISODate(t *testing.T) {
	service := NewService(&fakeRepository{})
	if _, err := service.Complete(context.Background(), uuid.New(), uuid.New(), CompleteHabitRequest{
		CompletedOn: "2026-02-30",
	}); !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("Complete() error = %v, want invalid request", err)
	}
}

func TestSummarizeCountsCompletionsAndWeeklyStreaks(t *testing.T) {
	userID := uuid.New().String()
	rangeDates := dateRange{
		From: time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC),
	}
	records := []completionRecord{
		{UserID: userID, Date: time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC)},
		{UserID: userID, Date: time.Date(2026, 3, 4, 0, 0, 0, 0, time.UTC)},
		{UserID: userID, Date: time.Date(2026, 3, 9, 0, 0, 0, 0, time.UTC)},
		{UserID: userID, Date: time.Date(2026, 3, 11, 0, 0, 0, 0, time.UTC)},
	}
	stats := summarize(
		Habit{ID: uuid.New().String(), TargetPerWeek: 2},
		[]string{userID},
		records,
		rangeDates,
		time.Date(2026, 3, 12, 0, 0, 0, 0, time.UTC),
	)
	if stats.TotalCompletions != 4 || len(stats.Members) != 1 ||
		stats.Members[0].SuccessfulWeeks != 2 ||
		stats.Members[0].CurrentStreakWeeks != 2 {
		t.Fatalf("unexpected habit statistics: %+v", stats)
	}
}

func TestParseDateRangeRejectsInvalidAndOversizedRanges(t *testing.T) {
	for _, test := range []struct {
		from string
		to   string
	}{
		{from: "2026-03-05", to: "2026-03-04"},
		{from: "2025-01-01", to: "2026-03-01"},
		{from: "2026-02-30", to: "2026-03-01"},
	} {
		if _, err := parseDateRange(test.from, test.to); !errors.Is(err, ErrInvalidRequest) {
			t.Errorf("parseDateRange(%q, %q) error = %v, want invalid request", test.from, test.to, err)
		}
	}
}
