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
	startTime := time.Now() // Start overall timer

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
		cacheStartTime := time.Now()
		err := Cache.GetJSON(cacheKey, &cachedResponse)
		cacheDuration := time.Since(cacheStartTime)
		
		if err == nil {

			// logging data so I can track metrics
			totalDuration := time.Since(startTime)
			Logger.Info("Cache HIT",
				zap.String("query", searchQuery),
				zap.Duration("cache_lookup_ms", cacheDuration),
				zap.Duration("total_ms", totalDuration),
				zap.Int("num_results", len(cachedResponse.Docs)))
			
			c.JSON(http.StatusOK, gin.H{
				"query":       query,
				"numFound":    cachedResponse.NumFound,
				"results":     cachedResponse.Docs,
				"cached":      true,
				"responseTime": fmt.Sprintf("%.2fms", totalDuration.Seconds()*1000),
			})
			return
		} else if err != redis.Nil {
			Logger.Warn("Cache error", zap.Error(err), zap.Duration("duration_ms", cacheDuration))
		} else {
			Logger.Info("Cache MISS", 
				zap.String("query", searchQuery),
				zap.Duration("cache_lookup_ms", cacheDuration))
		}
	}

	Logger.Info("Calling API", zap.String("query", searchQuery))

	searchURL := fmt.Sprintf("%s%s%s%s%s", 
		constants.OpenLibraryAPIURL, 
		constants.OpenLibrarySearchEndpoint, 
		searchQuery, 
		constants.QueryLimit, 
		"3")
	
	// Time the API call
	apiStartTime := time.Now()
	response, err := http.Get(searchURL)
	apiDuration := time.Since(apiStartTime)
	
	if err != nil {
		Logger.Error("API call failed", 
			zap.Error(err),
			zap.Duration("api_duration_ms", apiDuration))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get search results",
		})
		return
	}

	Logger.Info("API response received", 
		zap.Int("statusCode", response.StatusCode),
		zap.Duration("api_duration_ms", apiDuration))
	defer response.Body.Close()

	readStartTime := time.Now()
	body, err := io.ReadAll(response.Body)
	readDuration := time.Since(readStartTime)
	
	if err != nil {
		Logger.Error("Error reading response body", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to read response body",
		})
		return
	}

	Logger.Debug("Response body read", 
		zap.Int("body_size_bytes", len(body)),
		zap.Duration("read_duration_ms", readDuration))

	parseStartTime := time.Now()
	var apiResponse OpenLibraryResponse
	err = json.Unmarshal(body, &apiResponse)
	parseDuration := time.Since(parseStartTime)
	
	if err != nil {
		Logger.Error("Error unmarshalling response body", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to parse API response",
		})
		return
	}

	totalDuration := time.Since(startTime)
	
	Logger.Info("API search completed",
		zap.Int("numFound", apiResponse.NumFound),
		zap.Int("numReturned", len(apiResponse.Docs)),
		zap.Duration("parse_duration_ms", parseDuration),
		zap.Duration("total_duration_ms", totalDuration))

	// Store in cache ADJUST TIME TO HOLD CACHED DATA HERE
	if Cache != nil {
		cacheWriteStart := time.Now()
		err = Cache.Set(cacheKey, apiResponse, 2*time.Minute)
		cacheWriteDuration := time.Since(cacheWriteStart)
		
		if err != nil {
			Logger.Warn("Failed to cache result", 
				zap.Error(err),
				zap.Duration("cache_write_duration_ms", cacheWriteDuration))
		} else {
			Logger.Info("Result cached successfully", 
				zap.String("key", cacheKey),
				zap.Duration("cache_write_duration_ms", cacheWriteDuration))
		}
	}

	// Performance summary
	Logger.Info("⚡ Performance Summary",
		zap.String("query", searchQuery),
		zap.Duration("api_call_ms", apiDuration),
		zap.Duration("total_request_ms", totalDuration),
		zap.Float64("api_percentage", (apiDuration.Seconds()/totalDuration.Seconds())*100))

	c.JSON(http.StatusOK, gin.H{
		"query":        query,
		"numFound":     apiResponse.NumFound,
		"results":      apiResponse.Docs,
		"cached":       false,
		"responseTime": fmt.Sprintf("%.2fms", totalDuration.Seconds()*1000),
		"metrics": gin.H{
			"api_call_ms":    fmt.Sprintf("%.2f", apiDuration.Seconds()*1000),
			"total_ms":       fmt.Sprintf("%.2f", totalDuration.Seconds()*1000),
			"parse_ms":       fmt.Sprintf("%.2f", parseDuration.Seconds()*1000),
		},
	})
}