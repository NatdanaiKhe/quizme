package handler

import (
	"context"
	"crypto/subtle"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// GenerateService triggers today's question generation.
type GenerateService interface {
	Generate(ctx context.Context) error
}

// InternalAuth returns a Gin middleware that requires a Bearer token equal to
// wantToken using constant-time comparison.
func InternalAuth(wantToken string) gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		const prefix = "Bearer "
		if !strings.HasPrefix(auth, prefix) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			c.Abort()
			return
		}
		got := strings.TrimPrefix(auth, prefix)
		if subtle.ConstantTimeCompare([]byte(got), []byte(wantToken)) != 1 {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			c.Abort()
			return
		}
		c.Next()
	}
}

// Generate handles POST /internal/generate.
func Generate(svc GenerateService) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := svc.Generate(c.Request.Context()); err != nil {
			log.Printf("generate: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	}
}
