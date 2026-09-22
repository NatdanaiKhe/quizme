package service

import (
	"context"
	"time"

	"github.com/natdanai/quizme/internal/model"
)

// failingStatusRepo delegates to an underlying Repository but forces
// UpdateBatchStatus("success") to fail. It is used to test the failure
// handling path inside succeedBatch.
type failingStatusRepo struct {
	Repository
	err error
}

func (f failingStatusRepo) UpdateBatchStatus(ctx context.Context, id int, status string) error {
	if status == "success" {
		return f.err
	}
	return f.Repository.UpdateBatchStatus(ctx, id, status)
}

// errBatchRepo forces GetBatchByDate to return a fixed error. It is used to
// verify that GetTodayQuiz surfaces real database errors instead of silently
// falling back to stale data.
type errBatchRepo struct {
	Repository
	err error
}

func (e errBatchRepo) GetBatchByDate(ctx context.Context, batchDate time.Time) (model.QuizBatch, error) {
	return model.QuizBatch{}, e.err
}
