package handlers

import "go.uber.org/zap"

var Logger *zap.Logger
func SetLogger(l *zap.Logger) {
	Logger = l
}

// OpenLibraryResponse represents the response from OpenLibrary search API
type OpenLibraryResponse struct {
	NumFound      int                      `json:"numFound"`
	Start         int                      `json:"start"`
	NumFoundExact bool                     `json:"numFoundExact"`
	Docs          []map[string]interface{} `json:"docs"`
}

