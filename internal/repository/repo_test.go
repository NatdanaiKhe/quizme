package repository

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/natdanai/quizme/internal/model"
)

func testDB(t *testing.T) *Repo {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping integration test")
	}

	ctx := context.Background()
	r, err := New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect to test db: %v", err)
	}

	if _, err := r.pool.Exec(ctx, `
		TRUNCATE TABLE topics, quiz_batches, questions, user_answers, user_topic_stats, generation_logs
		RESTART IDENTITY CASCADE
	`); err != nil {
		t.Fatalf("truncate tables: %v", err)
	}

	if _, err := r.pool.Exec(ctx, `
		INSERT INTO topics (name, weight) VALUES
			('Frontend', 1),
			('Backend', 1),
			('Infrastructure', 1)
	`); err != nil {
		t.Fatalf("seed topics: %v", err)
	}

	t.Cleanup(func() {
		r.Close()
	})
	return r
}

func TestListTopics(t *testing.T) {
	r := testDB(t)
	ctx := context.Background()

	topics, err := r.ListTopics(ctx)
	if err != nil {
		t.Fatalf("ListTopics: %v", err)
	}
	if len(topics) != 3 {
		t.Fatalf("want 3 seeded topics, got %d", len(topics))
	}
}

func TestCreateTopic(t *testing.T) {
	r := testDB(t)
	ctx := context.Background()

	created, err := r.CreateTopic(ctx, "Security", 2)
	if err != nil {
		t.Fatalf("CreateTopic: %v", err)
	}
	if created.Name != "Security" || created.Weight != 2 {
		t.Fatalf("unexpected created topic: %+v", created)
	}

	topics, err := r.ListTopics(ctx)
	if err != nil {
		t.Fatalf("ListTopics: %v", err)
	}
	if len(topics) != 4 {
		t.Fatalf("want 4 topics after create, got %d", len(topics))
	}
}

func TestGetOrCreateBatch(t *testing.T) {
	r := testDB(t)
	ctx := context.Background()
	date := time.Date(2026, 9, 22, 0, 0, 0, 0, time.UTC)

	first, created, err := r.GetOrCreateBatch(ctx, date)
	if err != nil {
		t.Fatalf("GetOrCreateBatch first: %v", err)
	}
	if !created {
		t.Fatalf("first call should create a batch")
	}

	second, created, err := r.GetOrCreateBatch(ctx, date)
	if err != nil {
		t.Fatalf("GetOrCreateBatch second: %v", err)
	}
	if created {
		t.Fatalf("second call should not create a batch")
	}
	if first.ID != second.ID {
		t.Fatalf("expected same batch, got %d and %d", first.ID, second.ID)
	}
}

func TestBatchQuestionRoundTrip(t *testing.T) {
	r := testDB(t)
	ctx := context.Background()

	topics, _ := r.ListTopics(ctx)
	if len(topics) < 2 {
		t.Fatal("need at least 2 topics")
	}

	date := time.Date(2026, 9, 22, 0, 0, 0, 0, time.UTC)
	batch, _, err := r.GetOrCreateBatch(ctx, date)
	if err != nil {
		t.Fatalf("create batch: %v", err)
	}

	want := []model.Question{
		{
			TopicID:       topics[0].ID,
			Prompt:        "What is 2+2?",
			Options:       []model.Option{{ID: "a", Text: "3"}, {ID: "b", Text: "4"}},
			CorrectOption: "b",
			Explanation:   strPtr("Basic arithmetic."),
			Source:        "test",
		},
		{
			TopicID:       topics[1].ID,
			Prompt:        "What is the capital of France?",
			Options:       []model.Option{{ID: "a", Text: "Berlin"}, {ID: "b", Text: "Paris"}, {ID: "c", Text: "Madrid"}},
			CorrectOption: "b",
			Explanation:   nil,
			Source:        "manual",
		},
	}

	if err := r.InsertQuestions(ctx, batch.ID, want); err != nil {
		t.Fatalf("InsertQuestions: %v", err)
	}

	got, err := r.GetQuestionsByBatch(ctx, batch.ID)
	if err != nil {
		t.Fatalf("GetQuestionsByBatch: %v", err)
	}
	if len(got) != len(want) {
		t.Fatalf("want %d questions, got %d", len(want), len(got))
	}

	byPrompt := make(map[string]model.Question)
	for _, q := range got {
		byPrompt[q.Prompt] = q
	}

	for _, w := range want {
		g, ok := byPrompt[w.Prompt]
		if !ok {
			t.Fatalf("missing question: %q", w.Prompt)
		}
		if g.BatchID != batch.ID {
			t.Fatalf("batch_id mismatch for %q: got %d, want %d", w.Prompt, g.BatchID, batch.ID)
		}
		if g.TopicID != w.TopicID {
			t.Fatalf("topic_id mismatch for %q: got %d, want %d", w.Prompt, g.TopicID, w.TopicID)
		}
		if g.CorrectOption != w.CorrectOption {
			t.Fatalf("correct_option mismatch for %q: got %q", w.Prompt, g.CorrectOption)
		}
		if g.Source != w.Source {
			t.Fatalf("source mismatch for %q: got %q, want %q", w.Prompt, g.Source, w.Source)
		}
		if len(g.Options) != len(w.Options) {
			t.Fatalf("options count mismatch for %q: got %d", w.Prompt, len(g.Options))
		}
		for i, opt := range w.Options {
			if g.Options[i].ID != opt.ID || g.Options[i].Text != opt.Text {
				t.Fatalf("option mismatch for %q at %d: got %+v, want %+v", w.Prompt, i, g.Options[i], opt)
			}
		}
		if (g.Explanation == nil) != (w.Explanation == nil) {
			t.Fatalf("explanation nil mismatch for %q: got %v", w.Prompt, g.Explanation)
		}
		if g.Explanation != nil && w.Explanation != nil && *g.Explanation != *w.Explanation {
			t.Fatalf("explanation mismatch for %q: got %q", w.Prompt, *g.Explanation)
		}
	}
}

func TestSubmitAnswer(t *testing.T) {
	r := testDB(t)
	ctx := context.Background()

	topics, _ := r.ListTopics(ctx)
	topic := topics[0]
	batch, _, _ := r.GetOrCreateBatch(ctx, time.Date(2026, 9, 22, 0, 0, 0, 0, time.UTC))
	q := model.Question{
		TopicID:       topic.ID,
		Prompt:        "Q1",
		Options:       []model.Option{{ID: "a", Text: "yes"}, {ID: "b", Text: "no"}},
		CorrectOption: "a",
		Source:        "test",
	}
	if err := r.InsertQuestions(ctx, batch.ID, []model.Question{q}); err != nil {
		t.Fatalf("InsertQuestions: %v", err)
	}

	questions, _ := r.GetQuestionsByBatch(ctx, batch.ID)
	qid := questions[0].ID

	if err := r.SubmitAnswer(ctx, qid, "a", true); err != nil {
		t.Fatalf("SubmitAnswer correct: %v", err)
	}

	stats, err := r.GetStats(ctx)
	if err != nil {
		t.Fatalf("GetStats: %v", err)
	}
	if len(stats) != 1 {
		t.Fatalf("want 1 stats row, got %d", len(stats))
	}
	s := stats[0]
	if s.TopicID != topic.ID || s.CorrectCount != 1 || s.WrongCount != 0 {
		t.Fatalf("unexpected stats after correct: %+v", s)
	}
	if s.AccuracyRate != 100 {
		t.Fatalf("want accuracy 100, got %v", s.AccuracyRate)
	}
	if s.LastPracticedAt == nil {
		t.Fatal("expected LastPracticedAt to be set")
	}

	if err := r.SubmitAnswer(ctx, qid, "b", false); err != nil {
		t.Fatalf("SubmitAnswer wrong: %v", err)
	}

	stats, _ = r.GetStats(ctx)
	s = stats[0]
	if s.CorrectCount != 1 || s.WrongCount != 1 || s.AccuracyRate != 50 {
		t.Fatalf("unexpected stats after wrong: %+v", s)
	}
}

func TestSubmitAnswerRollback(t *testing.T) {
	r := testDB(t)
	ctx := context.Background()

	err := r.SubmitAnswer(ctx, 99999, "a", true)
	if err == nil {
		t.Fatal("expected error for missing question")
	}

	var count int
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM user_answers`).Scan(&count); err != nil {
		t.Fatalf("count user_answers: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected 0 answers after rollback, got %d", count)
	}

	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM user_topic_stats`).Scan(&count); err != nil {
		t.Fatalf("count user_topic_stats: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected 0 stats after rollback, got %d", count)
	}
}

func TestLatestSuccessfulBatch(t *testing.T) {
	r := testDB(t)
	ctx := context.Background()

	if _, err := r.LatestSuccessfulBatch(ctx); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("expected ErrNoRows with no successful batches, got %v", err)
	}

	batch, _, _ := r.GetOrCreateBatch(ctx, time.Date(2026, 9, 22, 0, 0, 0, 0, time.UTC))
	if err := r.UpdateBatchStatus(ctx, batch.ID, "success"); err != nil {
		t.Fatalf("UpdateBatchStatus: %v", err)
	}

	latest, err := r.LatestSuccessfulBatch(ctx)
	if err != nil {
		t.Fatalf("LatestSuccessfulBatch: %v", err)
	}
	if latest.ID != batch.ID {
		t.Fatalf("want batch %d, got %d", batch.ID, latest.ID)
	}
}

func TestGenerationLogs(t *testing.T) {
	r := testDB(t)
	ctx := context.Background()

	inserted, err := r.InsertGenerationLog(ctx, model.GenerationLog{
		Status:       "failed",
		ErrorMessage: strPtr("timeout"),
		TokensUsed:   intPtr(42),
	})
	if err != nil {
		t.Fatalf("InsertGenerationLog: %v", err)
	}
	if inserted.Status != "failed" {
		t.Fatalf("unexpected status: %s", inserted.Status)
	}

	logs, err := r.RecentGenerationLogs(ctx, 5)
	if err != nil {
		t.Fatalf("RecentGenerationLogs: %v", err)
	}
	if len(logs) != 1 {
		t.Fatalf("want 1 log, got %d", len(logs))
	}
	if logs[0].ErrorMessage == nil || *logs[0].ErrorMessage != "timeout" {
		t.Fatalf("error message mismatch: %v", logs[0].ErrorMessage)
	}
}

func strPtr(s string) *string { return &s }
func intPtr(i int) *int       { return &i }
