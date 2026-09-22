package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/natdanai/quizme/internal/model"
	"github.com/natdanai/quizme/internal/service"
)

func init() {
	gin.SetMode(gin.TestMode)
}

type mockQuizService struct {
	batch       model.QuizBatch
	questions   []model.Question
	topics      []model.Topic
	correct     bool
	explanation *string
	err         error
	answerErr   error
}

func (m *mockQuizService) GetTodayQuiz(ctx context.Context) (model.QuizBatch, []model.Question, error) {
	return m.batch, m.questions, m.err
}

func (m *mockQuizService) Answer(ctx context.Context, questionID int, selectedOption string) (bool, *string, error) {
	return m.correct, m.explanation, m.answerErr
}

func (m *mockQuizService) ListTopics(ctx context.Context) ([]model.Topic, error) {
	return m.topics, m.err
}

type mockStatsService struct {
	stats []model.UserTopicStats
	err   error
}

func (m *mockStatsService) GetStats(ctx context.Context) ([]model.UserTopicStats, error) {
	return m.stats, m.err
}

type mockTopicService struct {
	topics    []model.Topic
	created   model.Topic
	err       error
	createErr error
}

func (m *mockTopicService) ListTopics(ctx context.Context) ([]model.Topic, error) {
	return m.topics, m.err
}

func (m *mockTopicService) CreateTopic(ctx context.Context, name string, weight int) (model.Topic, error) {
	return m.created, m.createErr
}

type mockGenerateService struct {
	err error
}

func (m *mockGenerateService) Generate(ctx context.Context) error {
	return m.err
}

func TestGetTodayQuiz(t *testing.T) {
	batch := model.QuizBatch{BatchDate: time.Date(2026, 9, 22, 0, 0, 0, 0, time.UTC), Status: "success"}
	questions := []model.Question{
		{
			ID:            1,
			TopicID:       7,
			Prompt:        "What is 2+2?",
			Options:       []model.Option{{ID: "a", Text: "3"}, {ID: "b", Text: "4"}},
			CorrectOption: "b",
			Explanation:   strPtr("math"),
		},
	}
	topics := []model.Topic{{ID: 7, Name: "Math"}}

	router := gin.New()
	svc := &mockQuizService{batch: batch, questions: questions, topics: topics}
	router.GET("/quiz/today", GetTodayQuiz(svc))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/quiz/today", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", w.Code, w.Body.String())
	}

	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if body["batch_date"] != "2026-09-22" {
		t.Fatalf("want batch_date 2026-09-22, got %v", body["batch_date"])
	}

	qs := body["questions"].([]any)
	if len(qs) != 1 {
		t.Fatalf("want 1 question, got %d", len(qs))
	}
	q := qs[0].(map[string]any)
	if q["topic"] != "Math" {
		t.Fatalf("want topic Math, got %v", q["topic"])
	}

	bodyStr := w.Body.String()
	if strings.Contains(bodyStr, "correct_option") {
		t.Fatal("response leaks correct_option")
	}
	if strings.Contains(bodyStr, "explanation") {
		t.Fatal("response leaks explanation")
	}
}

func TestGetTodayQuizNoQuiz(t *testing.T) {
	router := gin.New()
	svc := &mockQuizService{err: service.ErrNoQuiz}
	router.GET("/quiz/today", GetTodayQuiz(svc))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/quiz/today", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("want 404, got %d", w.Code)
	}
}

func TestSubmitAnswerCorrect(t *testing.T) {
	router := gin.New()
	exp := "because"
	svc := &mockQuizService{correct: true, explanation: &exp}
	router.POST("/quiz/answer", SubmitAnswer(svc))

	w := httptest.NewRecorder()
	body := `{"question_id":1,"option":"b"}`
	req, _ := http.NewRequest(http.MethodPost, "/quiz/answer", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp["correct"] != true {
		t.Fatalf("want correct=true, got %v", resp["correct"])
	}
	if resp["explanation"] != "because" {
		t.Fatalf("want explanation because, got %v", resp["explanation"])
	}
}

func TestSubmitAnswerInvalidOption(t *testing.T) {
	router := gin.New()
	router.POST("/quiz/answer", SubmitAnswer(&mockQuizService{}))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/quiz/answer", bytes.NewReader([]byte(`{"question_id":1,"option":"e"}`)))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
}

func TestSubmitAnswerNotFound(t *testing.T) {
	router := gin.New()
	svc := &mockQuizService{answerErr: service.ErrNotFound}
	router.POST("/quiz/answer", SubmitAnswer(svc))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/quiz/answer", bytes.NewReader([]byte(`{"question_id":999,"option":"a"}`)))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("want 404, got %d", w.Code)
	}
}

func TestGetStats(t *testing.T) {
	router := gin.New()
	svc := &mockStatsService{stats: []model.UserTopicStats{{TopicID: 1, AccuracyRate: 75}}}
	router.GET("/stats", GetStats(svc))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/stats", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}
	var resp []model.UserTopicStats
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp) != 1 || resp[0].TopicID != 1 {
		t.Fatalf("unexpected stats: %+v", resp)
	}
}

func TestListTopics(t *testing.T) {
	router := gin.New()
	svc := &mockTopicService{topics: []model.Topic{{ID: 1, Name: "Frontend"}}}
	router.GET("/topics", ListTopics(svc))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/topics", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}
}

func TestCreateTopic(t *testing.T) {
	router := gin.New()
	svc := &mockTopicService{created: model.Topic{ID: 4, Name: "Security", Weight: 1}}
	router.POST("/topics", CreateTopic(svc))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/topics", bytes.NewReader([]byte(`{"name":"Security"}`)))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("want 201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateTopicDuplicate(t *testing.T) {
	router := gin.New()
	svc := &mockTopicService{createErr: &pgconn.PgError{Code: "23505"}}
	router.POST("/topics", CreateTopic(svc))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/topics", bytes.NewReader([]byte(`{"name":"Frontend"}`)))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("want 409, got %d", w.Code)
	}
}

func TestCreateTopicEmptyName(t *testing.T) {
	router := gin.New()
	router.POST("/topics", CreateTopic(&mockTopicService{}))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/topics", bytes.NewReader([]byte(`{"name":"  "}`)))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
}

func TestInternalAuth(t *testing.T) {
	router := gin.New()
	router.Use(InternalAuth("secret"))
	router.POST("/internal/generate", func(c *gin.Context) { c.Status(http.StatusOK) })

	cases := []struct {
		name  string
		token string
		want  int
	}{
		{"no token", "", http.StatusUnauthorized},
		{"wrong token", "Bearer wrong", http.StatusUnauthorized},
		{"correct token", "Bearer secret", http.StatusOK},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodPost, "/internal/generate", nil)
			if tc.token != "" {
				req.Header.Set("Authorization", tc.token)
			}
			router.ServeHTTP(w, req)
			if w.Code != tc.want {
				t.Fatalf("want %d, got %d", tc.want, w.Code)
			}
		})
	}
}

func TestInternalAuthEmptyWantToken(t *testing.T) {
	router := gin.New()
	router.Use(InternalAuth(""))
	router.POST("/internal/generate", func(c *gin.Context) { c.Status(http.StatusOK) })

	cases := []struct {
		name  string
		token string
		want  int
	}{
		{"unset rejects no header", "", http.StatusUnauthorized},
		{"unset rejects empty bearer", "Bearer ", http.StatusUnauthorized},
		{"unset rejects any token", "Bearer secret", http.StatusUnauthorized},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodPost, "/internal/generate", nil)
			if tc.token != "" {
				req.Header.Set("Authorization", tc.token)
			}
			router.ServeHTTP(w, req)
			if w.Code != tc.want {
				t.Fatalf("want %d, got %d", tc.want, w.Code)
			}
		})
	}
}

func TestGenerate(t *testing.T) {
	router := gin.New()
	svc := &mockGenerateService{}
	router.POST("/internal/generate", Generate(svc))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/internal/generate", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}
}

func TestGenerateError(t *testing.T) {
	router := gin.New()
	svc := &mockGenerateService{err: errors.New("boom")}
	router.POST("/internal/generate", Generate(svc))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/internal/generate", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("want 500, got %d", w.Code)
	}
}

func strPtr(s string) *string { return &s }
