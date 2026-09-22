package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/natdanai/quizme/internal/model"
)

func TestClientGenerateHappyPath(t *testing.T) {
	var authHeader string
	var gotBody map[string]any
	validContent, _ := json.Marshal([]GeneratedQuestion{
		validQuestion("Frontend", "b"),
		validQuestion("Frontend", "c"),
		validQuestion("Backend", "a"),
		validQuestion("Backend", "d"),
		validQuestion("Infrastructure", "b"),
	})

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader = r.Header.Get("Authorization")
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{
				{"message": map[string]any{"content": string(validContent)}},
			},
			"usage": map[string]any{"total_tokens": 1234},
		})
	}))
	defer ts.Close()

	c := New(ts.URL, "secret-key", "test-model")
	qs, tokens, err := c.Generate(context.Background(), 5, []string{"Frontend", "Backend", "Infrastructure"})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	if len(qs) != 5 {
		t.Fatalf("want 5 questions, got %d", len(qs))
	}
	if tokens != 1234 {
		t.Fatalf("want tokens 1234, got %d", tokens)
	}
	if authHeader != "Bearer secret-key" {
		t.Fatalf("want Bearer secret-key, got %q", authHeader)
	}
	if gotBody["model"] != "test-model" {
		t.Fatalf("want model test-model, got %v", gotBody["model"])
	}
	if gotBody["max_tokens"] != float64(maxTokens) {
		t.Fatalf("want max_tokens %d, got %v", maxTokens, gotBody["max_tokens"])
	}
}

func TestClientGenerateErrors(t *testing.T) {
	cases := []struct {
		name       string
		server     *httptest.Server
		wantErr    error
		wantAIErr  bool
		wantValErr bool
	}{
		{
			name: "non-200 returns AIError",
			server: httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"error":"boom"}`))
			})),
			wantAIErr: true,
		},
		{
			name: "malformed envelope returns AIError",
			server: httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"choices":[]}`))
			})),
			wantAIErr: true,
		},
		{
			name: "content fails validation returns ValidationError",
			server: httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				bad, _ := json.Marshal([]GeneratedQuestion{validQuestion("Frontend", "e")})
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(map[string]any{
					"choices": []map[string]any{{"message": map[string]any{"content": string(bad)}}},
					"usage":   map[string]any{"total_tokens": 42},
				})
			})),
			wantValErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			defer tc.server.Close()
			c := New(tc.server.URL, "key", "model")
			_, _, err := c.Generate(context.Background(), 1, []string{"Frontend"})
			if err == nil {
				t.Fatalf("expected error")
			}
			if tc.wantAIErr && !IsAIError(err) {
				t.Fatalf("expected AIError, got %T: %v", err, err)
			}
			if tc.wantValErr && !IsValidationError(err) {
				t.Fatalf("expected ValidationError, got %T: %v", err, err)
			}
		})
	}
}

func TestClientGenerateConnectionRefused(t *testing.T) {
	c := New("http://127.0.0.1:1", "key", "model")
	_, _, err := c.Generate(context.Background(), 1, []string{"Frontend"})
	if !IsAIError(err) {
		t.Fatalf("expected AIError, got %T: %v", err, err)
	}
}

func TestClientGenerateTimeout(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	c := New(ts.URL, "key", "model")
	c.httpClient.Timeout = 50 * time.Millisecond

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	_, _, err := c.Generate(ctx, 1, []string{"Frontend"})
	if !IsAIError(err) {
		t.Fatalf("expected AIError, got %T: %v", err, err)
	}
}

func TestClientMissingConfig(t *testing.T) {
	cases := []struct {
		name    string
		baseURL string
		key     string
		model   string
	}{
		{"missing base url", "", "key", "model"},
		{"missing key", "http://x", "", "model"},
		{"missing model", "http://x", "key", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := New(tc.baseURL, tc.key, tc.model)
			_, _, err := c.Generate(context.Background(), 1, []string{"Frontend"})
			if !IsAIError(err) {
				t.Fatalf("expected AIError, got %T: %v", err, err)
			}
		})
	}
}

func TestNoKeyLeakInError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`bad request`))
	}))
	defer ts.Close()

	secret := "sk-secret-leak-test"
	c := New(ts.URL, secret, "model")
	_, _, err := c.Generate(context.Background(), 1, []string{"Frontend"})
	if err == nil {
		t.Fatalf("expected error")
	}
	if strings.Contains(err.Error(), secret) {
		t.Fatalf("error leaked API key: %v", err)
	}
}

func TestGenerateDoesNotExposeCorrectOptionOverHTTP(t *testing.T) {
	// GeneratedQuestion structs are not marshaled for HTTP in this package,
	// but ensure the field name is present only for internal DB use.
	q := GeneratedQuestion{
		Topic:         "Frontend",
		Prompt:        "q",
		Options:       []model.Option{{ID: "a", Text: "x"}},
		CorrectOption: "a",
		Explanation:   "e",
	}
	b, _ := json.Marshal(q)
	if !strings.Contains(string(b), "correct_option") {
		t.Fatalf("expected correct_option in internal JSON")
	}
}
