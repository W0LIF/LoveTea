package wellness

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

type fakeRepository struct {
	moodRequest  SaveMoodRequest
	sleepRequest SaveSleepRequest
	err          error
}

func (r *fakeRepository) ListMood(context.Context, uuid.UUID, dateRange) ([]MoodEntry, error) {
	return []MoodEntry{}, r.err
}

func (r *fakeRepository) SaveMood(
	_ context.Context,
	_ uuid.UUID,
	request SaveMoodRequest,
) (MoodEntry, error) {
	r.moodRequest = request
	return MoodEntry{Mood: request.Mood, Note: request.Note}, r.err
}

func (r *fakeRepository) ListSleep(context.Context, uuid.UUID, dateRange) ([]SleepEntry, error) {
	return []SleepEntry{}, r.err
}

func (r *fakeRepository) SaveSleep(
	_ context.Context,
	_ uuid.UUID,
	request SaveSleepRequest,
) (SleepEntry, error) {
	r.sleepRequest = request
	return SleepEntry{Quality: request.Quality}, r.err
}

func TestSaveMoodValidatesAndNormalizesPrivateEntry(t *testing.T) {
	repo := &fakeRepository{}
	entry, err := NewService(repo).SaveMood(context.Background(), uuid.New(), SaveMoodRequest{
		EntryDate: "2026-05-01",
		Mood:      "  🙂  ",
		Note:      "  A private note  ",
	})
	if err != nil {
		t.Fatalf("SaveMood() error = %v", err)
	}
	if entry.Mood != "🙂" || entry.Note != "A private note" || repo.moodRequest.SharedWithCouple {
		t.Fatalf("SaveMood() did not normalize/default private entry: %+v", repo.moodRequest)
	}
}

func TestSaveMoodRejectsInvalidDateAndVersion(t *testing.T) {
	service := NewService(&fakeRepository{})
	version := int64(0)
	for _, request := range []SaveMoodRequest{
		{EntryDate: "2026-02-30", Mood: "good"},
		{EntryDate: "2026-02-28", Mood: "good", Version: &version},
		{EntryDate: "2026-02-28", Mood: "  "},
	} {
		if _, err := service.SaveMood(context.Background(), uuid.New(), request); !errors.Is(err, ErrInvalidEntry) {
			t.Errorf("SaveMood(%+v) error = %v, want invalid entry", request, err)
		}
	}
}

func TestSaveSleepRejectsQualityAndDurationOutsideBounds(t *testing.T) {
	service := NewService(&fakeRepository{})
	start := time.Date(2026, 5, 1, 21, 0, 0, 0, time.UTC)
	tests := []SaveSleepRequest{
		{EntryDate: "2026-05-02", SleepStartedAt: start, WokeAt: start.Add(8 * time.Hour), Quality: 0},
		{EntryDate: "2026-05-02", SleepStartedAt: start, WokeAt: start, Quality: 3},
		{EntryDate: "2026-05-02", SleepStartedAt: start, WokeAt: start.Add(25 * time.Hour), Quality: 3},
	}
	for _, request := range tests {
		if _, err := service.SaveSleep(context.Background(), uuid.New(), request); !errors.Is(err, ErrInvalidEntry) {
			t.Errorf("SaveSleep(%+v) error = %v, want invalid entry", request, err)
		}
	}
}

func TestParseDateRangeRejectsReversedAndOversizedRanges(t *testing.T) {
	tests := []struct {
		from string
		to   string
	}{
		{from: "2026-05-03", to: "2026-05-02"},
		{from: "2025-01-01", to: "2026-05-01"},
		{from: "2026-02-30", to: "2026-03-01"},
	}
	for _, test := range tests {
		if _, err := parseDateRange(test.from, test.to); !errors.Is(err, ErrInvalidDateRange) {
			t.Errorf("parseDateRange(%q, %q) error = %v, want invalid range", test.from, test.to, err)
		}
	}
}

func TestParseDateRangeDefaultsToThirtyDays(t *testing.T) {
	rangeDates, err := parseDateRange("", "")
	if err != nil {
		t.Fatalf("parseDateRange() error = %v", err)
	}
	if got := int(rangeDates.To.Sub(rangeDates.From).Hours()/24) + 1; got != 30 {
		t.Fatalf("default range has %d days, want 30", got)
	}
}
