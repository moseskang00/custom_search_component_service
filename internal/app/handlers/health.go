package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// HealthCheck handles the health check endpoint
func HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"service": "custom-search-service",
		"time":    time.Now().Format(time.RFC3339),
	})
}

