package routes

import (
	"log/slog"
	"net/http"

	"seekr/controllers"
	"seekr/docs"
	"seekr/ui"

	httpSwagger "github.com/swaggo/http-swagger"
)

func SetupRouter(c *controllers.SearchController) *http.ServeMux {
	router := http.NewServeMux()

	router.HandleFunc("/index", c.HandleIndex)
	router.HandleFunc("/bulk-index", c.HandleBulkIndex)
	router.HandleFunc("/api/documents", c.HandleDocuments)
	router.HandleFunc("/api/documents/update", c.HandleUpdate)

	router.HandleFunc("/search", c.HandleSearch)

	router.HandleFunc("/api/stats", c.HandleStats)

	router.HandleFunc("/api/collections", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			c.HandleListCollections(w, r)
		case http.MethodPost:
			c.HandleCreateCollection(w, r)
		case http.MethodDelete:
			c.HandleDeleteCollection(w, r)
		default:
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
	})

	router.HandleFunc("/swagger.yaml", docs.Handler)
	router.Handle("/swagger/", httpSwagger.Handler(httpSwagger.URL("/swagger.yaml")))

	router.Handle("/", http.FileServer(http.FS(ui.Files)))

	wrapped := http.NewServeMux()
	wrapped.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				slog.Error("Panic recovered", "error", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()
		router.ServeHTTP(w, r)
	})
	return wrapped
}
