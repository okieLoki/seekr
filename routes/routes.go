package routes

import (
	"log/slog"
	"net/http"

	"seekr/controllers"
	"seekr/ui"
)

func SetupRouter(c *controllers.SearchController) *http.ServeMux {
	router := http.NewServeMux()

	router.HandleFunc("/index", c.HandleIndex)
	router.HandleFunc("/bulk-index", c.HandleBulkIndex)
	router.HandleFunc("/search", c.HandleSearch)
	router.HandleFunc("/api/stats", c.HandleStats)
	router.HandleFunc("/api/documents", c.HandleDocuments)
	router.HandleFunc("/api/documents/update", c.HandleUpdate)

	router.Handle("/", http.FileServer(http.FS(ui.Files)))

	// Wrap in panic recovery middleware
	wrappedRouter := http.NewServeMux()
	wrappedRouter.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				slog.Error("Panic recovered in HTTP handler", "error", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()
		router.ServeHTTP(w, r)
	})

	return wrappedRouter
}
