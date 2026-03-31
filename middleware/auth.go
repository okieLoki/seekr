package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"os"
	"sync"
	"time"
)

const (
	sessionCookieName = "seekr_session"
	sessionTTL        = 24 * time.Hour
)

type session struct {
	createdAt time.Time
}

var (
	mu       sync.RWMutex
	sessions = make(map[string]session)
)

func getCredentials() (string, string) {
	username := os.Getenv("SEEKR_USERNAME")
	password := os.Getenv("SEEKR_PASSWORD")
	if username == "" {
		username = "admin"
	}
	if password == "" {
		password = "password"
	}
	return username, password
}

func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func isValidSession(r *http.Request) bool {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		return false
	}
	mu.RLock()
	sess, ok := sessions[cookie.Value]
	mu.RUnlock()
	if !ok {
		return false
	}
	if time.Since(sess.createdAt) > sessionTTL {
		mu.Lock()
		delete(sessions, cookie.Value)
		mu.Unlock()
		return false
	}
	return true
}

func HandleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	expectedUser, expectedPass := getCredentials()
	if body.Username != expectedUser || body.Password != expectedPass {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid username or password"})
		return
	}

	token, err := generateToken()
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	mu.Lock()
	sessions[token] = session{createdAt: time.Now()}
	mu.Unlock()

	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(sessionTTL.Seconds()),
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func HandleLogout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(sessionCookieName)
	if err == nil {
		mu.Lock()
		delete(sessions, cookie.Value)
		mu.Unlock()
	}
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "logged out"})
}

func Require(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		if path == "/api/login" {
			next.ServeHTTP(w, r)
			return
		}

		if isStaticAsset(path) {
			next.ServeHTTP(w, r)
			return
		}

		if !isValidSession(r) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"error": "Unauthorized"})
			return
		}

		next.ServeHTTP(w, r)
	})
}

func isStaticAsset(path string) bool {
	if path == "/" || path == "/index.html" {
		return true
	}
	for _, ext := range []string{".js", ".css", ".html", ".ico", ".png", ".svg", ".woff", ".woff2", ".ttf"} {
		if len(path) > len(ext) && path[len(path)-len(ext):] == ext {
			return true
		}
	}
	return false
}
