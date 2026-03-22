// @title           Seekr API
// @version         1.0
// @description     A BM25-powered full-text search engine with persistent bbolt storage. Supports plain text, JSON, YAML, TOML, XML, and HTML document ingestion.
// @contact.name    Seekr
// @license.name    MIT
// @host            localhost:8080
// @BasePath        /
package main

import (
	"log/slog"
	"net/http"
	"os"

	"seekr/controllers"
	"seekr/db"
	"seekr/routes"
	"seekr/services"
)

func main() {
	// Setup structured logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	store, err := db.NewStore("seekr.db")
	if err != nil {
		slog.Error("Database failed to boot", "error", err)
		os.Exit(1)
	}
	defer store.Close()

	engine := services.NewEngine(store)
	controller := controllers.NewSearchController(engine)
	router := routes.SetupRouter(controller)

	slog.Info("Starting REST API search server on :8080")
	if err := http.ListenAndServe(":8080", router); err != nil {
		slog.Error("Server failed", "error", err)
		os.Exit(1)
	}
}
