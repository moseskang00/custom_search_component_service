package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/moseskang00/custom_search_component_service/common/constants"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"github.com/agnivade/levenshtein"
)

// CacheMatch represents a fuzzy cache match result
type CacheMatch struct {
	Key          string
	CachedQuery  string
	Score        float64
	Method       string
}

// findSimilarCachedQueries finds similar queries in cache using fuzzy matching
func findSimilarCachedQueries(query string, maxResults int) []CacheMatch {
	if Cache == nil {
		return nil
	}

	normalized := normalizeQuery(query)
	queryWords := strings.Split(normalized, " ")
	
	// Get all search cache keys
	pattern := "search:*"
	allKeys, err := Cache.Keys(pattern)
	if err != nil {
		Logger.Warn("Failed to get cache keys for fuzzy matching", zap.Error(err))
		return nil
	}
	
	matches := []CacheMatch{}
	maxLevenshteinDistance := 3 // Maximum edit distance for whole query
	
	for _, key := range allKeys {
		cachedQuery := strings.TrimPrefix(key, "search:")
		
		// Skip exact matches (handled elsewhere)
		if cachedQuery == normalized {
			continue
		}
		
		// Method 1: Levenshtein distance for whole query
		distance := levenshtein.ComputeDistance(normalized, cachedQuery)
		if distance <= maxLevenshteinDistance {
			score := 1.0 / float64(distance+1) // Lower distance = higher score
			matches = append(matches, CacheMatch{
				Key:         key,
				CachedQuery: cachedQuery,
				Score:       score,
				Method:      "levenshtein",
			})
			continue
		}
		
		// Method 2: Word-by-word fuzzy matching
		cachedWords := strings.Split(cachedQuery, " ")
		matchingWords := 0
		
		for _, qWord := range queryWords {
			for _, cWord := range cachedWords {
				wordDistance := levenshtein.ComputeDistance(qWord, cWord)
				if wordDistance <= 2 {
					matchingWords++
					break
				}
			}
		}
		
		// If most words match, consider it similar
		maxLen := len(queryWords)
		if len(cachedWords) > maxLen {
			maxLen = len(cachedWords)
		}
		wordMatchRatio := float64(matchingWords) / float64(maxLen)
		
		if wordMatchRatio >= 0.6 { // 60% of words match
			matches = append(matches, CacheMatch{
				Key:         key,
				CachedQuery: cachedQuery,
				Score:       wordMatchRatio,
				Method:      "word-match",
			})
		}
	}
	
	// Sort by score (best matches first)
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].Score > matches[j].Score
	})
	
	// Return top N results
	if len(matches) > maxResults {
		matches = matches[:maxResults]
	}
	
	return matches
}

// normalizeQuery cleans and normalizes the search query
func normalizeQuery(query string) string {
	query = strings.ToLower(query)
	query = strings.TrimSpace(query)
	
	// Remove special characters (keep only letters, numbers, and spaces)
	reg := regexp.MustCompile(`[^\w\s]`)
	query = reg.ReplaceAllString(query, "")

	spaceReg := regexp.MustCompile(`\s+`)
	query = spaceReg.ReplaceAllString(query, " ")
	
	return query
}

// generateCacheKeyVariations creates multiple cache key variations to handle typos and different orderings
func generateCacheKeyVariations(query string) []string {
	normalized := normalizeQuery(query)
	queryWords := strings.Split(normalized, " ")
	
	variations := []string{
		normalized, // "project hail mary"
	}
	
	// Sorted words: "hail mary project"
	sortedWords := make([]string, len(queryWords))
	copy(sortedWords, queryWords)
	sort.Strings(sortedWords)
	variations = append(variations, strings.Join(sortedWords, " "))
	
	// Filter words longer than 3 characters (remove small words)
	longWords := []string{}
	for _, word := range queryWords {
		if len(word) > 3 {
			longWords = append(longWords, word)
		}
	}

	if len(longWords) > 0 {
		variations = append(variations, strings.Join(longWords, " "))
	}
	
	// No spaces: "projecthailmary"
	variations = append(variations, strings.Join(queryWords, ""))
	
	// Remove duplicates
	seen := make(map[string]bool)
	result := []string{}
	for _, v := range variations {
		if v != "" && !seen[v] {
			seen[v] = true
			result = append(result, v)
		}
	}
	
	return result
}

// checkCache attempts to retrieve cached results for a search query
// Tries multiple cache key variations to handle typos and different orderings
func checkCache(c *gin.Context, query string, searchQuery string, startTime time.Time) (bool, string) {
	if Cache == nil {
		return false, ""
	}

	// Generate all possible cache key variations
	variations := generateCacheKeyVariations(query)
	Logger.Info("Trying cache key variations", 
		zap.Int("num_variations", len(variations)),
		zap.Strings("variations", variations))
	
	cacheStartTime := time.Now()
	var cachedResponse OpenLibraryResponse
	
	// Try each variation until we find a hit
	for _, variation := range variations {
		cacheKey := fmt.Sprintf("search:%s", variation)
		err := Cache.GetJSON(cacheKey, &cachedResponse)
		
		if err == nil {
			// Cache HIT!
			cacheDuration := time.Since(cacheStartTime)
			totalDuration := time.Since(startTime)
			
			Logger.Info("Cache HIT",
				zap.String("original_query", query),
				zap.String("matched_variation", variation),
				zap.String("cache_key", cacheKey),
				zap.Duration("cache_lookup_ms", cacheDuration),
				zap.Duration("total_ms", totalDuration),
				zap.Int("num_results", len(cachedResponse.Docs)))
			
			c.JSON(http.StatusOK, gin.H{
				"query":         query,
				"numFound":      cachedResponse.NumFound,
				"results":       cachedResponse.Docs,
				"cached":        true,
				"cacheKey":      variation,
				"responseTime":  fmt.Sprintf("%.2fms", totalDuration.Seconds()*1000),
			})
			return true, cacheKey
		} else if err != redis.Nil {
			Logger.Warn("Cache error", 
				zap.String("key", cacheKey),
				zap.Error(err))
		}
	}
	
	// No exact match found, try fuzzy matching
	Logger.Info("Trying fuzzy matching", zap.String("query", query))
	fuzzyMatches := findSimilarCachedQueries(query, 5)
	
	if len(fuzzyMatches) > 0 {
		// Try the best fuzzy match
		bestMatch := fuzzyMatches[0]
		Logger.Info("Found fuzzy matches",
			zap.Int("num_matches", len(fuzzyMatches)),
			zap.String("best_match", bestMatch.CachedQuery),
			zap.Float64("score", bestMatch.Score),
			zap.String("method", bestMatch.Method))
		
		// Try to get the fuzzy match from cache
		err := Cache.GetJSON(bestMatch.Key, &cachedResponse)
		if err == nil {
			cacheDuration := time.Since(cacheStartTime)
			totalDuration := time.Since(startTime)
			
			Logger.Info("Cache HIT (fuzzy match)",
				zap.String("original_query", query),
				zap.String("matched_query", bestMatch.CachedQuery),
				zap.Float64("similarity_score", bestMatch.Score),
				zap.String("match_method", bestMatch.Method),
				zap.Duration("cache_lookup_ms", cacheDuration),
				zap.Duration("total_ms", totalDuration),
				zap.Int("num_results", len(cachedResponse.Docs)))
			
			c.JSON(http.StatusOK, gin.H{
				"query":           query,
				"numFound":        cachedResponse.NumFound,
				"results":         cachedResponse.Docs,
				"cached":          true,
				"fuzzyMatch":      true,
				"matchedQuery":    bestMatch.CachedQuery,
				"similarityScore": bestMatch.Score,
				"responseTime":    fmt.Sprintf("%.2fms", totalDuration.Seconds()*1000),
			})
			return true, bestMatch.Key
		}
	}
	
	// Cache MISS on all variations (including fuzzy)
	cacheDuration := time.Since(cacheStartTime)
	Logger.Info("Cache MISS (all variations + fuzzy)",
		zap.String("query", searchQuery),
		zap.Int("variations_tried", len(variations)),
		zap.Int("fuzzy_matches_found", len(fuzzyMatches)),
		zap.Duration("total_lookup_ms", cacheDuration))
	
	return false, ""
}

func Search(c *gin.Context) {
	startTime := time.Now() // Start overall timer

	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Search query parameter 'q' is required",
		})
		return
	}
	normalizedQuery := normalizeQuery(query)
	Logger.Info("Moses kang normalized query", zap.String("normalizedQuery", normalizedQuery))

	queryWords := strings.Split(normalizedQuery, " ")
	searchQuery := strings.Join(queryWords, "+")

	Logger.Info("Search request received", zap.String("query", searchQuery))

	// Try to get from cache first (tries multiple variations)
	cacheHit, _ := checkCache(c, query, searchQuery, startTime)
	if cacheHit {
		return
	}

	Logger.Info("Cache Miss, Calling API", zap.String("query", searchQuery))

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

	// Store in cache ADJUST TIME TO HOLD CACHED DATA IN CONSTANTS FILE
	if Cache != nil {
		// Generate cache key for storing the result
		cacheKey := fmt.Sprintf("search:%s", normalizedQuery)
		
		cacheWriteStart := time.Now()
		err = Cache.Set(cacheKey, apiResponse, constants.CACHE_TTL_MINUTES*time.Minute)
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