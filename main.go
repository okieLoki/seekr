package main

import (
	"bufio"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"seekr/controllers"
	"seekr/db"
	"seekr/middleware"
	"seekr/routes"
	"seekr/services"
)

// loadEnv reads a .env file and sets environment variables (does not override existing ones).
func loadEnv(filename string) {
	f, err := os.Open(filename)
	if err != nil {
		return // .env is optional
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		// Only set if not already set
		if os.Getenv(key) == "" {
			os.Setenv(key, val)
		}
	}
}

func main() {
	loadEnv(".env")

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	if err := middleware.Init(); err != nil {
		slog.Error("Auth failed to boot", "error", err)
		os.Exit(1)
	}

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
