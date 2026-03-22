package main

import (
	"log/slog"
	"net/http"
	"os"

	"seekr/search"
	"seekr/server"
)

func main() {
	// Setup structured logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	engine := search.NewEngine()

	// Pre-index a document for testing
	_ = engine.AddDocument(1, "Elasticsearch is a distributed, RESTful search and analytics engine.")
	_ = engine.AddDocument(2, "Building a search engine in Go is a fun and educational project.")

	srv := server.New(engine)

	slog.Info("Starting REST API search server on :8080")
	if err := http.ListenAndServe(":8080", srv); err != nil {
		slog.Error("Server failed", "error", err)
		os.Exit(1)
	}
}
