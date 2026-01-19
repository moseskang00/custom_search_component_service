package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/moseskang00/custom_search_component_service/common/constants"
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

	c.JSON(http.StatusOK, gin.H{
		"query":    query,
		"numFound": apiResponse.NumFound,
		"results":  apiResponse.Docs,
	})
}

