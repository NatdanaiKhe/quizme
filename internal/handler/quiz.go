package handler

import (
	"context"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/natdanai/quizme/internal/model"
	"github.com/natdanai/quizme/internal/service"
)

// QuizService is the handler's view of the service layer.
type QuizService interface {
	GetTodayQuiz(ctx context.Context) (model.QuizBatch, []model.Question, error)
	Answer(ctx context.Context, questionID int, selectedOption string) (bool, *string, error)
	ListTopics(ctx context.Context) ([]model.Topic, error)
}

// StatsService is the handler's view of stats operations.
type StatsService interface {
	GetStats(ctx context.Context) ([]model.UserTopicStats, error)
}

// publicQuestion is the answer-safe question shape sent to clients.
type publicQuestion struct {
	ID      int            `json:"id"`
	Topic   string         `json:"topic"`
	Prompt  string         `json:"prompt"`
	Options []model.Option `json:"options"`
}

type todayResponse struct {
	BatchDate string           `json:"batch_date"`
	Questions []publicQuestion `json:"questions"`
}

type answerRequest struct {
	QuestionID int    `json:"question_id" binding:"required"`
	Option     string `json:"option" binding:"required"`
}

type answerResponse struct {
	Correct     bool   `json:"correct"`
	Explanation string `json:"explanation"`
}

// GetTodayQuiz handles GET /quiz/today.
func GetTodayQuiz(svc QuizService) gin.HandlerFunc {
	return func(c *gin.Context) {
		batch, questions, err := svc.GetTodayQuiz(c.Request.Context())
		if err != nil {
			if errors.Is(err, service.ErrNoQuiz) {
				c.JSON(http.StatusNotFound, gin.H{"error": "no quiz available"})
				return
			}
			log.Printf("get today quiz: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
			return
		}

		topics, err := svc.ListTopics(c.Request.Context())
		if err != nil {
			log.Printf("list topics for today quiz: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
			return
		}
		topicByID := make(map[int]string, len(topics))
		for _, t := range topics {
			topicByID[t.ID] = t.Name
		}

		public := make([]publicQuestion, len(questions))
		for i, q := range questions {
			public[i] = publicQuestion{
				ID:      q.ID,
				Topic:   topicName(topicByID, q.TopicID),
				Prompt:  q.Prompt,
				Options: q.Options,
			}
		}

		c.JSON(http.StatusOK, todayResponse{
			BatchDate: batch.BatchDate.Format(time.DateOnly),
			Questions: public,
		})
	}
}

// SubmitAnswer handles POST /quiz/answer.
func SubmitAnswer(svc QuizService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req answerRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
			return
		}

		if !validOption(req.Option) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "option must be a, b, c, or d"})
			return
		}

		correct, explanation, err := svc.Answer(c.Request.Context(), req.QuestionID, req.Option)
		if err != nil {
			if errors.Is(err, service.ErrNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"error": "question not found"})
				return
			}
			log.Printf("submit answer: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
			return
		}

		resp := answerResponse{Correct: correct}
		if explanation != nil {
			resp.Explanation = *explanation
		}
		c.JSON(http.StatusOK, resp)
	}
}

// GetStats handles GET /stats.
func GetStats(svc StatsService) gin.HandlerFunc {
	return func(c *gin.Context) {
		stats, err := svc.GetStats(c.Request.Context())
		if err != nil {
			log.Printf("get stats: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
			return
		}
		c.JSON(http.StatusOK, stats)
	}
}

func validOption(o string) bool {
	return o == "a" || o == "b" || o == "c" || o == "d"
}

func topicName(names map[int]string, id int) string {
	if name, ok := names[id]; ok {
		return name
	}
	return ""
}
