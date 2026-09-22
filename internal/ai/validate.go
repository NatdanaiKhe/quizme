package ai

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/natdanai/quizme/internal/model"
)

// GeneratedQuestion is the validated shape returned by the AI.
// It intentionally mirrors the DB question row but is not serialized to HTTP.
type GeneratedQuestion struct {
	Topic         string         `json:"topic"`
	Prompt        string         `json:"prompt"`
	Options       []model.Option `json:"options"`
	CorrectOption string         `json:"correct_option"`
	Explanation   string         `json:"explanation"`
}

// ValidationError means the AI returned content but it failed structural checks.
type ValidationError struct {
	Reasons []string
}

func (e *ValidationError) Error() string {
	if len(e.Reasons) == 0 {
		return "ai response validation failed"
	}
	return "ai response validation failed: " + strings.Join(e.Reasons, "; ")
}

// Parse strips markdown fences and unmarshals the raw AI content.
func Parse(raw string, n int) ([]GeneratedQuestion, error) {
	content := stripFences(raw)
	var qs []GeneratedQuestion
	if err := json.Unmarshal([]byte(content), &qs); err != nil {
		return nil, &ValidationError{Reasons: []string{fmt.Sprintf("json parse: %v", err)}}
	}
	return qs, nil
}

func stripFences(raw string) string {
	s := strings.TrimSpace(raw)
	for _, prefix := range []string{"```json", "```"} {
		if strings.HasPrefix(strings.ToLower(s), strings.ToLower(prefix)) {
			s = strings.TrimPrefix(s, prefix)
			s = strings.TrimPrefix(s, "```JSON")
			s = strings.TrimPrefix(s, "```json")
			s = strings.TrimPrefix(s, "```")
			break
		}
	}
	s = strings.TrimSpace(s)
	if strings.HasSuffix(s, "```") {
		s = strings.TrimSuffix(s, "```")
		s = strings.TrimSpace(s)
	}
	return s
}

// Validate checks every structural rule and returns one aggregated error.
func Validate(qs []GeneratedQuestion, n int, topics []string) error {
	topicSet := make(map[string]struct{}, len(topics))
	for _, t := range topics {
		topicSet[strings.ToLower(strings.TrimSpace(t))] = struct{}{}
	}

	var reasons []string
	if len(qs) != n {
		reasons = append(reasons, fmt.Sprintf("want %d questions, got %d", n, len(qs)))
	}

	for i, q := range qs {
		prefix := fmt.Sprintf("q[%d]", i)
		qtopic := strings.ToLower(strings.TrimSpace(q.Topic))
		if strings.TrimSpace(q.Topic) == "" {
			reasons = append(reasons, prefix+": topic empty")
		} else if _, ok := topicSet[qtopic]; !ok {
			reasons = append(reasons, prefix+": topic '"+q.Topic+"' not in requested topics")
		}
		if strings.TrimSpace(q.Prompt) == "" {
			reasons = append(reasons, prefix+": prompt empty")
		}
		if strings.TrimSpace(q.Explanation) == "" {
			reasons = append(reasons, prefix+": explanation empty")
		}
		if len(q.Options) != 4 {
			reasons = append(reasons, fmt.Sprintf("%s: want 4 options, got %d", prefix, len(q.Options)))
		} else {
			seen := make(map[string]struct{}, 4)
			valid := true
			for _, opt := range q.Options {
				if opt.ID != "a" && opt.ID != "b" && opt.ID != "c" && opt.ID != "d" {
					reasons = append(reasons, fmt.Sprintf("%s: invalid option id %q", prefix, opt.ID))
					valid = false
				}
				if strings.TrimSpace(opt.Text) == "" {
					reasons = append(reasons, fmt.Sprintf("%s: option %q text empty", prefix, opt.ID))
					valid = false
				}
				if _, dup := seen[opt.ID]; dup {
					reasons = append(reasons, fmt.Sprintf("%s: duplicate option id %q", prefix, opt.ID))
					valid = false
				}
				seen[opt.ID] = struct{}{}
			}
			if valid {
				if _, ok := seen[q.CorrectOption]; !ok {
					reasons = append(reasons, fmt.Sprintf("%s: correct_option %q not in options", prefix, q.CorrectOption))
				}
			}
		}
		if q.CorrectOption != "a" && q.CorrectOption != "b" && q.CorrectOption != "c" && q.CorrectOption != "d" {
			reasons = append(reasons, fmt.Sprintf("%s: correct_option %q invalid", prefix, q.CorrectOption))
		}
		if len(reasons) > 8 {
			reasons = append(reasons, "...")
			break
		}
	}

	if len(reasons) == 0 {
		return nil
	}
	return &ValidationError{Reasons: reasons}
}

// IsValidationError reports whether err is a *ValidationError.
func IsValidationError(err error) bool {
	var v *ValidationError
	return errors.As(err, &v)
}
