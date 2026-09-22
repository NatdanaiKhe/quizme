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

// GetTodayQuiz serves today's successful batch, or the latest successful batch
// as a fallback. It never creates a pending batch.
func (s *Service) GetTodayQuiz(ctx context.Context) (model.QuizBatch, []model.Question, error) {
	today := s.Now().UTC().Truncate(24 * time.Hour)

	batch, err := s.Repo.GetBatchByDate(ctx, today)
	if err == nil && batch.Status == "success" {
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
