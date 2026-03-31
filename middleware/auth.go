package middleware

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"crypto/pbkdf2"
)

const (
	sessionCookieName           = "seekr_session"
	defaultUsername             = "admin"
	defaultSessionTTL           = 24 * time.Hour
	defaultSessionIdleTTL       = 30 * time.Minute
	defaultLoginAttemptWindow   = 15 * time.Minute
	defaultLoginLockoutDuration = 15 * time.Minute
	defaultMaxLoginFailures     = 5
	pbkdf2Iterations            = 600000
	passwordSaltBytes           = 16
	passwordHashBytes           = 32
	maxJSONBodyBytes            = 1 << 20
)

type session struct {
	username  string
	createdAt time.Time
	lastSeen  time.Time
}

type loginAttempt struct {
	failures     int
	firstFailure time.Time
	blockedUntil time.Time
}

type passwordVerifier struct {
	iterations int
	salt       []byte
	hash       []byte
}

type authManager struct {
	mu sync.RWMutex

	username      string
	password      passwordVerifier
	secureCookies bool
	sessionTTL    time.Duration
	idleTTL       time.Duration
	maxFailures   int
	attemptWindow time.Duration
	lockoutWindow time.Duration
	now           func() time.Time

	sessions map[string]session
	attempts map[string]loginAttempt
}

var (
	authOnce    sync.Once
	authCurrent *authManager
	authInitErr error
)

func Init() error {
	_, err := getAuthManager()
	return err
}

func getAuthManager() (*authManager, error) {
	authOnce.Do(func() {
		authCurrent, authInitErr = newAuthManagerFromEnv()
	})
	return authCurrent, authInitErr
}

func newAuthManagerFromEnv() (*authManager, error) {
	username := strings.TrimSpace(os.Getenv("SEEKR_USERNAME"))
	if username == "" {
		username = defaultUsername
	}

	hashSpec := strings.TrimSpace(os.Getenv("SEEKR_PASSWORD_HASH"))
	plainPassword := os.Getenv("SEEKR_PASSWORD")

	var (
		verifier          passwordVerifier
		err               error
		bootstrapPassword string
	)

	switch {
	case hashSpec != "":
		verifier, err = parsePasswordHash(hashSpec)
		if err != nil {
			return nil, fmt.Errorf("invalid SEEKR_PASSWORD_HASH: %w", err)
		}
	case plainPassword != "":
		verifier, err = newPasswordVerifier(plainPassword)
		if err != nil {
			return nil, err
		}
		slog.Warn("Auth is using SEEKR_PASSWORD from environment; prefer SEEKR_PASSWORD_HASH for production deployments")
	default:
		bootstrapPassword, err = generateBootstrapPassword()
		if err != nil {
			return nil, err
		}
		verifier, err = newPasswordVerifier(bootstrapPassword)
		if err != nil {
			return nil, err
		}
	}

	manager := &authManager{
		username:      username,
		password:      verifier,
		secureCookies: envBool("SEEKR_SECURE_COOKIES", false),
		sessionTTL:    envDurationFromHours("SEEKR_SESSION_TTL_HOURS", defaultSessionTTL),
		idleTTL:       envDurationFromMinutes("SEEKR_SESSION_IDLE_MINUTES", defaultSessionIdleTTL),
		maxFailures:   envInt("SEEKR_LOGIN_MAX_FAILURES", defaultMaxLoginFailures),
		attemptWindow: envDurationFromMinutes("SEEKR_LOGIN_WINDOW_MINUTES", defaultLoginAttemptWindow),
		lockoutWindow: envDurationFromMinutes("SEEKR_LOGIN_LOCKOUT_MINUTES", defaultLoginLockoutDuration),
		now:           time.Now,
		sessions:      make(map[string]session),
		attempts:      make(map[string]loginAttempt),
	}

	if bootstrapPassword != "" {
		slog.Warn(
			"No auth credentials configured. Generated a temporary bootstrap password for this process.",
			"username", username,
			"password", bootstrapPassword,
		)
	}

	return manager, nil
}

func newPasswordVerifier(password string) (passwordVerifier, error) {
	if password == "" {
		return passwordVerifier{}, errors.New("password cannot be empty")
	}
	salt := make([]byte, passwordSaltBytes)
	if _, err := rand.Read(salt); err != nil {
		return passwordVerifier{}, fmt.Errorf("generate password salt: %w", err)
	}
	hash, err := pbkdf2.Key(sha256.New, password, salt, pbkdf2Iterations, passwordHashBytes)
	if err != nil {
		return passwordVerifier{}, fmt.Errorf("derive password hash: %w", err)
	}
	return passwordVerifier{
		iterations: pbkdf2Iterations,
		salt:       salt,
		hash:       hash,
	}, nil
}

func parsePasswordHash(spec string) (passwordVerifier, error) {
	parts := strings.Split(spec, "$")
	if len(parts) != 4 {
		return passwordVerifier{}, errors.New("expected format pbkdf2_sha256$<iterations>$<salt-base64>$<hash-base64>")
	}
	if parts[0] != "pbkdf2_sha256" {
		return passwordVerifier{}, fmt.Errorf("unsupported hash algorithm %q", parts[0])
	}
	iterations, err := strconv.Atoi(parts[1])
	if err != nil || iterations < 100000 {
		return passwordVerifier{}, errors.New("iterations must be an integer >= 100000")
	}
	salt, err := base64.StdEncoding.DecodeString(parts[2])
	if err != nil || len(salt) < 16 {
		return passwordVerifier{}, errors.New("salt must be valid base64 and at least 16 bytes")
	}
	hash, err := base64.StdEncoding.DecodeString(parts[3])
	if err != nil || len(hash) < 32 {
		return passwordVerifier{}, errors.New("hash must be valid base64 and at least 32 bytes")
	}
	return passwordVerifier{
		iterations: iterations,
		salt:       salt,
		hash:       hash,
	}, nil
}

func (p passwordVerifier) verify(password string) bool {
	derived, err := pbkdf2.Key(sha256.New, password, p.salt, p.iterations, len(p.hash))
	if err != nil {
		return false
	}
	return subtle.ConstantTimeCompare(derived, p.hash) == 1
}

func generateBootstrapPassword() (string, error) {
	buf := make([]byte, 18)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate bootstrap password: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func envBool(key string, fallback bool) bool {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	switch strings.ToLower(raw) {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
	}
}

func envInt(key string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	v, err := strconv.Atoi(raw)
	if err != nil || v <= 0 {
		return fallback
	}
	return v
}

func envDurationFromHours(key string, fallback time.Duration) time.Duration {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	hours, err := strconv.Atoi(raw)
	if err != nil || hours <= 0 {
		return fallback
	}
	return time.Duration(hours) * time.Hour
}

func envDurationFromMinutes(key string, fallback time.Duration) time.Duration {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	minutes, err := strconv.Atoi(raw)
	if err != nil || minutes <= 0 {
		return fallback
	}
	return time.Duration(minutes) * time.Minute
}

func remoteIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil && host != "" {
		return host
	}
	return r.RemoteAddr
}

func (a *authManager) attemptKey(r *http.Request, username string) string {
	return strings.ToLower(strings.TrimSpace(username)) + "|" + remoteIP(r)
}

func (a *authManager) allowLogin(key string) (bool, time.Duration) {
	now := a.now()

	a.mu.Lock()
	defer a.mu.Unlock()

	attempt, ok := a.attempts[key]
	if !ok {
		return true, 0
	}
	if attempt.blockedUntil.After(now) {
		return false, attempt.blockedUntil.Sub(now).Round(time.Second)
	}
	if !attempt.firstFailure.IsZero() && now.Sub(attempt.firstFailure) > a.attemptWindow {
		delete(a.attempts, key)
		return true, 0
	}
	return true, 0
}

func (a *authManager) recordFailure(key string) {
	now := a.now()

	a.mu.Lock()
	defer a.mu.Unlock()

	attempt := a.attempts[key]
	if attempt.firstFailure.IsZero() || now.Sub(attempt.firstFailure) > a.attemptWindow {
		attempt = loginAttempt{
			failures:     1,
			firstFailure: now,
		}
	} else {
		attempt.failures++
	}
	if attempt.failures >= a.maxFailures {
		attempt.failures = 0
		attempt.firstFailure = now
		attempt.blockedUntil = now.Add(a.lockoutWindow)
	}
	a.attempts[key] = attempt
}

func (a *authManager) clearFailures(key string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	delete(a.attempts, key)
}

func (a *authManager) loginAllowed(username string, password string, r *http.Request) (bool, string, int) {
	key := a.attemptKey(r, username)
	allowed, retryAfter := a.allowLogin(key)
	if !allowed {
		return false, fmt.Sprintf("Too many failed attempts. Try again in %s.", retryAfter), http.StatusTooManyRequests
	}

	if subtle.ConstantTimeCompare([]byte(username), []byte(a.username)) != 1 || !a.password.verify(password) {
		a.recordFailure(key)
		return false, "Invalid username or password", http.StatusUnauthorized
	}

	a.clearFailures(key)
	return true, "", http.StatusOK
}

func (a *authManager) issueSession(username string, secureCookie bool) (string, *http.Cookie, error) {
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", nil, err
	}
	token := hex.EncodeToString(tokenBytes)

	now := a.now()
	a.mu.Lock()
	a.sessions[token] = session{
		username:  username,
		createdAt: now,
		lastSeen:  now,
	}
	a.mu.Unlock()

	return token, &http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   int(a.sessionTTL.Seconds()),
		Secure:   secureCookie,
	}, nil
}

func tokenFromRequest(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if strings.HasPrefix(authHeader, "Bearer ") {
		return strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
	}
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		return ""
	}
	return cookie.Value
}

func (a *authManager) isValidSession(r *http.Request) bool {
	token := tokenFromRequest(r)
	if token == "" {
		return false
	}

	now := a.now()

	a.mu.Lock()
	defer a.mu.Unlock()

	sess, ok := a.sessions[token]
	if !ok {
		return false
	}
	if now.Sub(sess.createdAt) > a.sessionTTL || now.Sub(sess.lastSeen) > a.idleTTL {
		delete(a.sessions, token)
		return false
	}
	sess.lastSeen = now
	a.sessions[token] = sess
	return true
}

func (a *authManager) revokeSession(r *http.Request) {
	token := tokenFromRequest(r)
	if token == "" {
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	delete(a.sessions, token)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(payload)
}

func HandleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	manager, err := getAuthManager()
	if err != nil {
		slog.Error("Auth initialization failed", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxJSONBodyBytes)

	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	username := strings.TrimSpace(body.Username)
	ok, message, status := manager.loginAllowed(username, body.Password, r)
	if !ok {
		writeJSON(w, status, map[string]string{"error": message})
		return
	}

	secureCookie := manager.secureCookies || r.TLS != nil
	token, cookie, err := manager.issueSession(manager.username, secureCookie)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Cache-Control", "no-store")
	http.SetCookie(w, cookie)
	writeJSON(w, http.StatusOK, map[string]any{
		"status":             "ok",
		"token":              token,
		"expiresInSeconds":   int(manager.sessionTTL.Seconds()),
		"idleTimeoutSeconds": int(manager.idleTTL.Seconds()),
	})
}

func HandleLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	manager, err := getAuthManager()
	if err == nil {
		manager.revokeSession(r)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
		SameSite: http.SameSiteStrictMode,
	})

	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, map[string]string{"status": "logged out"})
}

func Require(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		if path == "/api/login" {
			next.ServeHTTP(w, r)
			return
		}

		if isStaticAsset(path) || strings.HasPrefix(path, "/swagger") {
			next.ServeHTTP(w, r)
			return
		}

		manager, err := getAuthManager()
		if err != nil {
			slog.Error("Auth initialization failed", "error", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		if manager.isValidSession(r) {
			next.ServeHTTP(w, r)
			return
		}

		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
	})
}

func isStaticAsset(path string) bool {
	if path == "/" || path == "/index.html" {
		return true
	}
	for _, ext := range []string{".js", ".css", ".html", ".ico", ".png", ".svg", ".woff", ".woff2", ".ttf"} {
		if strings.HasSuffix(path, ext) {
			return true
		}
	}
	return false
}
