package ai

import (
	"context"
	"os"
	"testing"

	"github.com/natdanai/quizme/internal/config"
)

// TestRealSmoke calls the real AI provider. It only runs when AI_SMOKE_TEST=true.
// Record tokens_used in plan/PROGRESS.md.
func TestRealSmoke(t *testing.T) {
	if os.Getenv("AI_SMOKE_TEST") != "true" {
		t.Skip("set AI_SMOKE_TEST=true to run a real AI call")
	}
	cfg := config.Load()
	if cfg.AIBaseURL == "" || cfg.AIAPIKey == "" || cfg.AIModel == "" {
		t.Fatal("AI_BASE_URL, AI_API_KEY, AI_MODEL must be set for smoke test")
	}

	c := New(cfg.AIBaseURL, cfg.AIAPIKey, cfg.AIModel)
	qs, tokens, err := c.Generate(context.Background(), 5, []string{"Frontend", "Backend", "Infrastructure"})
	if err != nil {
		t.Fatalf("real AI call failed: %v", err)
	}
	if len(qs) != 5 {
		t.Fatalf("want 5 questions, got %d", len(qs))
	}
	t.Logf("tokens_used=%d", tokens)
	for i, q := range qs {
		t.Logf("q[%d] topic=%q correct=%q", i, q.Topic, q.CorrectOption)
	}
}
