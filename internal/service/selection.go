package service

import (
	"math/rand"
	"sort"

	"github.com/natdanai/quizme/internal/model"
)

// SelectTopics picks n topic names using a deterministic 70/30 adaptive split.
//
// Roughly 70% of the slots are filled from the weakest topics (unattempted first,
// then lowest accuracy_rate), and the remaining ~30% are drawn randomly from the
// rest. The random source is injected so tests can be deterministic.
func SelectTopics(n int, topics []model.Topic, stats []model.UserTopicStats, rnd *rand.Rand) []string {
	if n <= 0 {
		return nil
	}
	if len(topics) == 0 {
		return make([]string, n)
	}

	ranked := rankTopics(topics, stats)

	weakCount := n * 7 / 10
	if weakCount < 1 {
		weakCount = 1
	}
	if weakCount >= n {
		weakCount = n - 1
	}

	weak := make([]string, 0, weakCount)
	for i := 0; i < weakCount && i < len(ranked); i++ {
		weak = append(weak, ranked[i].topic.Name)
	}

	weakSet := make(map[string]struct{}, len(weak))
	for _, w := range weak {
		weakSet[w] = struct{}{}
	}

	rest := make([]string, 0, len(ranked))
	for _, r := range ranked {
		if _, ok := weakSet[r.topic.Name]; !ok {
			rest = append(rest, r.topic.Name)
		}
	}

	result := make([]string, n)
	for i := 0; i < weakCount; i++ {
		result[i] = weak[i%len(weak)]
	}

	fill := rest
	if len(rest) == 0 {
		fill = make([]string, len(ranked))
		for i, r := range ranked {
			fill[i] = r.topic.Name
		}
	}
	rnd.Shuffle(len(fill), func(i, j int) { fill[i], fill[j] = fill[j], fill[i] })
	for i := weakCount; i < n; i++ {
		result[i] = fill[(i-weakCount)%len(fill)]
	}

	return result
}

type rankedTopic struct {
	topic       model.Topic
	unattempted bool
	accuracy    float64
}

func rankTopics(topics []model.Topic, stats []model.UserTopicStats) []rankedTopic {
	statMap := make(map[int]model.UserTopicStats, len(stats))
	for _, s := range stats {
		statMap[s.TopicID] = s
	}

	ranked := make([]rankedTopic, len(topics))
	for i, t := range topics {
		s, ok := statMap[t.ID]
		ranked[i] = rankedTopic{
			topic:       t,
			unattempted: !ok || s.LastPracticedAt == nil,
			accuracy:    s.AccuracyRate,
		}
	}

	sort.Slice(ranked, func(i, j int) bool {
		if ranked[i].unattempted != ranked[j].unattempted {
			return ranked[i].unattempted
		}
		if ranked[i].accuracy != ranked[j].accuracy {
			return ranked[i].accuracy < ranked[j].accuracy
		}
		return ranked[i].topic.ID < ranked[j].topic.ID
	})
	return ranked
}
