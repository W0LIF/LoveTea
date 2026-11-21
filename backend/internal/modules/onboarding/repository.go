package onboarding

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrTestNotFound = errors.New("onboarding test not found")

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) ListPublishedTests(ctx context.Context) ([]repositoryTest, error) {
	rows, err := r.db.Query(ctx, `
		SELECT current.id, current.slug, current.version,
		       current.title, current.description
		FROM onboarding_tests current
		JOIN (
			SELECT slug, MAX(version) AS version
			FROM onboarding_tests
			WHERE published_at IS NOT NULL
			GROUP BY slug
		) latest ON latest.slug = current.slug
		        AND latest.version = current.version
		WHERE current.published_at IS NOT NULL
		ORDER BY current.slug
	`)
	if err != nil {
		return nil, fmt.Errorf("list onboarding tests: %w", err)
	}
	defer rows.Close()

	tests := make([]repositoryTest, 0)
	for rows.Next() {
		var (
			test               repositoryTest
			title, description []byte
		)
		if err := rows.Scan(
			&test.ID,
			&test.Slug,
			&test.Version,
			&title,
			&description,
		); err != nil {
			return nil, fmt.Errorf("scan onboarding test: %w", err)
		}
		if err := json.Unmarshal(title, &test.Title); err != nil {
			return nil, fmt.Errorf("decode onboarding test title: %w", err)
		}
		if err := json.Unmarshal(description, &test.Description); err != nil {
			return nil, fmt.Errorf("decode onboarding test description: %w", err)
		}
		tests = append(tests, test)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate onboarding tests: %w", err)
	}

	for index := range tests {
		questions, err := r.listQuestions(ctx, tests[index].ID)
		if err != nil {
			return nil, err
		}
		tests[index].Questions = questions
	}

	return tests, nil
}

func (r *Repository) GetPublishedTest(
	ctx context.Context,
	slug string,
	version int,
) (repositoryTest, error) {
	var (
		test               repositoryTest
		title, description []byte
	)
	err := r.db.QueryRow(ctx, `
		SELECT id, slug, version, title, description
		FROM onboarding_tests
		WHERE slug = $1 AND version = $2 AND published_at IS NOT NULL
	`, slug, version).Scan(
		&test.ID,
		&test.Slug,
		&test.Version,
		&title,
		&description,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return repositoryTest{}, ErrTestNotFound
	}
	if err != nil {
		return repositoryTest{}, fmt.Errorf("get onboarding test: %w", err)
	}
	if err := json.Unmarshal(title, &test.Title); err != nil {
		return repositoryTest{}, fmt.Errorf("decode onboarding test title: %w", err)
	}
	if err := json.Unmarshal(description, &test.Description); err != nil {
		return repositoryTest{}, fmt.Errorf("decode onboarding test description: %w", err)
	}

	questions, err := r.listQuestions(ctx, test.ID)
	if err != nil {
		return repositoryTest{}, err
	}
	test.Questions = questions

	return test, nil
}

func (r *Repository) listQuestions(
	ctx context.Context,
	testID uuid.UUID,
) ([]Question, error) {
	rows, err := r.db.Query(ctx, `
		SELECT question_key, prompt, answer_type, position,
		       is_required, options, settings
		FROM onboarding_questions
		WHERE test_id = $1
		ORDER BY position
	`, testID)
	if err != nil {
		return nil, fmt.Errorf("list onboarding questions: %w", err)
	}
	defer rows.Close()

	questions := make([]Question, 0)
	for rows.Next() {
		var (
			question                  Question
			prompt, options, settings []byte
		)
		if err := rows.Scan(
			&question.Key,
			&prompt,
			&question.Type,
			&question.Position,
			&question.Required,
			&options,
			&settings,
		); err != nil {
			return nil, fmt.Errorf("scan onboarding question: %w", err)
		}
		if err := json.Unmarshal(prompt, &question.Prompt); err != nil {
			return nil, fmt.Errorf("decode onboarding question prompt: %w", err)
		}
		if err := json.Unmarshal(options, &question.Options); err != nil {
			return nil, fmt.Errorf("decode onboarding question options: %w", err)
		}
		if err := json.Unmarshal(settings, &question.Settings); err != nil {
			return nil, fmt.Errorf("decode onboarding question settings: %w", err)
		}
		questions = append(questions, question)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate onboarding questions: %w", err)
	}

	return questions, nil
}

func (r *Repository) SaveResponse(
	ctx context.Context,
	userID uuid.UUID,
	slug string,
	version int,
	answers json.RawMessage,
) (Response, error) {
	var response Response
	var encodedAnswers []byte

	err := r.db.QueryRow(ctx, `
		INSERT INTO onboarding_responses (
			user_id, test_slug, test_version, answers
		)
		SELECT $1, $2::VARCHAR(80), $3::INTEGER, $4::JSONB
		WHERE EXISTS (
			SELECT 1
			FROM onboarding_tests
			WHERE slug = $2 AND version = $3
			  AND published_at IS NOT NULL
		)
		ON CONFLICT (user_id, test_slug)
		DO UPDATE SET
			test_version = EXCLUDED.test_version,
			answers = EXCLUDED.answers,
			submitted_at = NOW()
		RETURNING test_slug, test_version, answers, submitted_at
	`, userID, slug, version, answers).Scan(
		&response.TestSlug,
		&response.TestVersion,
		&encodedAnswers,
		&response.SubmittedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Response{}, ErrTestNotFound
	}
	if err != nil {
		return Response{}, fmt.Errorf("save onboarding answers: %w", err)
	}
	if err := json.Unmarshal(encodedAnswers, &response.Answers); err != nil {
		return Response{}, fmt.Errorf("decode saved onboarding answers: %w", err)
	}

	return response, nil
}

func (r *Repository) ListResponses(
	ctx context.Context,
	userID uuid.UUID,
) ([]Response, error) {
	rows, err := r.db.Query(ctx, `
		SELECT test_slug, test_version, answers, submitted_at
		FROM onboarding_responses
		WHERE user_id = $1
		ORDER BY submitted_at DESC, test_slug
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("list onboarding responses: %w", err)
	}
	defer rows.Close()

	responses := make([]Response, 0)
	for rows.Next() {
		var (
			response       Response
			encodedAnswers []byte
		)
		if err := rows.Scan(
			&response.TestSlug,
			&response.TestVersion,
			&encodedAnswers,
			&response.SubmittedAt,
		); err != nil {
			return nil, fmt.Errorf("scan onboarding response: %w", err)
		}
		if err := json.Unmarshal(encodedAnswers, &response.Answers); err != nil {
			return nil, fmt.Errorf("decode onboarding response answers: %w", err)
		}
		responses = append(responses, response)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate onboarding responses: %w", err)
	}

	return responses, nil
}
