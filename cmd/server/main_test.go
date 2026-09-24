package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/natdanai/quizme/internal/config"
)

func TestMonolithRouter(t *testing.T) {
	cfg := config.Config{
		Port:          "8080",
		InternalToken: "test-token",
	}

	router := setupRouter(cfg, nil)

	tests := []struct {
		name         string
		method       string
		path         string
		wantStatus   int
		wantContains string
	}{
		{
			name:         "Health check endpoint",
			method:       http.MethodGet,
			path:         "/health",
			wantStatus:   http.StatusOK,
			wantContains: `{"status":"ok"}`,
		},
		{
			name:         "Root serves embedded index.html",
			method:       http.MethodGet,
			path:         "/",
			wantStatus:   http.StatusOK,
			wantContains: "<!doctype html>",
		},
		{
			name:         "SPA route fallback serves index.html",
			method:       http.MethodGet,
			path:         "/quiz",
			wantStatus:   http.StatusOK,
			wantContains: "<!doctype html>",
		},
		{
			name:         "Static asset favicon.svg served",
			method:       http.MethodGet,
			path:         "/favicon.svg",
			wantStatus:   http.StatusOK,
			wantContains: "<svg",
		},
		{
			name:         "Unknown POST route returns 404",
			method:       http.MethodPost,
			path:         "/unknown-api",
			wantStatus:   http.StatusNotFound,
			wantContains: `{"error":"not found"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(tt.method, tt.path, nil)
			router.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body = %s", w.Code, tt.wantStatus, w.Body.String())
			}
			if tt.wantContains != "" && !strings.Contains(w.Body.String(), tt.wantContains) {
				t.Errorf("body does not contain %q; got: %s", tt.wantContains, w.Body.String())
			}
		})
	}
}
