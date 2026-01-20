package handlers

import (
	"go.uber.org/zap"
	"github.com/moseskang00/custom_search_component_service/internal/cache"
)

var (
	Logger *zap.Logger
	Cache  *cache.Cache
)

func SetLogger(l *zap.Logger) {
	Logger = l
}

func SetCache(c *cache.Cache) {
	Cache = c
}

// OpenLibraryResponse represents the response from OpenLibrary search API
type OpenLibraryResponse struct {
	NumFound      int                      `json:"numFound"`
	Start         int                      `json:"start"`
	NumFoundExact bool                     `json:"numFoundExact"`
	Docs          []map[string]interface{} `json:"docs"`
}
