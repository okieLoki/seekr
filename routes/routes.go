package routes

import (
	"log/slog"
	"net/http"

	"seekr/controllers"
)

func SetupRouter(c *controllers.SearchController) *http.ServeMux {
	router := http.NewServeMux()

	router.HandleFunc("/index", c.HandleIndex)
	router.HandleFunc("/search", c.HandleSearch)

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
