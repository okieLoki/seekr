package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

func resetAuthStateForTest() {
	authOnce = sync.Once{}
	authCurrent = nil
	authInitErr = nil
}

func initTestAuth(t *testing.T) *authManager {
	t.Helper()

	resetAuthStateForTest()
	t.Cleanup(resetAuthStateForTest)

	manager, err := getAuthManager()
	if err != nil {
		t.Fatalf("getAuthManager() error = %v", err)
	}
	return manager
}

func loginRequest(t *testing.T, username, password string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(http.MethodPost, "/api/login", strings.NewReader(`{"username":"`+username+`","password":"`+password+`"}`))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "127.0.0.1:12345"

	rr := httptest.NewRecorder()
	HandleLogin(rr, req)
	return rr
}

func TestHandleLoginAndRequire(t *testing.T) {
	t.Setenv("SEEKR_USERNAME", "admin")
	t.Setenv("SEEKR_PASSWORD", "correct horse battery staple")

	manager := initTestAuth(t)

	now := time.Now()
	manager.now = func() time.Time { return now }

	rr := loginRequest(t, "admin", "correct horse battery staple")
	if rr.Code != http.StatusOK {
		t.Fatalf("login status = %d, want %d", rr.Code, http.StatusOK)
	}

	var payload map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode login response: %v", err)
	}
	if payload["status"] != "ok" {
		t.Fatalf("login status payload = %v, want ok", payload["status"])
	}

	cookies := rr.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("expected session cookie to be set")
	}

	protected := Require(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/stats", nil)
	req.AddCookie(cookies[0])
	req.RemoteAddr = "127.0.0.1:12345"

	protectedRR := httptest.NewRecorder()
	protected.ServeHTTP(protectedRR, req)
	if protectedRR.Code != http.StatusOK {
		t.Fatalf("protected status = %d, want %d", protectedRR.Code, http.StatusOK)
	}
}

func TestLoginLockout(t *testing.T) {
	t.Setenv("SEEKR_USERNAME", "admin")
	t.Setenv("SEEKR_PASSWORD", "super-secret")

	manager := initTestAuth(t)

	current := time.Now()
	manager.now = func() time.Time { return current }

	for i := 0; i < defaultMaxLoginFailures; i++ {
		rr := loginRequest(t, "admin", "wrong-password")
		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("attempt %d status = %d, want %d", i+1, rr.Code, http.StatusUnauthorized)
		}
	}

	locked := loginRequest(t, "admin", "super-secret")
	if locked.Code != http.StatusTooManyRequests {
		t.Fatalf("lockout status = %d, want %d", locked.Code, http.StatusTooManyRequests)
	}

	current = current.Add(defaultLoginLockoutDuration + time.Second)

	retry := loginRequest(t, "admin", "super-secret")
	if retry.Code != http.StatusOK {
		t.Fatalf("post-lockout login status = %d, want %d", retry.Code, http.StatusOK)
	}
}

func TestSessionIdleTimeout(t *testing.T) {
	t.Setenv("SEEKR_USERNAME", "admin")
	t.Setenv("SEEKR_PASSWORD", "idle-secret")
	t.Setenv("SEEKR_SESSION_IDLE_MINUTES", "30")

	manager := initTestAuth(t)

	current := time.Now()
	manager.now = func() time.Time { return current }

	login := loginRequest(t, "admin", "idle-secret")
	if login.Code != http.StatusOK {
		t.Fatalf("login status = %d, want %d", login.Code, http.StatusOK)
	}

	cookies := login.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("expected session cookie to be set")
	}

	protected := Require(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	current = current.Add(defaultSessionIdleTTL + time.Second)

	req := httptest.NewRequest(http.MethodGet, "/api/documents", nil)
	req.AddCookie(cookies[0])
	req.RemoteAddr = "127.0.0.1:12345"

	rr := httptest.NewRecorder()
	protected.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expired session status = %d, want %d", rr.Code, http.StatusUnauthorized)
	}
}
