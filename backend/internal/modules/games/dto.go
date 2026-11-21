package games

import "time"

type GameSession struct {
	ID             string    `json:"id"`
	CoupleID       string    `json:"couple_id"`
	PlayerID       string    `json:"player_id"`
	OpponentID     string    `json:"opponent_id"`
	Type           string    `json:"type"`
	TotalQuestions int       `json:"total_questions"`
	CorrectAnswers int       `json:"correct_answers"`
	Score          int       `json:"score"`
	Status         string    `json:"status"`
	StartedAt      time.Time `json:"started_at"`
	FinishedAt     *time.Time `json:"finished_at,omitempty"`
}

type Achievement struct {
	ID          string    `json:"id"`
	CoupleID    *string   `json:"couple_id,omitempty"`
	Key         string    `json:"key"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Icon        string    `json:"icon"`
	CreatedAt   time.Time `json:"created_at"`
}

type StartGameRequest struct {
	Type           string `json:"type"`
	TotalQuestions int    `json:"total_questions"`
}

type SubmitAnswerRequest struct {
	SessionID string `json:"session_id"`
	Question  string `json:"question"`
	Answer    string `json:"answer"`
}

type AnswerResult struct {
	Correct      bool  `json:"correct"`
	CurrentScore int   `json:"current_score"`
	TotalAnswered int  `json:"total_answered"`
}
