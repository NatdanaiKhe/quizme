package service

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/natdanai/quizme/internal/model"
)

// ErrNoQuiz is returned when no successful batch exists to serve.
var ErrNoQuiz = errors.New("no quiz available")

// ErrNotFound is returned when a referenced question does not exist.
var ErrNotFound = errors.New("not found")

// GetTodayQuiz serves today's successful batch, or the latest successful batch
// as a fallback. It never creates a pending batch.
// Answer compares the selected option against the stored correct option and
// records the answer transactionally. It returns whether the answer was correct
// and the question's explanation (which may be nil).
func (s *Service) Answer(ctx context.Context, questionID int, selectedOption string) (bool, *string, error) {
	q, err := s.Repo.GetQuestion(ctx, questionID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil, ErrNotFound
		}
		return false, nil, err
	}

	correct := selectedOption == q.CorrectOption
	if err := s.Repo.SubmitAnswer(ctx, questionID, selectedOption, correct); err != nil {
		return false, nil, err
	}
	return correct, q.Explanation, nil
}

// ListTopics returns every topic.
func (s *Service) ListTopics(ctx context.Context) ([]model.Topic, error) {
	return s.Repo.ListTopics(ctx)
}

// CreateTopic adds a new topic.
func (s *Service) CreateTopic(ctx context.Context, name string, weight int) (model.Topic, error) {
	return s.Repo.CreateTopic(ctx, name, weight)
}

// GetStats returns per-topic answer stats.
func (s *Service) GetStats(ctx context.Context) ([]model.UserTopicStats, error) {
	return s.Repo.GetStats(ctx)
}

func (s *Service) GetTodayQuiz(ctx context.Context) (model.QuizBatch, []model.Question, error) {
	today := s.Now().UTC().Truncate(24 * time.Hour)

	batch, err := s.Repo.GetBatchByDate(ctx, today)
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return model.QuizBatch{}, nil, err
		}
	} else if batch.Status == "success" {
		questions, err := s.Repo.GetQuestionsByBatch(ctx, batch.ID)
		if err != nil {
			return model.QuizBatch{}, nil, err
		}
		return batch, questions, nil
	}

	batch, err = s.Repo.LatestSuccessfulBatch(ctx)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.QuizBatch{}, nil, ErrNoQuiz
		}
		return model.QuizBatch{}, nil, err
	}

	questions, err := s.Repo.GetQuestionsByBatch(ctx, batch.ID)
	if err != nil {
		return model.QuizBatch{}, nil, err
	}
	return batch, questions, nil
}
