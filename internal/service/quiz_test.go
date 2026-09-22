package service

import (
	"context"
	"errors"
	"math/rand"
	"testing"
	"time"

	"github.com/natdanai/quizme/internal/ai"
	"github.com/natdanai/quizme/internal/model"
)

func TestGetTodayQuizFallback(t *testing.T) {
	r := testDB(t)
	ctx := context.Background()

	// Seed a successful batch from yesterday.
	yesterday := time.Date(2026, 9, 21, 0, 0, 0, 0, time.UTC)
	oldBatch, _, err := r.GetOrCreateBatch(ctx, yesterday)
	if err != nil {
		t.Fatalf("create yesterday batch: %v", err)
	}
	if err := r.UpdateBatchStatus(ctx, oldBatch.ID, "success"); err != nil {
		t.Fatalf("update yesterday batch status: %v", err)
	}
	oldQuestions := []model.Question{
		{
			TopicID:       1,
			Prompt:        "Old question",
			Options:       []model.Option{{ID: "a", Text: "A"}, {ID: "b", Text: "B"}, {ID: "c", Text: "C"}, {ID: "d", Text: "D"}},
			CorrectOption: "a",
			Explanation:   strPtr("old"),
			Source:        "ai_generated",
		},
	}
	if err := r.InsertQuestions(ctx, oldBatch.ID, oldQuestions); err != nil {
		t.Fatalf("insert old questions: %v", err)
	}

	// Today's generation fails.
	f := &fakeAI{results: []fakeAIResult{{err: &ai.AIError{Message: "down"}}}}
	svc := newTestService(r, f)
	_ = svc.Generate(ctx)

	batch, questions, err := svc.GetTodayQuiz(ctx)
	if err != nil {
		t.Fatalf("GetTodayQuiz: %v", err)
	}
	if batch.Status != "success" {
		t.Fatalf("want fallback batch status success, got %q", batch.Status)
	}
	if !batch.BatchDate.Equal(yesterday) {
		t.Fatalf("want fallback batch date %s, got %s", yesterday.Format(time.DateOnly), batch.BatchDate.Format(time.DateOnly))
	}
	if len(questions) != 1 {
		t.Fatalf("want 1 fallback question, got %d", len(questions))
	}
}

func TestGetTodayQuizDBErrorNotFallback(t *testing.T) {
	r := testDB(t)
	ctx := context.Background()

	yesterday := time.Date(2026, 9, 21, 0, 0, 0, 0, time.UTC)
	oldBatch, _, err := r.GetOrCreateBatch(ctx, yesterday)
	if err != nil {
		t.Fatalf("create yesterday batch: %v", err)
	}
	if err := r.UpdateBatchStatus(ctx, oldBatch.ID, "success"); err != nil {
		t.Fatalf("update yesterday batch status: %v", err)
	}
	if err := r.InsertQuestions(ctx, oldBatch.ID, []model.Question{
		{
			TopicID:       1,
			Prompt:        "Old question",
			Options:       []model.Option{{ID: "a", Text: "A"}, {ID: "b", Text: "B"}, {ID: "c", Text: "C"}, {ID: "d", Text: "D"}},
			CorrectOption: "a",
			Explanation:   strPtr("old"),
			Source:        "ai_generated",
		},
	}); err != nil {
		t.Fatalf("insert old questions: %v", err)
	}

	svc := newTestService(r, &fakeAI{})
	svc.Repo = errBatchRepo{Repository: r, err: errors.New("db unavailable")}

	_, _, err = svc.GetTodayQuiz(ctx)
	if err == nil {
		t.Fatal("expected error")
	}
	if errors.Is(err, ErrNoQuiz) {
		t.Fatal("expected real DB error, not ErrNoQuiz fallback")
	}
}

func TestGetTodayQuizErrNoQuiz(t *testing.T) {
	r := testDB(t)
	ctx := context.Background()

	svc := &Service{
		Repo: r,
		AI:   &fakeAI{},
		Now:  func() time.Time { return time.Date(2026, 9, 22, 0, 0, 0, 0, time.UTC) },
		Rand: rand.New(rand.NewSource(1)),
	}

	_, _, err := svc.GetTodayQuiz(ctx)
	if !errors.Is(err, ErrNoQuiz) {
		t.Fatalf("want ErrNoQuiz, got %v", err)
	}
}

func strPtr(s string) *string { return &s }
