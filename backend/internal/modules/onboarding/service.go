package onboarding

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
)

var ErrInvalidAnswers = errors.New("invalid onboarding answers")

type store interface {
	ListPublishedTests(context.Context) ([]repositoryTest, error)
	GetPublishedTest(context.Context, string, int) (repositoryTest, error)
	SaveResponse(context.Context, uuid.UUID, string, int, json.RawMessage) (Response, error)
	ListResponses(context.Context, uuid.UUID) ([]Response, error)
}

type Service struct {
	repo store
}

func NewService(repo store) *Service {
	return &Service{repo: repo}
}

func (s *Service) ListTests(ctx context.Context) ([]Test, error) {
	tests, err := s.repo.ListPublishedTests(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]Test, 0, len(tests))
	for _, test := range tests {
		result = append(result, Test{
			Slug:        test.Slug,
			Version:     test.Version,
			Title:       test.Title,
			Description: test.Description,
			Questions:   test.Questions,
		})
	}

	return result, nil
}

func (s *Service) SubmitAnswers(
	ctx context.Context,
	userID uuid.UUID,
	request SubmitAnswersRequest,
) (Response, error) {
	if !validSlug(request.TestSlug) || request.Version < 1 ||
		len(request.Answers) == 0 || len(request.Answers) > 200 {
		return Response{}, ErrInvalidAnswers
	}

	test, err := s.repo.GetPublishedTest(
		ctx,
		request.TestSlug,
		request.Version,
	)
	if err != nil {
		return Response{}, err
	}
	if err := validateAnswers(test.Questions, request.Answers); err != nil {
		return Response{}, err
	}

	answers, err := json.Marshal(request.Answers)
	if err != nil {
		return Response{}, fmt.Errorf("encode onboarding answers: %w", err)
	}

	return s.repo.SaveResponse(
		ctx,
		userID,
		request.TestSlug,
		request.Version,
		answers,
	)
}

func (s *Service) ListMyResponses(
	ctx context.Context,
	userID uuid.UUID,
) ([]Response, error) {
	return s.repo.ListResponses(ctx, userID)
}

func validSlug(slug string) bool {
	if len(slug) == 0 || len(slug) > 80 {
		return false
	}

	for index, char := range slug {
		valid := char >= 'a' && char <= 'z' ||
			char >= '0' && char <= '9' ||
			(index > 0 && (char == '-' || char == '_'))
		if !valid {
			return false
		}
	}

	return true
}

func validateAnswers(
	questions []Question,
	answers map[string]json.RawMessage,
) error {
	questionsByKey := make(map[string]Question, len(questions))
	for _, question := range questions {
		questionsByKey[question.Key] = question
	}

	for key, raw := range answers {
		question, ok := questionsByKey[key]
		if !ok || len(bytes.TrimSpace(raw)) == 0 {
			return ErrInvalidAnswers
		}
		if answerIsEmpty(raw) {
			if question.Required {
				return ErrInvalidAnswers
			}
			continue
		}
		if err := validateAnswer(question, raw); err != nil {
			return err
		}
	}

	for _, question := range questions {
		raw, ok := answers[question.Key]
		if question.Required && (!ok || answerIsEmpty(raw)) {
			return ErrInvalidAnswers
		}
	}

	return nil
}

func answerIsEmpty(raw json.RawMessage) bool {
	value := bytes.TrimSpace(raw)
	return len(value) == 0 ||
		bytes.Equal(value, []byte("null")) ||
		bytes.Equal(value, []byte(`""`)) ||
		bytes.Equal(value, []byte("[]"))
}

func validateAnswer(question Question, raw json.RawMessage) error {
	switch question.Type {
	case "single_choice":
		var value string
		if err := json.Unmarshal(raw, &value); err != nil || !hasOption(question.Options, value) {
			return ErrInvalidAnswers
		}
	case "multiple_choice":
		var values []string
		if err := json.Unmarshal(raw, &values); err != nil || len(values) == 0 {
			return ErrInvalidAnswers
		}
		seen := make(map[string]struct{}, len(values))
		for _, value := range values {
			if !hasOption(question.Options, value) {
				return ErrInvalidAnswers
			}
			if _, exists := seen[value]; exists {
				return ErrInvalidAnswers
			}
			seen[value] = struct{}{}
		}
	case "text":
		var value string
		if err := json.Unmarshal(raw, &value); err != nil {
			return ErrInvalidAnswers
		}
		if question.Required && strings.TrimSpace(value) == "" {
			return ErrInvalidAnswers
		}
		maxLength := question.Settings.MaxLength
		if maxLength == 0 {
			maxLength = 2000
		}
		if maxLength < 1 || maxLength > 10000 ||
			utf8.RuneCountInString(value) > maxLength {
			return ErrInvalidAnswers
		}
	case "scale", "number":
		value, err := decodeNumber(raw)
		if err != nil {
			return ErrInvalidAnswers
		}
		minimum, maximum := question.Settings.Min, question.Settings.Max
		if question.Type == "scale" {
			if minimum == nil {
				defaultMinimum := float64(1)
				minimum = &defaultMinimum
			}
			if maximum == nil {
				defaultMaximum := float64(5)
				maximum = &defaultMaximum
			}
			if math.Trunc(value) != value {
				return ErrInvalidAnswers
			}
		}
		if minimum != nil && value < *minimum ||
			maximum != nil && value > *maximum ||
			minimum != nil && maximum != nil && *minimum > *maximum {
			return ErrInvalidAnswers
		}
	case "boolean":
		var value bool
		if err := json.Unmarshal(raw, &value); err != nil {
			return ErrInvalidAnswers
		}
	default:
		return ErrInvalidAnswers
	}

	return nil
}

func hasOption(options []Option, value string) bool {
	for _, option := range options {
		if option.Value == value {
			return true
		}
	}
	return false
}

func decodeNumber(raw json.RawMessage) (float64, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()

	var value any
	if err := decoder.Decode(&value); err != nil {
		return 0, err
	}
	number, ok := value.(json.Number)
	if !ok {
		return 0, errors.New("answer is not a number")
	}
	parsed, err := strconv.ParseFloat(strings.TrimSpace(number.String()), 64)
	if err != nil || math.IsInf(parsed, 0) || math.IsNaN(parsed) {
		return 0, errors.New("answer is not a finite number")
	}
	return parsed, nil
}
