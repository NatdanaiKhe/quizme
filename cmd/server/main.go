package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/natdanai/quizme/internal/ai"
	"github.com/natdanai/quizme/internal/config"
	"github.com/natdanai/quizme/internal/handler"
	"github.com/natdanai/quizme/internal/repository"
	"github.com/natdanai/quizme/internal/service"
)

func main() {
	cfg := config.Load()

	ctx := os_ctx()
	repo, err := repository.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer repo.Close()

	aiClient := ai.New(cfg.AIBaseURL, cfg.AIAPIKey, cfg.AIModel)
	notifier := service.NewWebhookNotifier(cfg.WebhookURL, 5*time.Second)
	svc := service.New(repo, aiClient, notifier)

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(corsMiddleware(defaultString(os.Getenv("CORS_ORIGIN"), "http://localhost:5173")))

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	r.GET("/quiz/today", handler.GetTodayQuiz(svc))
	r.POST("/quiz/answer", handler.SubmitAnswer(svc))
	r.GET("/stats", handler.GetStats(svc))
	r.GET("/topics", handler.ListTopics(svc))
	r.POST("/topics", handler.CreateTopic(svc))

	internal := r.Group("/internal", handler.InternalAuth(cfg.InternalToken))
	internal.POST("/generate", handler.Generate(svc))

	addr := ":" + cfg.Port
	log.Printf("quizme server listening on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

func corsMiddleware(allowed string) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin != "" && (allowed == "*" || origin == allowed) {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		}
		if c.Request.Method == http.MethodOptions {
			c.Status(http.StatusNoContent)
			c.Abort()
			return
		}
		c.Next()
	}
}

func defaultString(v, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return v
}

func os_ctx() context.Context {
	return context.Background()
}
