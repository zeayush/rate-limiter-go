package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"rate-limiter-go/limiter"
)

// ─── Test doubles ─────────────────────────────────────────────────────────────

// stubKeyedLimiter always returns a preset Result.
type stubKeyedLimiter struct {
	result limiter.Result
	err    error
}

func (s *stubKeyedLimiter) Allow(_ context.Context, _ string) (limiter.Result, error) {
	return s.result, s.err
}

// okHandler is a simple handler that writes 200 OK.
var okHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
})

// ─── IPExtractor Tests ────────────────────────────────────────────────────────

func TestIPExtractor_UsesXForwardedFor(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("X-Forwarded-For", "10.0.0.1, 10.0.0.2")

	key := IPExtractor(r)
	if key != "10.0.0.1" {
		t.Errorf("expected 10.0.0.1, got %q", key)
	}
}

func TestIPExtractor_FallsBackToRemoteAddr(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = "192.168.1.5:1234"

	key := IPExtractor(r)
	// Should strip port and return just the IP.
	if key != "192.168.1.5" {
		t.Errorf("expected 192.168.1.5, got %q", key)
	}
}

func TestHeaderExtractor_UsesHeader(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("X-API-Key", "secret-token")

	key := HeaderExtractor("X-API-Key")(r)
	if key != "secret-token" {
		t.Errorf("expected 'secret-token', got %q", key)
	}
}

func TestHeaderExtractor_FallsBackToIP(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = "10.1.2.3:5000"
	// X-API-Key is absent — should fall back to IP.

	extract := HeaderExtractor("X-API-Key")
	key := extract(r)
	if key == "" {
		t.Error("expected non-empty fallback key")
	}
}

// ─── HTTPMiddleware Tests ─────────────────────────────────────────────────────

func TestHTTPMiddleware_SetsHeadersOnAllow(t *testing.T) {
	kl := &stubKeyedLimiter{result: limiter.Result{
		Allowed:   true,
		Limit:     100,
		Remaining: 99,
		Reset:     time.Now().Add(time.Minute),
	}}

	handler := HTTPMiddleware(kl, IPExtractor)(okHandler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "1.2.3.4:0"
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	if rr.Header().Get("X-RateLimit-Limit") == "" {
		t.Error("X-RateLimit-Limit header missing")
	}
	if rr.Header().Get("X-RateLimit-Remaining") == "" {
		t.Error("X-RateLimit-Remaining header missing")
	}
	if rr.Header().Get("X-RateLimit-Reset") == "" {
		t.Error("X-RateLimit-Reset header missing")
	}
}

func TestHTTPMiddleware_Returns429WhenDenied(t *testing.T) {
	kl := &stubKeyedLimiter{result: limiter.Result{
		Allowed:    false,
		Limit:      10,
		Remaining:  0,
		Reset:      time.Now().Add(time.Second),
		RetryAfter: time.Second,
	}}

	handler := HTTPMiddleware(kl, IPExtractor)(okHandler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "1.2.3.4:0"
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusTooManyRequests {
		t.Errorf("expected 429, got %d", rr.Code)
	}
	if rr.Header().Get("Retry-After") == "" {
		t.Error("Retry-After header missing on 429 response")
	}
}

func TestHTTPMiddleware_FailOpenOnError(t *testing.T) {
	// When the limiter returns an error, the middleware should allow the request
	// rather than block traffic.
	kl := &stubKeyedLimiter{err: http.ErrNoCookie}

	handler := HTTPMiddleware(kl, IPExtractor)(okHandler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "1.2.3.4:0"
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected fail-open (200), got %d", rr.Code)
	}
}
