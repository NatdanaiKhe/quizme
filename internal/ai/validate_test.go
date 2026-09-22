package ai

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/natdanai/quizme/internal/model"
)

func validQuestion(topic, correct string) GeneratedQuestion {
	return GeneratedQuestion{
		Topic:         topic,
		Prompt:        "What is 2+2?",
		Options:       []model.Option{{ID: "a", Text: "3"}, {ID: "b", Text: "4"}, {ID: "c", Text: "5"}, {ID: "d", Text: "6"}},
		CorrectOption: correct,
		Explanation:   "Two plus two equals four.",
	}
}

func makeValidJSON(n int) string {
	qs := make([]GeneratedQuestion, n)
	for i := 0; i < n; i++ {
		qs[i] = validQuestion("Frontend", "b")
	}
	b, _ := json.Marshal(qs)
	return string(b)
}

func TestParseAndValidate(t *testing.T) {
	topics := []string{"Frontend", "Backend"}

	cases := []struct {
		name      string
		raw       string
		n         int
		wantValid bool
	}{
		{
			name:      "valid 5 questions",
			raw:       makeValidJSON(5),
			n:         5,
			wantValid: true,
		},
		{
			name:      "fenced json",
			raw:       "```json\n" + makeValidJSON(5) + "\n```",
			n:         5,
			wantValid: true,
		},
		{
			name:      "malformed json",
			raw:       "[{not json",
			n:         5,
			wantValid: false,
		},
		{
			name:      "wrong count 4",
			raw:       makeValidJSON(4),
			n:         5,
			wantValid: false,
		},
		{
			name:      "wrong count 6",
			raw:       makeValidJSON(6),
			n:         5,
			wantValid: false,
		},
		{
			name: "3 options",
			raw: func() string {
				q := validQuestion("Frontend", "b")
				q.Options = q.Options[:3]
				b, _ := json.Marshal([]GeneratedQuestion{q, q, q, q, q})
				return string(b)
			}(),
			n:         5,
			wantValid: false,
		},
		{
			name: "duplicate option id",
			raw: func() string {
				q := validQuestion("Frontend", "b")
				q.Options[3].ID = "a"
				b, _ := json.Marshal([]GeneratedQuestion{q, q, q, q, q})
				return string(b)
			}(),
			n:         5,
			wantValid: false,
		},
		{
			name: "missing option id e",
			raw: func() string {
				q := validQuestion("Frontend", "b")
				q.Options[3].ID = "e"
				b, _ := json.Marshal([]GeneratedQuestion{q, q, q, q, q})
				return string(b)
			}(),
			n:         5,
			wantValid: false,
		},
		{
			name: "correct option e",
			raw: func() string {
				q := validQuestion("Frontend", "e")
				b, _ := json.Marshal([]GeneratedQuestion{q, q, q, q, q})
				return string(b)
			}(),
			n:         5,
			wantValid: false,
		},
		{
			name: "correct option empty",
			raw: func() string {
				q := validQuestion("Frontend", "")
				b, _ := json.Marshal([]GeneratedQuestion{q, q, q, q, q})
				return string(b)
			}(),
			n:         5,
			wantValid: false,
		},
		{
			name: "missing topic key",
			raw: func() string {
				q := validQuestion("Frontend", "b")
				q.Topic = ""
				b, _ := json.Marshal([]GeneratedQuestion{q, q, q, q, q})
				return string(b)
			}(),
			n:         5,
			wantValid: false,
		},
		{
			name: "correct option uppercase B",
			raw: func() string {
				q := validQuestion("Frontend", "B")
				b, _ := json.Marshal([]GeneratedQuestion{q, q, q, q, q})
				return string(b)
			}(),
			n:         5,
			wantValid: false,
		},
		{
			name: "empty prompt",
			raw: func() string {
				q := validQuestion("Frontend", "b")
				q.Prompt = "   "
				b, _ := json.Marshal([]GeneratedQuestion{q, q, q, q, q})
				return string(b)
			}(),
			n:         5,
			wantValid: false,
		},
		{
			name: "empty option text",
			raw: func() string {
				q := validQuestion("Frontend", "b")
				q.Options[1].Text = ""
				b, _ := json.Marshal([]GeneratedQuestion{q, q, q, q, q})
				return string(b)
			}(),
			n:         5,
			wantValid: false,
		},
		{
			name: "missing explanation",
			raw: func() string {
				q := validQuestion("Frontend", "b")
				q.Explanation = ""
				b, _ := json.Marshal([]GeneratedQuestion{q, q, q, q, q})
				return string(b)
			}(),
			n:         5,
			wantValid: false,
		},
		{
			name:      "non array",
			raw:       `{"topic":"Frontend"}`,
			n:         5,
			wantValid: false,
		},
		{
			name: "topic not in list",
			raw: func() string {
				q := validQuestion("DevOps", "b")
				b, _ := json.Marshal([]GeneratedQuestion{q, q, q, q, q})
				return string(b)
			}(),
			n:         5,
			wantValid: false,
		},
		{
			name: "topic case insensitive ok",
			raw: func() string {
				q := validQuestion("frontend", "b")
				b, _ := json.Marshal([]GeneratedQuestion{q, q, q, q, q})
				return string(b)
			}(),
			n:         5,
			wantValid: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			qs, err := Parse(tc.raw, tc.n)
			if err != nil && tc.wantValid {
				t.Fatalf("Parse unexpectedly failed: %v", err)
			}
			if err == nil {
				err = Validate(qs, tc.n, topics)
			}
			if tc.wantValid && err != nil {
				t.Fatalf("expected valid, got: %v", err)
			}
			if !tc.wantValid {
				var v *ValidationError
				if !errors.As(err, &v) {
					t.Fatalf("expected *ValidationError, got %T: %v", err, err)
				}
			}
		})
	}
}

func TestParseFenceVariants(t *testing.T) {
	payload := makeValidJSON(5)
	variants := []string{
		payload,
		"```json\n" + payload + "\n```",
		"```JSON\n" + payload + "\n```",
		"```\n" + payload + "\n```",
		"  ```json\n" + payload + "\n```  ",
	}
	for i, v := range variants {
		qs, err := Parse(v, 5)
		if err != nil {
			t.Fatalf("variant %d parse failed: %v", i, err)
		}
		if err := Validate(qs, 5, []string{"Frontend"}); err != nil {
			t.Fatalf("variant %d validate failed: %v", i, err)
		}
	}
}

func TestValidationErrorNamesItem(t *testing.T) {
	qs := make([]GeneratedQuestion, 5)
	for i := range qs {
		qs[i] = validQuestion("Frontend", "b")
	}
	qs[2].CorrectOption = "x"
	qs[4].Options[0].ID = ""

	err := Validate(qs, 5, []string{"Frontend"})
	v, ok := err.(*ValidationError)
	if !ok {
		t.Fatalf("expected *ValidationError, got %T", err)
	}
	hasQ2, hasQ4 := false, false
	for _, r := range v.Reasons {
		if strings.Contains(r, "q[2]") {
			hasQ2 = true
		}
		if strings.Contains(r, "q[4]") {
			hasQ4 = true
		}
	}
	if !hasQ2 {
		t.Fatalf("expected reason naming q[2], got: %v", err)
	}
	if !hasQ4 {
		t.Fatalf("expected reason naming q[4], got: %v", err)
	}
}

func TestValidationErrorMessageCap(t *testing.T) {
	qs := make([]GeneratedQuestion, 5)
	for i := range qs {
		qs[i] = validQuestion("Frontend", "e")
	}
	err := Validate(qs, 5, []string{"Frontend"})
	v := err.(*ValidationError)
	joined := strings.Join(v.Reasons, "")
	if !strings.Contains(joined, "...") {
		t.Fatalf("expected cap indicator in error, got: %v", err)
	}
}
