# Custom Search Service

A middleware service that acts as a proxy between clients and the OpenLibrary API, featuring rate limiting and caching to optimize API usage.

## Features

- Fast HTTP server built with Gin
- Request caching from UI and using Redis to reduce API calls
- Rate limiting to protect the OpenLibrary API (not yet implemented)
- Graceful shutdown
- Learning how Zap logging works

## Getting Started

### Installation

1. Clone the repository
2. Install dependencies:
```bash
go mod download
```

3. Create a `.env` file (optional):
```bash
# Server Configuration
PORT=8080
ENV=development

# OpenLibrary API Configuration
OPENLIBRARY_API_URL=https://openlibrary.org/search.json
OPENLIBRARY_RATE_LIMIT=50

# Cache Configuration
CACHE_TTL_MINUTES=30
CACHE_MAX_SIZE=1000

# Redis Configuration
REDIS_ENABLED=false
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=
REDIS_DB=0

# Rate Limiting
RATE_LIMIT_REQUESTS_PER_MINUTE=30
```

### Running the Server

```bash
# Run the server
go run cmd/myapp/main.go

# Or build and run
go build -o bin/search-service cmd/myapp/main.go
./bin/search-service
```

## API Endpoints

### Health Check

```bash
GET /health
```

**Response:**
```json
{
  "status": "healthy",
  "service": "custom-search-service",
  "time": "2026-01-17T12:00:00Z"
}
```

### Search Books

```bash
GET /api/v1/search?q=lord+of+the+rings
```

**Query Parameters:**
- `q` (required): Search query string

**Response:**
```json
{
  "message": "Search functionality coming soon",
  "query": "lord of the rings",
  "results": []
}
```

## Testing

Test the server with curl:

```bash
# Health check
curl http://localhost:8080/health

# Search endpoint
curl "http://localhost:8080/api/v1/search?q=harry+potter"
```

## Next Steps

1. Basic HTTP server setup
2. Implement OpenLibrary API client
3. Add in-memory caching layer
4. Implement rate limiting (?)
5. Add Redis support for distributed caching
6. Write unit tests
7. Add Docker support (?)

## Development


# Run tests
go test ./...

# Format code
go fmt ./...

# Lint
golangci-lint run
```
