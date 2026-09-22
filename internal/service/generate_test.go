package service

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/natdanai/quizme/internal/ai"
	"github.com/natdanai/quizme/internal/model"
	"github.com/natdanai/quizme/internal/repository"
)

type fakeAI struct {
	mu      sync.Mutex
	calls   int
	results []fakeAIResult
}

type fakeAIResult struct {
	qs     []ai.GeneratedQuestion
	tokens int
	err    error
}

func (f *fakeAI) Generate(ctx context.Context, n int, topics []string) ([]ai.GeneratedQuestion, int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.calls >= len(f.results) {
		return nil, 0, &ai.AIError{Message: "exhausted"}
	}
	r := f.results[f.calls]
	f.calls++
	return r.qs, r.tokens, r.err
}

func (f *fakeAI) callCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls
}

func TestGenerateWithRetryAIErrorMaxThree(t *testing.T) {
	f := &fakeAI{results: []fakeAIResult{
		{err: &ai.AIError{Message: "fail 1"}},
		{err: &ai.AIError{Message: "fail 2"}},
		{err: &ai.AIError{Message: "fail 3"}},
	}}
	svc := &Service{AI: f}

	_, _, err := svc.generateWithRetry(context.Background(), 5, []string{"Frontend"})
	if err == nil {
		t.Fatal("expected error")
	}
	if f.callCount() != 3 {
		t.Fatalf("want 3 AI calls, got %d", f.callCount())
	}
}

func TestGenerateWithRetryValidationRetryOnce(t *testing.T) {
	f := &fakeAI{results: []fakeAIResult{
		{err: &ai.ValidationError{Reasons: []string{"bad"}}},
		{qs: makeQuestions([]string{"Frontend"}, 5), tokens: 123},
	}}
	svc := &Service{AI: f}

	qs, tokens, err := svc.generateWithRetry(context.Background(), 5, []string{"Frontend"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(qs) != 5 {
		t.Fatalf("want 5 questions, got %d", len(qs))
	}
	if tokens != 123 {
		t.Fatalf("want tokens 123, got %d", tokens)
	}
	if f.callCount() != 2 {
		t.Fatalf("want 2 AI calls, got %d", f.callCount())
	}
}

func TestGenerateWithRetryValidationFailTwice(t *testing.T) {
	f := &fakeAI{results: []fakeAIResult{
		{err: &ai.ValidationError{Reasons: []string{"bad"}}},
		{err: &ai.ValidationError{Reasons: []string{"still bad"}}},
	}}
	svc := &Service{AI: f}

	_, _, err := svc.generateWithRetry(context.Background(), 5, []string{"Frontend"})
	if err == nil {
		t.Fatal("expected error")
	}
	if f.callCount() != 2 {
		t.Fatalf("want 2 AI calls, got %d", f.callCount())
	}
}

func TestGenerateIntegrationSuccess(t *testing.T) {
	r := testDB(t)
	ctx := context.Background()

	f := &fakeAI{results: []fakeAIResult{
		{qs: makeQuestions([]string{"Frontend", "Backend", "Infrastructure"}, 5), tokens: 200},
	}}
	svc := newTestService(r, f)

	if err := svc.Generate(ctx); err != nil {
		t.Fatalf("Generate: %v", err)
	}

	batch, questions, err := svc.GetTodayQuiz(ctx)
	if err != nil {
		t.Fatalf("GetTodayQuiz: %v", err)
	}
	if batch.Status != "success" {
		t.Fatalf("want batch status success, got %q", batch.Status)
	}
	if len(questions) != 5 {
		t.Fatalf("want 5 questions, got %d", len(questions))
	}

	logs, err := r.RecentGenerationLogs(ctx, 10)
	if err != nil {
		t.Fatalf("RecentGenerationLogs: %v", err)
	}
	if len(logs) != 1 {
		t.Fatalf("want 1 log row, got %d", len(logs))
	}
	if logs[0].Status != "success" {
		t.Fatalf("want log status success, got %q", logs[0].Status)
	}
	if logs[0].TokensUsed == nil || *logs[0].TokensUsed != 200 {
		t.Fatalf("want tokens_used 200, got %v", logs[0].TokensUsed)
	}
}

func TestGenerateIntegrationIdempotentRerun(t *testing.T) {
	r := testDB(t)
	ctx := context.Background()

	f := &fakeAI{results: []fakeAIResult{
		{qs: makeQuestions([]string{"Frontend", "Backend", "Infrastructure"}, 5), tokens: 100},
	}}
	svc := newTestService(r, f)

	if err := svc.Generate(ctx); err != nil {
		t.Fatalf("Generate first: %v", err)
	}
	if err := svc.Generate(ctx); err != nil {
		t.Fatalf("Generate second: %v", err)
	}

	if f.callCount() != 1 {
		t.Fatalf("want 1 AI call across two Generate runs, got %d", f.callCount())
	}

	logs, err := r.RecentGenerationLogs(ctx, 10)
	if err != nil {
		t.Fatalf("RecentGenerationLogs: %v", err)
	}
	if len(logs) != 1 {
		t.Fatalf("want 1 log row, got %d", len(logs))
	}
}

func TestGenerateIntegrationAIAlwaysFails(t *testing.T) {
	r := testDB(t)
	ctx := context.Background()

	f := &fakeAI{results: []fakeAIResult{
		{err: &ai.AIError{Message: "down"}},
		{err: &ai.AIError{Message: "down"}},
		{err: &ai.AIError{Message: "down"}},
	}}
	svc := newTestService(r, f)

	err := svc.Generate(ctx)
	if err == nil {
		t.Fatal("expected error")
	}
	if f.callCount() != 3 {
		t.Fatalf("want 3 AI calls, got %d", f.callCount())
	}

	batch, err := r.GetBatchByDate(ctx, svc.Now().UTC().Truncate(24*time.Hour))
	if err != nil {
		t.Fatalf("GetBatchByDate: %v", err)
	}
	if batch.Status != "failed" {
		t.Fatalf("want batch status failed, got %q", batch.Status)
	}

	logs, err := r.RecentGenerationLogs(ctx, 10)
	if err != nil {
		t.Fatalf("RecentGenerationLogs: %v", err)
	}
	if len(logs) != 1 {
		t.Fatalf("want 1 log row, got %d", len(logs))
	}
	if logs[0].Status != "failed" {
		t.Fatalf("want log status failed, got %q", logs[0].Status)
	}
	if logs[0].ErrorMessage == nil || *logs[0].ErrorMessage == "" {
		t.Fatal("want failed log error message")
	}
}

func TestGenerateIntegrationValidationRetryOnceThenFail(t *testing.T) {
	r := testDB(t)
	ctx := context.Background()

	f := &fakeAI{results: []fakeAIResult{
		{err: &ai.ValidationError{Reasons: []string{"bad correct_option"}}},
		{err: &ai.ValidationError{Reasons: []string{"still bad"}}},
	}}
	svc := newTestService(r, f)

	err := svc.Generate(ctx)
	if err == nil {
		t.Fatal("expected error")
	}
	if f.callCount() != 2 {
		t.Fatalf("want 2 AI calls, got %d", f.callCount())
	}

	logs, err := r.RecentGenerationLogs(ctx, 10)
	if err != nil {
		t.Fatalf("RecentGenerationLogs: %v", err)
	}
	if len(logs) != 1 {
		t.Fatalf("want 1 log row, got %d", len(logs))
	}
	if logs[0].Status != "failed" {
		t.Fatalf("want log status failed, got %q", logs[0].Status)
	}
}

func newTestService(r *repository.Repo, f *fakeAI) *Service {
	return &Service{
		Repo: r,
		AI:   f,
		Now:  func() time.Time { return time.Date(2026, 9, 22, 0, 0, 0, 0, time.UTC) },
		Rand: rand.New(rand.NewSource(1)),
	}
}

func makeQuestions(topics []string, n int) []ai.GeneratedQuestion {
	qs := make([]ai.GeneratedQuestion, n)
	for i := 0; i < n; i++ {
		qs[i] = ai.GeneratedQuestion{
			Topic:  topics[i%len(topics)],
			Prompt: fmt.Sprintf("Question %d", i+1),
			Options: []model.Option{
				{ID: "a", Text: "Alpha"},
				{ID: "b", Text: "Beta"},
				{ID: "c", Text: "Gamma"},
				{ID: "d", Text: "Delta"},
			},
			CorrectOption: "b",
			Explanation:   "Beta is correct",
		}
	}
	return qs
}
