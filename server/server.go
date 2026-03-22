package server

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"seekr/search"
)

type Server struct {
	engine *search.Engine
	router *http.ServeMux
}

func New(engine *search.Engine) *Server {
	s := &Server{
		engine: engine,
		router: http.NewServeMux(),
	}
	s.routes()
	return s
}

func (s *Server) routes() {
	s.router.HandleFunc("/index", s.handleIndex())
	s.router.HandleFunc("/search", s.handleSearch())
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Panic recovery middleware
	defer func() {
		if err := recover(); err != nil {
			slog.Error("Panic recovered in HTTP handler", "error", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
	}()

	s.router.ServeHTTP(w, r)
}

func (s *Server) handleIndex() http.HandlerFunc {
	type request struct {
		ID   int    `json:"id"`
		Text string `json:"text"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		err := s.engine.AddDocument(req.ID, req.Text)
		if err != nil {
			http.Error(w, err.Error(), http.StatusConflict) // Or 400 Bad Request
			return
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"status": "indexed"})
	}
}

func (s *Server) handleSearch() http.HandlerFunc {
	type response struct {
		Results []string `json:"results"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		query := r.URL.Query().Get("q")
		if query == "" {
			http.Error(w, "Missing 'q' query parameter", http.StatusBadRequest)
			return
		}

		results, err := s.engine.Search(query)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response{Results: results})
	}
}
