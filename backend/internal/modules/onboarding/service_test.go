package onboarding

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

type fakeStore struct {
	test         repositoryTest
	testErr      error
	savedUserID  uuid.UUID
	savedAnswers json.RawMessage
	response     Response
}

func (s *fakeStore) ListPublishedTests(context.Context) ([]repositoryTest, error) {
	return []repositoryTest{s.test}, nil
}

func (s *fakeStore) GetPublishedTest(
	_ context.Context,
	slug string,
	version int,
) (repositoryTest, error) {
	if s.testErr != nil {
		return repositoryTest{}, s.testErr
	}
	if s.test.Slug != slug || s.test.Version != version {
		return repositoryTest{}, ErrTestNotFound
	}
	return s.test, nil
}

func (s *fakeStore) SaveResponse(
	_ context.Context,
	userID uuid.UUID,
	slug string,
	version int,
	answers json.RawMessage,
) (Response, error) {
	s.savedUserID = userID
	s.savedAnswers = append(json.RawMessage(nil), answers...)
	return Response{
		TestSlug:    slug,
		TestVersion: version,
		SubmittedAt: time.Now(),
	}, nil
}

func (s *fakeStore) ListResponses(context.Context, uuid.UUID) ([]Response, error) {
	return []Response{s.response}, nil
}

func TestValidateAnswersAcceptsAllSupportedTypes(t *testing.T) {
	minimum, maximum := 0.0, 10.0
	questions := []Question{
		{
			Key: "language", Type: "single_choice", Required: true,
			Options: []Option{{Value: "time"}, {Value: "words"}},
		},
		{
			Key: "interests", Type: "multiple_choice", Required: true,
			Options: []Option{{Value: "walk"}, {Value: "food"}},
		},
		{Key: "note", Type: "text", Required: true, Settings: Settings{MaxLength: 20}},
		{Key: "comfort", Type: "scale", Required: true},
		{Key: "budget", Type: "number", Required: true, Settings: Settings{Min: &minimum, Max: &maximum}},
		{Key: "reminders", Type: "boolean", Required: true},
	}
	answers := map[string]json.RawMessage{
		"language":  json.RawMessage(`"time"`),
		"interests": json.RawMessage(`["walk","food"]`),
		"note":      json.RawMessage(`"quiet evening"`),
		"comfort":   json.RawMessage(`4`),
		"budget":    json.RawMessage(`6.5`),
		"reminders": json.RawMessage(`false`),
	}

	if err := validateAnswers(questions, answers); err != nil {
		t.Fatalf("validateAnswers() error = %v", err)
	}
}

func TestValidateAnswersRejectsInvalidValues(t *testing.T) {
	questions := []Question{
		{
			Key: "choice", Type: "single_choice", Required: true,
			Options: []Option{{Value: "one"}, {Value: "two"}},
		},
		{
			Key: "many", Type: "multiple_choice", Required: false,
			Options: []Option{{Value: "one"}, {Value: "two"}},
		},
		{Key: "scale", Type: "scale", Required: true},
	}
	tests := []struct {
		name    string
		answers map[string]json.RawMessage
	}{
		{
			name:    "missing required answer",
			answers: map[string]json.RawMessage{"choice": json.RawMessage(`"one"`)},
		},
		{
			name: "unknown question",
			answers: map[string]json.RawMessage{
				"choice":  json.RawMessage(`"one"`),
				"unknown": json.RawMessage(`"value"`),
				"scale":   json.RawMessage(`3`),
			},
		},
		{
			name: "unknown option",
			answers: map[string]json.RawMessage{
				"choice": json.RawMessage(`"three"`),
				"scale":  json.RawMessage(`3`),
			},
		},
		{
			name: "duplicate multiple choice",
			answers: map[string]json.RawMessage{
				"choice": json.RawMessage(`"one"`),
				"many":   json.RawMessage(`["one","one"]`),
				"scale":  json.RawMessage(`3`),
			},
		},
		{
			name: "scale out of range",
			answers: map[string]json.RawMessage{
				"choice": json.RawMessage(`"one"`),
				"scale":  json.RawMessage(`6`),
			},
		},
		{
			name: "fractional scale",
			answers: map[string]json.RawMessage{
				"choice": json.RawMessage(`"one"`),
				"scale":  json.RawMessage(`2.5`),
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := validateAnswers(questions, test.answers); !errors.Is(err, ErrInvalidAnswers) {
				t.Fatalf("validateAnswers() error = %v, want invalid answers", err)
			}
		})
	}
}

func TestSubmitAnswersStoresOnlyAuthenticatedUserResponse(t *testing.T) {
	repo := &fakeStore{
		test: repositoryTest{
			Slug:    "preferences",
			Version: 2,
			Questions: []Question{
				{
					Key: "activity", Type: "single_choice", Required: true,
					Options: []Option{{Value: "walk"}},
				},
			},
		},
	}
	service := NewService(repo)
	userID := uuid.New()

	response, err := service.SubmitAnswers(context.Background(), userID, SubmitAnswersRequest{
		TestSlug: "preferences",
		Version:  2,
		Answers:  map[string]json.RawMessage{"activity": json.RawMessage(`"walk"`)},
	})
	if err != nil {
		t.Fatalf("SubmitAnswers() error = %v", err)
	}
	if repo.savedUserID != userID {
		t.Fatalf("saved user id = %s, want authenticated user %s", repo.savedUserID, userID)
	}
	if response.TestSlug != "preferences" || response.TestVersion != 2 {
		t.Fatalf("unexpected response: %+v", response)
	}
	if string(repo.savedAnswers) != `{"activity":"walk"}` {
		t.Fatalf("saved answers = %s", repo.savedAnswers)
	}
}

func TestSubmitAnswersRejectsUnpublishedOrMalformedTestReference(t *testing.T) {
	service := NewService(&fakeStore{})
	tests := []SubmitAnswersRequest{
		{TestSlug: "not a slug", Version: 1, Answers: map[string]json.RawMessage{"q": json.RawMessage(`true`)}},
		{TestSlug: "valid-slug", Version: 0, Answers: map[string]json.RawMessage{"q": json.RawMessage(`true`)}},
		{TestSlug: "valid-slug", Version: 1},
	}

	for _, request := range tests {
		if _, err := service.SubmitAnswers(context.Background(), uuid.New(), request); !errors.Is(err, ErrInvalidAnswers) {
			t.Fatalf("SubmitAnswers(%+v) error = %v, want invalid answers", request, err)
		}
	}
}
