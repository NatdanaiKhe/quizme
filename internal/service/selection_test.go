package service

import (
	"math/rand"
	"testing"
	"time"

	"github.com/natdanai/quizme/internal/model"
)

func TestSelectionDeterministicSeeded(t *testing.T) {
	topics := []model.Topic{
		{ID: 1, Name: "Frontend"},
		{ID: 2, Name: "Backend"},
		{ID: 3, Name: "Infrastructure"},
	}
	stats := []model.UserTopicStats{
		{TopicID: 1, AccuracyRate: 80, LastPracticedAt: ptrTime(time.Now())},
		{TopicID: 2, AccuracyRate: 50, LastPracticedAt: ptrTime(time.Now())},
		{TopicID: 3, AccuracyRate: 90, LastPracticedAt: ptrTime(time.Now())},
	}
	rnd := rand.New(rand.NewSource(42))

	first := SelectTopics(5, topics, stats, rnd)
	rnd = rand.New(rand.NewSource(42))
	second := SelectTopics(5, topics, stats, rnd)

	if len(first) != 5 || len(second) != 5 {
		t.Fatalf("want 5 topics, got %d and %d", len(first), len(second))
	}
	for i := range first {
		if first[i] != second[i] {
			t.Fatalf("seeded selection not deterministic at %d: %q vs %q", i, first[i], second[i])
		}
	}
}

func TestSelectionUnattemptedFirst(t *testing.T) {
	topics := []model.Topic{
		{ID: 1, Name: "Frontend"},
		{ID: 2, Name: "Backend"},
		{ID: 3, Name: "Infrastructure"},
	}
	// Frontend is unattempted (missing stats row); Backend has 0% accuracy;
	// Infrastructure is strongest. Unattempted must be in the weak group.
	stats := []model.UserTopicStats{
		{TopicID: 2, AccuracyRate: 0, LastPracticedAt: ptrTime(time.Now())},
		{TopicID: 3, AccuracyRate: 100, LastPracticedAt: ptrTime(time.Now())},
	}

	for seed := int64(0); seed < 200; seed++ {
		rnd := rand.New(rand.NewSource(seed))
		got := SelectTopics(5, topics, stats, rnd)
		if len(got) != 5 {
			t.Fatalf("seed %d: want 5 topics, got %d", seed, len(got))
		}
		foundFrontend := false
		for _, name := range got[:3] {
			if name == "Frontend" {
				foundFrontend = true
				break
			}
		}
		if !foundFrontend {
			t.Fatalf("seed %d: unattempted Frontend missing from weak group %v", seed, got[:3])
		}
	}
}

func TestSelectionSplitBounds(t *testing.T) {
	topics := []model.Topic{
		{ID: 1, Name: "A"},
		{ID: 2, Name: "B"},
		{ID: 3, Name: "C"},
		{ID: 4, Name: "D"},
		{ID: 5, Name: "E"},
	}
	stats := []model.UserTopicStats{}

	for seed := int64(0); seed < 200; seed++ {
		rnd := rand.New(rand.NewSource(seed))
		got := SelectTopics(5, topics, stats, rnd)
		if len(got) != 5 {
			t.Fatalf("seed %d: want 5 topics, got %d", seed, len(got))
		}
		if got[0] == "" || got[1] == "" || got[2] == "" {
			t.Fatalf("seed %d: weak group contains empty topic", seed)
		}

		seen := make(map[string]struct{})
		for _, name := range got {
			seen[name] = struct{}{}
		}
		if len(seen) != 5 {
			t.Fatalf("seed %d: want 5 distinct topics, got %v", seed, got)
		}
	}
}

func TestSelectionSingleTopicDegradation(t *testing.T) {
	topics := []model.Topic{{ID: 1, Name: "Only"}}
	stats := []model.UserTopicStats{}
	rnd := rand.New(rand.NewSource(1))

	got := SelectTopics(5, topics, stats, rnd)
	if len(got) != 5 {
		t.Fatalf("want 5 topics, got %d", len(got))
	}
	for _, name := range got {
		if name != "Only" {
			t.Fatalf("single-topic degradation should return only topic, got %q", name)
		}
	}
}

func ptrTime(v time.Time) *time.Time { return &v }
