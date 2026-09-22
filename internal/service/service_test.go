package service

import (
	"context"
	"os"
	"testing"

	"github.com/natdanai/quizme/internal/repository"
)

func testDB(t *testing.T) *repository.Repo {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping integration test")
	}

	ctx := context.Background()
	r, err := repository.New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect to test db: %v", err)
	}

	if _, err := r.Pool().Exec(ctx, `
		TRUNCATE TABLE topics, quiz_batches, questions, user_answers, user_topic_stats, generation_logs
		RESTART IDENTITY CASCADE
	`); err != nil {
		t.Fatalf("truncate tables: %v", err)
	}

	if _, err := r.Pool().Exec(ctx, `
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
