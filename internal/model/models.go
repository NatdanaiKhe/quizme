package model

import "time"

type Topic struct {
	ID        int       `json:"id" db:"id"`
	Name      string    `json:"name" db:"name"`
	Weight    int       `json:"weight" db:"weight"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

type QuizBatch struct {
	ID        int       `json:"id" db:"id"`
	BatchDate time.Time `json:"batch_date" db:"batch_date"`
	Status    string    `json:"status" db:"status"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

type Question struct {
	ID            int       `json:"id" db:"id"`
	BatchID       int       `json:"batch_id" db:"batch_id"`
	TopicID       int       `json:"topic_id" db:"topic_id"`
	Prompt        string    `json:"prompt" db:"prompt"`
	Options       []Option  `json:"options" db:"options"`
	CorrectOption string    `json:"correct_option" db:"correct_option"`
	Explanation   string    `json:"explanation" db:"explanation"`
	Source        string    `json:"source" db:"source"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
}

type Option struct {
	ID   string `json:"id"`
	Text string `json:"text"`
}

type UserAnswer struct {
	ID             int       `json:"id" db:"id"`
	QuestionID     int       `json:"question_id" db:"question_id"`
	SelectedOption string    `json:"selected_option" db:"selected_option"`
	IsCorrect      bool      `json:"is_correct" db:"is_correct"`
	AnsweredAt     time.Time `json:"answered_at" db:"answered_at"`
}

type UserTopicStats struct {
	TopicID         int       `json:"topic_id" db:"topic_id"`
	CorrectCount    int       `json:"correct_count" db:"correct_count"`
	WrongCount      int       `json:"wrong_count" db:"wrong_count"`
	LastPracticedAt time.Time `json:"last_practiced_at" db:"last_practiced_at"`
	AccuracyRate    float64   `json:"accuracy_rate" db:"accuracy_rate"`
}

type GenerationLog struct {
	ID           int       `json:"id" db:"id"`
	BatchID      int       `json:"batch_id" db:"batch_id"`
	Status       string    `json:"status" db:"status"`
	ErrorMessage string    `json:"error_message" db:"error_message"`
	TokensUsed   int       `json:"tokens_used" db:"tokens_used"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}
