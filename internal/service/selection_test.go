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

func TestSelectionDistributionBand(t *testing.T) {
	// Five topics; ranked weakest→strongest: D, E (unattempted), C (50%), B (80%), A (100%).
	topics := []model.Topic{
		{ID: 1, Name: "A"},
		{ID: 2, Name: "B"},
		{ID: 3, Name: "C"},
		{ID: 4, Name: "D"},
		{ID: 5, Name: "E"},
	}
	stats := []model.UserTopicStats{
		{TopicID: 1, AccuracyRate: 100, LastPracticedAt: ptrTime(time.Now())},
		{TopicID: 2, AccuracyRate: 80, LastPracticedAt: ptrTime(time.Now())},
		{TopicID: 3, AccuracyRate: 50, LastPracticedAt: ptrTime(time.Now())},
	}
	weakest := map[string]struct{}{"C": {}, "D": {}, "E": {}}
	rest := map[string]struct{}{"A": {}, "B": {}}

	for seed := int64(0); seed < 200; seed++ {
		rnd := rand.New(rand.NewSource(seed))
		got := SelectTopics(5, topics, stats, rnd)
		if len(got) != 5 {
			t.Fatalf("seed %d: want 5 topics, got %d", seed, len(got))
		}
		for _, name := range got[:3] {
			if _, ok := weakest[name]; !ok {
				t.Fatalf("seed %d: weak slot %q not in weakest set %v", seed, name, got)
			}
		}
		for _, name := range got[3:] {
			if _, ok := rest[name]; !ok {
				t.Fatalf("seed %d: rest slot %q not in rest set %v", seed, name, got)
			}
		}
	}
}

func TestSelectionNeverAllOneTopic(t *testing.T) {
	topics := []model.Topic{
		{ID: 1, Name: "A"},
		{ID: 2, Name: "B"},
	}
	stats := []model.UserTopicStats{}

	for seed := int64(0); seed < 200; seed++ {
		rnd := rand.New(rand.NewSource(seed))
		got := SelectTopics(5, topics, stats, rnd)
		seen := make(map[string]struct{})
		for _, name := range got {
			seen[name] = struct{}{}
		}
		if len(seen) < 2 {
			t.Fatalf("seed %d: expected at least 2 distinct topics, got %v", seed, got)
		}
	}
}

func TestSelectionMoreTopicsThanN(t *testing.T) {
	topics := make([]model.Topic, 8)
	for i := range topics {
		topics[i] = model.Topic{ID: i + 1, Name: string(rune('A' + i))}
	}
	stats := []model.UserTopicStats{}

	for seed := int64(0); seed < 200; seed++ {
		rnd := rand.New(rand.NewSource(seed))
		got := SelectTopics(5, topics, stats, rnd)
		if len(got) != 5 {
			t.Fatalf("seed %d: want 5 topics, got %d", seed, len(got))
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

func TestSelectionEqualAccuracyTieBreak(t *testing.T) {
	topics := []model.Topic{
		{ID: 1, Name: "A"},
		{ID: 2, Name: "B"},
		{ID: 3, Name: "C"},
		{ID: 4, Name: "D"},
		{ID: 5, Name: "E"},
	}
	now := time.Now()
	stats := []model.UserTopicStats{
		{TopicID: 1, AccuracyRate: 50, LastPracticedAt: ptrTime(now)},
		{TopicID: 2, AccuracyRate: 50, LastPracticedAt: ptrTime(now)},
		{TopicID: 3, AccuracyRate: 50, LastPracticedAt: ptrTime(now)},
		{TopicID: 4, AccuracyRate: 50, LastPracticedAt: ptrTime(now)},
		{TopicID: 5, AccuracyRate: 50, LastPracticedAt: ptrTime(now)},
	}

	// With identical accuracy, ranking falls back to ID order. Weak slots must be
	// deterministic; rest slots are shuffled from the remaining topics.
	for seed := int64(0); seed < 50; seed++ {
		rnd := rand.New(rand.NewSource(seed))
		got := SelectTopics(5, topics, stats, rnd)
		wantWeak := []string{"A", "B", "C"}
		for i, name := range got[:3] {
			if name != wantWeak[i] {
				t.Fatalf("seed %d: want weak slot %d = %q, got %q", seed, i, wantWeak[i], name)
			}
		}
		rest := map[string]struct{}{got[3]: {}, got[4]: {}}
		if len(rest) != 2 {
			t.Fatalf("seed %d: rest slots not distinct: %v", seed, got)
		}
		for _, name := range []string{"D", "E"} {
			if _, ok := rest[name]; !ok {
				t.Fatalf("seed %d: missing rest topic %q in %v", seed, name, got)
			}
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

func TestMultiDayAdaptiveProgression(t *testing.T) {
	topics := []model.Topic{
		{ID: 1, Name: "Frontend"},
		{ID: 2, Name: "Backend"},
		{ID: 3, Name: "Infrastructure"},
		{ID: 4, Name: "Security"},
	}

	// Day 1: Brand new user, all unattempted.
	var stats []model.UserTopicStats
	rnd := rand.New(rand.NewSource(100))

	day1 := SelectTopics(5, topics, stats, rnd)
	if len(day1) != 5 {
		t.Fatalf("day 1: want 5 questions, got %d", len(day1))
	}

	// User answers Day 1: perfect on Frontend, fails Security and Backend.
	now := time.Now()
	stats = []model.UserTopicStats{
		{TopicID: 1, CorrectCount: 3, WrongCount: 0, AccuracyRate: 100, LastPracticedAt: &now},
		{TopicID: 2, CorrectCount: 0, WrongCount: 2, AccuracyRate: 0, LastPracticedAt: &now},
		{TopicID: 4, CorrectCount: 0, WrongCount: 2, AccuracyRate: 0, LastPracticedAt: &now},
	}

	// Day 2: Security & Backend (0%) and Infrastructure (unattempted) are bottom topics.
	// Frontend (100%) must NOT be in the bottom/weak 70% pool.
	day2 := SelectTopics(5, topics, stats, rnd)
	for i := 0; i < 3; i++ {
		if day2[i] == "Frontend" {
			t.Fatalf("day 2 weak slot %d selected strong topic Frontend", i)
		}
	}

	// Over 5 simulated days, the weak group (Security, Backend, Infrastructure) must
	// surface more frequently (~70/30 split) than the strong group (Frontend).
	weakGroupCount := 0
	strongGroupCount := 0
	for day := 1; day <= 5; day++ {
		selected := SelectTopics(5, topics, stats, rnd)
		for _, name := range selected {
			if name == "Frontend" {
				strongGroupCount++
			} else {
				weakGroupCount++
			}
		}
	}

	if weakGroupCount <= strongGroupCount {
		t.Fatalf("expected weak topic group to surface more frequently than strong topic over 5 days: weak=%d, strong=%d",
			weakGroupCount, strongGroupCount)
	}
}

func ptrTime(v time.Time) *time.Time { return &v }
