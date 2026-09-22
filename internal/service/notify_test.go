package service

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/natdanai/quizme/internal/ai"
)

func TestWebhookNotifierSuccessPayload(t *testing.T) {
	var got map[string]any
	var contentType string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		contentType = r.Header.Get("Content-Type")
		body, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(body, &got); err != nil {
			t.Errorf("unmarshal body: %v", err)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	n := NewWebhookNotifier(srv.URL, time.Second)
	err := n.Notify(context.Background(), GenerateResult{
		BatchDate:     time.Date(2026, 9, 22, 0, 0, 0, 0, time.UTC),
		Status:        "success",
		QuestionCount: 5,
		Topics:        []string{"Frontend", "Backend", "Infrastructure"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if contentType != "application/json" {
		t.Fatalf("want Content-Type application/json, got %q", contentType)
	}
	assertPayloadKeys(t, got, []string{"batch_date", "status", "question_count", "topics"})
	if got["batch_date"] != "2026-09-22" {
		t.Fatalf("want batch_date 2026-09-22, got %v", got["batch_date"])
	}
	if got["status"] != "success" {
		t.Fatalf("want status success, got %v", got["status"])
	}
	if got["question_count"] != float64(5) {
		t.Fatalf("want question_count 5, got %v", got["question_count"])
	}
	topics, _ := got["topics"].([]any)
	if len(topics) != 3 || topics[0] != "Frontend" || topics[1] != "Backend" || topics[2] != "Infrastructure" {
		t.Fatalf("want topics [Frontend Backend Infrastructure], got %v", topics)
	}
}

func TestWebhookNotifierFailedPayload(t *testing.T) {
	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(body, &got); err != nil {
			t.Errorf("unmarshal body: %v", err)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	n := NewWebhookNotifier(srv.URL, time.Second)
	err := n.Notify(context.Background(), GenerateResult{
		BatchDate: time.Date(2026, 9, 22, 0, 0, 0, 0, time.UTC),
		Status:    "failed",
		Error:     errors.New("AI API: connection refused"),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertPayloadKeys(t, got, []string{"batch_date", "status", "error"})
	if got["status"] != "failed" {
		t.Fatalf("want status failed, got %v", got["status"])
	}
	if got["error"] != "AI API: connection refused" {
		t.Fatalf("want error 'AI API: connection refused', got %v", got["error"])
	}
	if _, ok := got["question_count"]; ok {
		t.Fatal("question_count should be omitted on failure")
	}
	if _, ok := got["topics"]; ok {
		t.Fatal("topics should be omitted on failure")
	}
}

func TestWebhookNotifierNon2xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	n := NewWebhookNotifier(srv.URL, time.Second)
	err := n.Notify(context.Background(), GenerateResult{
		BatchDate:     time.Now(),
		Status:        "success",
		QuestionCount: 1,
		Topics:        []string{"Frontend"},
	})
	if err == nil {
		t.Fatal("expected error for non-2xx response")
	}
}

func TestWebhookNotifierTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	n := NewWebhookNotifier(srv.URL, 50*time.Millisecond)
	start := time.Now()
	err := n.Notify(context.Background(), GenerateResult{
		BatchDate:     time.Now(),
		Status:        "success",
		QuestionCount: 1,
		Topics:        []string{"Frontend"},
	})
	elapsed := time.Since(start)
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if elapsed > 150*time.Millisecond {
		t.Fatalf("elapsed too long: %v", elapsed)
	}
}

func TestWebhookNotifierUnreachable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	srv.Close()

	n := NewWebhookNotifier(srv.URL, time.Second)
	err := n.Notify(context.Background(), GenerateResult{
		BatchDate:     time.Now(),
		Status:        "success",
		QuestionCount: 1,
		Topics:        []string{"Frontend"},
	})
	if err == nil {
		t.Fatal("expected error for unreachable host")
	}
}

func TestWebhookNotifierEmptyURL(t *testing.T) {
	n := NewWebhookNotifier("", time.Second)
	err := n.Notify(context.Background(), GenerateResult{
		BatchDate:     time.Now(),
		Status:        "success",
		QuestionCount: 1,
		Topics:        []string{"Frontend"},
	})
	if err != nil {
		t.Fatalf("empty URL should be no-op, got error: %v", err)
	}
}

func TestWebhookNotifierContextDetach(t *testing.T) {
	called := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(called)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	n := NewWebhookNotifier(srv.URL, time.Second)
	err := n.Notify(ctx, GenerateResult{
		BatchDate:     time.Now(),
		Status:        "success",
		QuestionCount: 1,
		Topics:        []string{"Frontend"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	select {
	case <-called:
	case <-time.After(2 * time.Second):
		t.Fatal("server was not called with canceled context")
	}
}

func TestWebhookNotifierDefaultTimeout(t *testing.T) {
	n := NewWebhookNotifier("http://example.com", 0)
	if n.Client.Timeout != 5*time.Second {
		t.Fatalf("want default timeout 5s, got %v", n.Client.Timeout)
	}
}

func TestWebhookNotifierSeamIntegration(t *testing.T) {
	r := testDB(t)
	ctx := context.Background()

	payloads := make(chan map[string]any, 2)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var p map[string]any
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &p)
		payloads <- p
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	// Success path.
	f := &fakeAI{results: []fakeAIResult{
		{qs: makeQuestions([]string{"Frontend", "Backend", "Infrastructure"}, 5), tokens: 100},
	}}
	svc := newTestService(r, f)
	svc.Notifier = NewWebhookNotifier(srv.URL, time.Second)
	if err := svc.Generate(ctx); err != nil {
		t.Fatalf("Generate success: %v", err)
	}

	// Reset DB for failure path on a different date.
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

	f2 := &fakeAI{results: []fakeAIResult{
		{err: &ai.AIError{Message: "down"}},
		{err: &ai.AIError{Message: "down"}},
		{err: &ai.AIError{Message: "down"}},
	}}
	svc2 := newTestService(r, f2)
	svc2.Notifier = NewWebhookNotifier(srv.URL, time.Second)
	svc2.Now = func() time.Time { return time.Date(2026, 9, 23, 0, 0, 0, 0, time.UTC) }
	_ = svc2.Generate(ctx) // error expected

	waitPayload := func() map[string]any {
		t.Helper()
		select {
		case p := <-payloads:
			return p
		case <-time.After(2 * time.Second):
			t.Fatal("timed out waiting for webhook payload")
			return nil
		}
	}

	success := waitPayload()
	if success["status"] != "success" {
		t.Fatalf("want success payload, got %v", success)
	}
	assertPayloadKeys(t, success, []string{"batch_date", "status", "question_count", "topics"})

	failed := waitPayload()
	if failed["status"] != "failed" {
		t.Fatalf("want failed payload, got %v", failed)
	}
	assertPayloadKeys(t, failed, []string{"batch_date", "status", "error"})
}

func assertPayloadKeys(t *testing.T, got map[string]any, want []string) {
	t.Helper()
	gotKeys := make([]string, 0, len(got))
	for k := range got {
		gotKeys = append(gotKeys, k)
	}
	sort.Strings(gotKeys)
	sort.Strings(want)
	if !reflect.DeepEqual(gotKeys, want) {
		t.Fatalf("want keys %v, got %v", want, gotKeys)
	}
}
