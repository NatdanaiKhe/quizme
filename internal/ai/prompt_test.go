package ai

import (
	"strings"
	"testing"
)

func TestBuildPrompt(t *testing.T) {
	got := buildPrompt(5, []string{"Frontend", "Backend"})
	if !strings.Contains(got, "Generate 5 multiple-choice questions") {
		t.Fatalf("prompt did not substitute n: %q", got)
	}
	if !strings.Contains(got, "Frontend, Backend") {
		t.Fatalf("prompt did not substitute topics: %q", got)
	}
}
