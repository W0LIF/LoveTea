package onboarding

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type Test struct {
	Slug        string     `json:"slug"`
	Version     int        `json:"version"`
	Title       Translated `json:"title"`
	Description Translated `json:"description"`
	Questions   []Question `json:"questions"`
}

type Question struct {
	Key      string     `json:"key"`
	Prompt   Translated `json:"prompt"`
	Type     string     `json:"type"`
	Position int        `json:"position"`
	Required bool       `json:"required"`
	Options  []Option   `json:"options"`
	Settings Settings   `json:"settings"`
}

type Option struct {
	Value string     `json:"value"`
	Label Translated `json:"label"`
}

type Translated map[string]string

type Settings struct {
	Min       *float64 `json:"min,omitempty"`
	Max       *float64 `json:"max,omitempty"`
	MaxLength int      `json:"max_length,omitempty"`
}

type SubmitAnswersRequest struct {
	TestSlug string                     `json:"test_slug"`
	Version  int                        `json:"version"`
	Answers  map[string]json.RawMessage `json:"answers"`
}

type Response struct {
	TestSlug    string                     `json:"test_slug"`
	TestVersion int                        `json:"test_version"`
	Answers     map[string]json.RawMessage `json:"answers"`
	SubmittedAt time.Time                  `json:"submitted_at"`
}

type repositoryTest struct {
	ID          uuid.UUID
	Slug        string
	Version     int
	Title       Translated
	Description Translated
	Questions   []Question
}
