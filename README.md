# Seekr

Seekr is a self-hosted document search app built in Go. It stores documents in a local `bbolt` database, indexes them with BM25-style ranking and fuzzy token matching, and ships with an authenticated dashboard for collections, search, bulk imports, and document editing.

## Highlights

- **BM25 + fuzzy search** with stemming and stop-word filtering
- **Persistent storage** in a single local `seekr.db` file
- **Collections** for separating datasets inside one database
- **Authenticated dashboard** with login, stats, document browsing, editing, and bulk import
- **Async bulk imports** with progress updates over SSE and in-app notifications
- **JSON-aware indexing** where JSON values are extracted for indexing while raw payloads are preserved
- **Swagger docs** served by the app at `http://localhost:8080/swagger/`

## Quick Start

```bash
git clone https://github.com/okieLoki/seekr.git
cd seekr
go run main.go
```

Open `http://localhost:8080` in your browser.

## Authentication

Seekr requires login for API and dashboard access except static assets and Swagger.

Configure credentials with:

```bash
SEEKR_USERNAME=admin
SEEKR_PASSWORD_HASH=pbkdf2_sha256$600000$<salt-base64>$<hash-base64>
```

You can also use plaintext `SEEKR_PASSWORD`, but `SEEKR_PASSWORD_HASH` is preferred for real deployments.

If no password is configured, Seekr generates a temporary bootstrap password at startup and prints it to the server logs.

Useful auth settings:

```bash
SEEKR_SESSION_TTL_HOURS=24
SEEKR_SESSION_IDLE_MINUTES=30
SEEKR_LOGIN_MAX_FAILURES=5
SEEKR_LOGIN_WINDOW_MINUTES=15
SEEKR_LOGIN_LOCKOUT_MINUTES=15
SEEKR_SECURE_COOKIES=false
```

## Environment

An example config is included in [.env.example](./.env.example).

Seekr loads `.env` at startup if present.

## API Overview

Interactive docs are available at `http://localhost:8080/swagger/` once the server is running.

All authenticated endpoints accept the session cookie set by `/api/login`. The middleware also accepts `Authorization: Bearer <token>`.

### `POST /api/login`

```bash
curl -X POST http://localhost:8080/api/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"your-password"}'
```

### `POST /index?collection=<name>`

Index a single document.

```bash
curl -X POST "http://localhost:8080/index?collection=movies" \
  -H "Content-Type: application/json" \
  -d '{"id":"movie-1","text":"{\"Title\":\"Interstellar\",\"Director\":\"Christopher Nolan\"}"}'
```

### `POST /bulk-index?collection=<name>`

Synchronous bulk indexing endpoint. Still useful for scripts, but the dashboard now prefers async imports.

```bash
curl -X POST "http://localhost:8080/bulk-index?collection=movies" \
  -H "Content-Type: application/json" \
  -d '[
    {"text":"plain text document"},
    {"id":"custom-id","text":"explicit id and text"},
    {"Title":"Avatar","Director":"James Cameron"}
  ]'
```

### `POST /api/imports?collection=<name>`

Create an async bulk import job.

```bash
curl -X POST "http://localhost:8080/api/imports?collection=movies" \
  -H "Content-Type: application/json" \
  -d '[
    {"Title":"Interstellar","Director":"Christopher Nolan"},
    {"Title":"Oppenheimer","Director":"Christopher Nolan"}
  ]'
```

Response:

```json
{
  "job": {
    "id": "job-id",
    "collection": "movies",
    "status": "queued",
    "total": 2,
    "processed": 0,
    "indexed": 0,
    "skipped": 0
  }
}
```

### `GET /api/imports?collection=<name>`

List recent import jobs for a collection.

### `GET /api/imports/events?collection=<name>`

Server-Sent Events stream for import progress.

### `POST /search?collection=<name>`

Search is JSON POST-based in the current app.

```bash
curl -X POST "http://localhost:8080/search?collection=movies" \
  -H "Content-Type: application/json" \
  -d '{"q":"christopher nolan","boosts":{"Title":2.0}}'
```

### `GET /api/documents?collection=<name>&page=1&limit=20`

Paginated document listing.

### `PUT /api/documents/update?collection=<name>&id=<id>`

Update a stored document and reindex it atomically.

### `GET /api/stats?collection=<name>`

Collection-level stats.

### `GET|POST|DELETE /api/collections`

- `GET /api/collections` lists collections
- `POST /api/collections` creates one
- `DELETE /api/collections?name=<name>` deletes one

## Dashboard

The embedded UI includes:

- authenticated sign-in
- collection switching and management
- single-document indexing
- bulk JSON import by paste or `.json` file
- async import progress panel with dismiss support
- paginated browsing
- inline JSON viewing with syntax coloring
- field-boosted search
- document editing

## Search and Indexing Notes

- Seekr extracts JSON values for indexing when a document body is valid JSON
- raw document text is preserved exactly as stored
- query matching uses tokenization, stop-word filtering, stemming, and fuzzy posting lookup
- field boosts are applied only when the stored document body is valid JSON and the boosted field is a string

## Project Layout

```text
seekr/
├── main.go              # Entry point and server bootstrap
├── routes/              # Router wiring
├── controllers/         # HTTP handlers, including async import jobs
├── services/            # Search engine logic
├── db/                  # bbolt persistence layer
├── middleware/          # Auth and request protection
├── parser/              # Text extraction helpers
├── tokenizer/           # Tokenization, stemming, stop-word filtering
├── types/               # Shared request/response types
├── ui/                  # Embedded dashboard assets
├── docs/                # Swagger output
└── tests/               # Integration and unit tests
```

## Running Tests

```bash
go test ./...
```

## Regenerating Swagger

```bash
go install github.com/swaggo/swag/cmd/swag@latest
swag init --parseDependency --parseInternal
```

## Tech Stack

| Component | Library |
|-----------|---------|
| HTTP server | `net/http` |
| Storage | `go.etcd.io/bbolt` |
| Stemming | `github.com/kljensen/snowball` |
| Auth hashing | `crypto/pbkdf2` |
| Swagger UI | `github.com/swaggo/http-swagger` |

