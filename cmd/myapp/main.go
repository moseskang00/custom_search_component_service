package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	"strings"

	"github.com/moseskang00/custom_search_component_service/common/constants"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
)

var logger *zap.Logger

// OpenLibraryResponse represents the response from OpenLibrary search API
type OpenLibraryResponse struct {
	NumFound      int                      `json:"numFound"`
	Start         int                      `json:"start"`
	NumFoundExact bool                     `json:"numFoundExact"`
	Docs          []map[string]interface{} `json:"docs"`
}

func main() {
	// Load environment variables from .env file (if it exists)
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	// Initialize logger
	var err error
	if os.Getenv("ENV") == "production" {
		logger, err = zap.NewProduction()
	} else {
		logger, err = zap.NewDevelopment()
	}
	if err != nil {
		log.Fatal("Failed to initialize logger:", err)
	}
	defer logger.Sync()

	// Get port from environment or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Set Gin mode
	if os.Getenv("ENV") == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := setupRouter()

	// Create HTTP server
	srv := &http.Server{
		Addr:           ":" + port,
		Handler:        router,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	// Start server in a goroutine
	go func() {
		logger.Info("Starting server", zap.String("port", port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Server failed to start", zap.Error(err))
		}
	}()

	// Wait for interrupt signal to gracefully shut down the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// Graceful shutdown with 5 second timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal("Server forced to shutdown", zap.Error(err))
	}

	logger.Info("Server exited")
}

// make changes for endpoints here
func setupRouter() *gin.Engine {
	router := gin.Default()

	// Add middleware
	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	router.Use(corsMiddleware())

	// Health check endpoint
	router.GET("/health", healthCheckHandler)

	// API routes
	api := router.Group("/api/v1")
	{
		api.GET("/search", searchHandler)
	}

	return router
}

// CORS middleware
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// Health check handler
func healthCheckHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"service": "custom-search-service",
		"time":    time.Now().Format(time.RFC3339),
	})
}

// Search handler (placeholder)
func searchHandler(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Search query parameter 'q' is required",
		})
		return
	}
	queryWords := strings.Split(query, " ")
	searchQuery := strings.Join(queryWords, "+")

	logger.Info("Search request received", zap.String("query", searchQuery))

	searchURL := fmt.Sprintf("%s%s%s%s%s", constants.OpenLibraryAPIURL, constants.OpenLibrarySearchEndpoint, searchQuery, constants.QueryLimit, "3")
	logger.Info("Calling OpenLibrary API", zap.String("searchURL", searchURL))
	
	response, err := http.Get(searchURL)
	if err != nil {
		logger.Error("Error getting search results", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get search results",
		})
		return
	}

	logger.Info("OpenLibrary API response received", zap.Int("statusCode", response.StatusCode))

	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		logger.Error("Error reading response body", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to read response body",
		})
		return
	}

	logger.Info("API Response body", zap.String("body", string(body)))

	var apiResponse OpenLibraryResponse
	err = json.Unmarshal(body, &apiResponse)
	if err != nil {
		logger.Error("Error unmarshalling response body", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to parse API response",
		})
		return
	}

	logger.Info("Search completed", 
		zap.Int("numFound", apiResponse.NumFound),
		zap.Int("numReturned", len(apiResponse.Docs)))
	
	c.JSON(http.StatusOK, gin.H{
		"query":    query,
		"numFound": apiResponse.NumFound,
		"results":  apiResponse.Docs,
	})
}