package games

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNoActiveCouple  = errors.New("no active couple")
	ErrSessionNotFound = errors.New("game session not found")
	ErrInvalidInput    = errors.New("invalid input")
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) GetActiveCoupleAndPartner(ctx context.Context, userID uuid.UUID) (uuid.UUID, uuid.UUID, error) {
	var coupleID, partnerID uuid.UUID
	err := r.db.QueryRow(ctx, `
		SELECT cm.couple_id, m2.user_id
		FROM couple_members cm
		JOIN couple_members m2 ON m2.couple_id = cm.couple_id AND m2.user_id != cm.user_id
		JOIN couples c ON c.id = cm.couple_id
		WHERE cm.user_id = $1
		  AND cm.left_at IS NULL AND c.status = 'active'
		LIMIT 1
	`, userID).Scan(&coupleID, &partnerID)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, uuid.Nil, ErrNoActiveCouple
	}
	return coupleID, partnerID, err
}

func (r *Repository) CreateSession(
	ctx context.Context,
	coupleID, playerID, opponentID uuid.UUID,
	gameType string,
	totalQuestions int,
) (GameSession, error) {
	var s GameSession
	var id uuid.UUID
	err := r.db.QueryRow(ctx, `
		INSERT INTO game_sessions
			(couple_id, player_user_id, opponent_user_id, type, total_questions)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, started_at
	`, coupleID, playerID, opponentID, gameType, totalQuestions).Scan(&id, &s.StartedAt)
	if err != nil {
		return GameSession{}, fmt.Errorf("create session: %w", err)
	}
	s.ID = id.String()
	s.CoupleID = coupleID.String()
	s.PlayerID = playerID.String()
	s.OpponentID = opponentID.String()
	s.Type = gameType
	s.TotalQuestions = totalQuestions
	s.Status = "in_progress"
	return s, nil
}

func (r *Repository) ListSessions(ctx context.Context, userID uuid.UUID) ([]GameSession, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, couple_id, player_user_id, opponent_user_id, type,
		       total_questions, correct_answers, score, status, started_at, finished_at
		FROM game_sessions
		WHERE player_user_id = $1 OR opponent_user_id = $1
		ORDER BY started_at DESC
		LIMIT 50
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("list sessions: %w", err)
	}
	defer rows.Close()

	var sessions []GameSession
	for rows.Next() {
		var s GameSession
		if err := rows.Scan(&s.ID, &s.CoupleID, &s.PlayerID, &s.OpponentID,
			&s.Type, &s.TotalQuestions, &s.CorrectAnswers, &s.Score,
			&s.Status, &s.StartedAt, &s.FinishedAt); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		sessions = append(sessions, s)
	}
	return sessions, rows.Err()
}

func (r *Repository) SubmitAnswer(
	ctx context.Context,
	sessionID, playerID uuid.UUID,
	questionID, answerValue string,
) (AnswerResult, error) {
	var correct bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM onboarding_responses r
			WHERE r.user_id = (SELECT opponent_user_id FROM game_sessions WHERE id = $1)
			  AND r.question_id = $2::UUID
			  AND r.answer_value = $3
		)
	`, sessionID, questionID, answerValue).Scan(&correct)
	if err != nil {
		return AnswerResult{}, fmt.Errorf("check answer: %w", err)
	}

	correctCount := 0
	score := 0
	if correct {
		correctCount = 1
		score = 10
	}

	_, err = r.db.Exec(ctx, `
		UPDATE game_sessions
		SET correct_answers = correct_answers + $2,
		    score = score + $3
		WHERE id = $1 AND status = 'in_progress'
	`, sessionID, correctCount, score)
	if err != nil {
		return AnswerResult{}, fmt.Errorf("update session: %w", err)
	}

	return AnswerResult{Correct: correct, CurrentScore: score, TotalAnswered: 1}, nil
}

func (r *Repository) FinishSession(ctx context.Context, sessionID, userID uuid.UUID) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE game_sessions
		SET status = 'finished', finished_at = NOW()
		WHERE id = $1 AND (player_user_id = $2 OR opponent_user_id = $2)
		  AND status = 'in_progress'
	`, sessionID, userID)
	if err != nil {
		return fmt.Errorf("finish: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrSessionNotFound
	}
	return nil
}
