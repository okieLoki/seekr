# Seekr

A BM25-powered full-text search engine with a persistent bbolt database, multi-format document ingestion, and a built-in dashboard UI.

## Features

- **BM25 + Fuzzy Search** — ranked full-text search with stemming, stop-word removal, and fuzzy token matching
- **Persistent Storage** — documents and the inverted index live in a single `seekr.db` bbolt file; data survives restarts
- **Multi-format Ingestion** — auto-detects format and extracts only human-readable text for indexing; raw payload is stored verbatim
- **REST API** — six JSON endpoints covering all CRUD and search operations
- **Dashboard UI** — MongoDB Compass-inspired web UI served at `http://localhost:8080`
- **Swagger Docs** — interactive API explorer at `http://localhost:8080/swagger/`

## Supported Document Formats

| Format | Detection | Indexed content |
|--------|-----------|-----------------|
| JSON | Starts with `{` or `[` | All string values (keys, numbers, booleans skipped) |
| HTML | Starts with `<` | All visible text nodes |
| XML | Starts with `<` (fallback) | All `CharData` text nodes |
| TOML | Has `key = value` lines | All string values |
| YAML | Has `key: value` lines | All string values |
| Plain text | Fallback | Passed through as-is |

## Quick Start

```bash
git clone https://github.com/okieLoki/seekr.git
cd seekr
go run main.go
```

Open `http://localhost:8080` in your browser.

## API Reference

Interactive docs are available at **`http://localhost:8080/swagger/`** once the server is running.

### `POST /index`
Index a single document.

```bash
curl -X POST http://localhost:8080/index \
  -H "Content-Type: application/json" \
  -d '{"id": "doc-1", "text": "The quick brown fox"}'
```

```bash
# JSON document — only values are indexed, keys are not
curl -X POST http://localhost:8080/index \
  -H "Content-Type: application/json" \
  -d '{"id": "movie-1", "text": "{\"Title\": \"Interstellar\", \"Director\": \"Christopher Nolan\"}"}'
```

**Response:** `201 Created`

---

### `POST /bulk-index`
Index multiple documents in one request. Each item may include optional `id` and `text` fields. If `id` is omitted a UUID v4 is auto-generated. If `text` is omitted the **entire object** is used as the document body.

```bash
curl -X POST http://localhost:8080/bulk-index \
  -H "Content-Type: application/json" \
  -d '[
    {"text": "plain text document"},
    {"id": "custom-id", "text": "explicit id and text"},
    {"Title": "Avatar", "Director": "James Cameron", "Plot": "A paraplegic marine..."}
  ]'
```

**Response:**
```json
{"indexed": 3, "skipped": 0}
```

---

### `GET /search?q=<query>`
Full-text BM25 search with fuzzy matching. Results are ranked by relevance score.

```bash
curl "http://localhost:8080/search?q=christopher+nolan"
```

**Response:**
```json
{
  "results": [
    {"id": "movie-1", "text": "{\"Title\": \"Interstellar\", ...}", "score": 3.42}
  ]
}
```

---

### `GET /api/documents?page=1&limit=20`
Paginated list of all stored documents.

```bash
curl "http://localhost:8080/api/documents?page=1&limit=10"
```

**Response:**
```json
{
  "documents": [{"id": "doc-1", "text": "The quick brown fox"}],
  "total": 1,
  "page": 1,
  "limit": 10
}
```

---

### `PUT /api/documents/update?id=<id>`
Update a document's content. The BM25 index is updated atomically.

```bash
curl -X PUT "http://localhost:8080/api/documents/update?id=doc-1" \
  -H "Content-Type: application/json" \
  -d '{"text": "The quick brown fox jumps over the lazy dog"}'
```

**Response:** `200 OK`

---

### `GET /api/stats`
Global database statistics.

```bash
curl http://localhost:8080/api/stats
```

**Response:**
```json
{"totalDocs": 6096, "totalLength": 673018}
```

## Architecture

```
seekr/
├── main.go              # Entry point, server bootstrap
├── routes/              # HTTP router and middleware
├── controllers/         # HTTP handlers (annotated for Swagger)
├── services/            # BM25 search engine
├── db/                  # bbolt persistence layer
├── tokenizer/           # Stemming, stop-words, tokenization
├── parser/              # Multi-format text extractor (JSON/YAML/TOML/XML/HTML)
├── index/               # Fuzzy index
├── types/               # Shared types
├── ui/                  # Embedded web dashboard (HTML/CSS/JS)
├── assets/              # Embedded stop-words and assets
├── docs/                # Auto-generated Swagger spec (swag init)
└── tests/               # Integration and unit tests
```

## Running Tests

```bash
go test -v ./tests/...
```

## Generating / Updating Swagger Docs

```bash
# Install swag CLI (one-time)
go install github.com/swaggo/swag/cmd/swag@latest

# Regenerate after changing handler annotations
swag init --parseDependency --parseInternal
```

## Tech Stack

| Component | Library |
|-----------|---------|
| HTTP server | `net/http` (stdlib) |
| Storage | `go.etcd.io/bbolt` |
| Stemming | `github.com/kljensen/snowball` |
| YAML parsing | `gopkg.in/yaml.v3` |
| TOML parsing | `github.com/BurntSushi/toml` |
| HTML parsing | `golang.org/x/net/html` |
| Swagger | `github.com/swaggo/swag` + `http-swagger` |
