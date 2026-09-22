package handler

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/natdanai/quizme/internal/model"
)

// TopicService is the handler's view of topic operations.
type TopicService interface {
	ListTopics(ctx context.Context) ([]model.Topic, error)
	CreateTopic(ctx context.Context, name string, weight int) (model.Topic, error)
}

// ListTopics handles GET /topics.
func ListTopics(svc TopicService) gin.HandlerFunc {
	return func(c *gin.Context) {
		topics, err := svc.ListTopics(c.Request.Context())
		if err != nil {
			log.Printf("list topics: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
			return
		}
		c.JSON(http.StatusOK, topics)
	}
}

type createTopicRequest struct {
	Name string `json:"name" binding:"required"`
}

// CreateTopic handles POST /topics.
func CreateTopic(svc TopicService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req createTopicRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
			return
		}

		name := strings.TrimSpace(req.Name)
		if name == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
			return
		}

		topic, err := svc.CreateTopic(c.Request.Context(), name, 1)
		if err != nil {
			if isUniqueViolation(err) {
				c.JSON(http.StatusConflict, gin.H{"error": "topic already exists"})
				return
			}
			log.Printf("create topic: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
			return
		}

		c.JSON(http.StatusCreated, topic)
	}
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == pgerrcode.UniqueViolation
	}
	return false
}
