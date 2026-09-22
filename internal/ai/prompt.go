package ai

import (
	"strconv"
	"strings"
)

const promptTemplate = `System: You are writing exam questions for a full-stack developer.
Generate {n} multiple-choice questions on the following topics: {topics}
Respond with a JSON array only, no other text, in this format:
[
  {
    "topic": "...",
    "prompt": "...",
    "options": [{"id":"a","text":"..."}, {"id":"b","text":"..."}, {"id":"c","text":"..."}, {"id":"d","text":"..."}],
    "correct_option": "b",
    "explanation": "..."
  }
]`

func buildPrompt(n int, topics []string) string {
	s := strings.ReplaceAll(promptTemplate, "{n}", strconv.Itoa(n))
	return strings.ReplaceAll(s, "{topics}", strings.Join(topics, ", "))
}
