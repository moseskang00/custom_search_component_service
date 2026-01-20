package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/moseskang00/custom_search_component_service/common/constants"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

func Search(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Search query parameter 'q' is required",
		})
		return
	}

	queryWords := strings.Split(query, " ")
	searchQuery := strings.Join(queryWords, "+")

	Logger.Info("Search request received", zap.String("query", searchQuery))

	// Try to get from cache first
	cacheKey := fmt.Sprintf("search:%s", searchQuery)
	var cachedResponse OpenLibraryResponse
	
	if Cache != nil {
		err := Cache.GetJSON(cacheKey, &cachedResponse)
		if err == nil {
			Logger.Info("Cache hit", zap.String("query", searchQuery))
			c.JSON(http.StatusOK, gin.H{
				"query":    query,
				"numFound": cachedResponse.NumFound,
				"results":  cachedResponse.Docs,
				"cached":   true,
			})
			return
		} else if err != redis.Nil {
			Logger.Warn("Cache error", zap.Error(err))
		}
	}

	Logger.Info("Cache miss, calling API", zap.String("query", searchQuery))

	searchURL := fmt.Sprintf("%s%s%s%s%s", 
		constants.OpenLibraryAPIURL, 
		constants.OpenLibrarySearchEndpoint, 
		searchQuery, 
		constants.QueryLimit, 
		"3")
	Logger.Info("Calling OpenLibrary API", zap.String("searchURL", searchURL))

	response, err := http.Get(searchURL)
	if err != nil {
		Logger.Error("Error getting search results", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get search results",
		})
		return
	}

	Logger.Info("OpenLibrary API response received", zap.Int("statusCode", response.StatusCode))
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		Logger.Error("Error reading response body", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to read response body",
		})
		return
	}

	Logger.Info("API Response body", zap.String("body", string(body)))

	var apiResponse OpenLibraryResponse
	err = json.Unmarshal(body, &apiResponse)
	if err != nil {
		Logger.Error("Error unmarshalling response body", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to parse API response",
		})
		return
	}

	Logger.Info("Search completed",
		zap.Int("numFound", apiResponse.NumFound),
		zap.Int("numReturned", len(apiResponse.Docs)))

	// Store in cache (30 minutes TTL)
	if Cache != nil {
		err = Cache.Set(cacheKey, apiResponse, 2*time.Minute)
		if err != nil {
			Logger.Warn("Failed to cache result", zap.Error(err))
		} else {
			Logger.Info("Result cached", zap.String("key", cacheKey))
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"query":    query,
		"numFound": apiResponse.NumFound,
		"results":  apiResponse.Docs,
		"cached":   false,
	})
}
