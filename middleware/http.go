// Package middleware provides rate-limit middleware for net/http and Gin.
//
// Every response — allowed or denied — gets three headers:
//
//	X-RateLimit-Limit     : the configured rate (requests per window)
//	X-RateLimit-Remaining : how many tokens/slots remain in the current window
//	X-RateLimit-Reset     : Unix timestamp (seconds) when the window resets
//
// When a request is denied the middleware additionally sets:
//
//	Retry-After : seconds to wait before retrying (RFC 6585)
//
// and responds with HTTP 429 Too Many Requests.
package middleware

import (
	"net"
	"net/http"
	"strconv"
	"strings"

	"rate-limiter-go/limiter"
)

// KeyExtractor extracts the rate-limit key from an HTTP request.
// Common implementations: IPExtractor, HeaderExtractor, PathExtractor.
type KeyExtractor func(r *http.Request) string

// ─── Key extractor helpers ────────────────────────────────────────────────────

// IPExtractor returns the client's IP address as the rate-limit key.
//
// It prefers the X-Forwarded-For header (set by reverse proxies) and falls
// back to r.RemoteAddr.
//
// TODO:
//
//	func IPExtractor(r *http.Request) string {
//	    if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
//	        // X-Forwarded-For may be "client, proxy1, proxy2" — take the first.
//	        if idx := strings.Index(xff, ","); idx != -1 {
//	            return strings.TrimSpace(xff[:idx])
//	        }
//	        return strings.TrimSpace(xff)
//	    }
//	    // RemoteAddr is "host:port" — strip the port.
//	    host, _, err := net.SplitHostPort(r.RemoteAddr)
//	    if err != nil { return r.RemoteAddr }
//	    return host
//	}
func IPExtractor(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if idx := strings.Index(xff, ","); idx != -1 {
			return strings.TrimSpace(xff[:idx])
		}
		return strings.TrimSpace(xff)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// HeaderExtractor returns a KeyExtractor that uses the value of a specific
// HTTP header (e.g. "X-API-Key") as the rate-limit key.
//
// TODO:
//
//	func HeaderExtractor(header string) KeyExtractor {
//	    return func(r *http.Request) string {
//	        if v := r.Header.Get(header); v != "" {
//	            return v
//	        }
//	        return IPExtractor(r) // fallback to IP when header is absent
//	    }
//	}
func HeaderExtractor(header string) KeyExtractor {
	return func(r *http.Request) string {
		if v := r.Header.Get(header); v != "" {
			return v
		}
		return IPExtractor(r)
	}
}

// ─── writeHeaders ─────────────────────────────────────────────────────────────

// writeHeaders sets the three standard rate-limit response headers.
// Call this for every response (allowed and denied).
func writeHeaders(w http.ResponseWriter, res limiter.Result) {
	w.Header().Set("X-RateLimit-Limit", strconv.FormatInt(res.Limit, 10))
	w.Header().Set("X-RateLimit-Remaining", strconv.FormatInt(res.Remaining, 10))
	w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(res.Reset.Unix(), 10))
	if !res.Allowed {
		retrySeconds := int64(res.RetryAfter.Seconds()) + 1
		w.Header().Set("Retry-After", strconv.FormatInt(retrySeconds, 10))
	}
}

// ─── net/http middleware ──────────────────────────────────────────────────────

// HTTPMiddleware returns a standard net/http middleware that rate-limits
// requests by the key returned by extractor.
//
// Usage:
//
//	store, _ := store.NewMemoryStore(factory)
//	mux.Handle("/api/", middleware.HTTPMiddleware(store, middleware.IPExtractor)(apiHandler))
func HTTPMiddleware(kl limiter.KeyedLimiter, extractor KeyExtractor) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := extractor(r)
			res, err := kl.Allow(r.Context(), key)
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}
			writeHeaders(w, res)
			if !res.Allowed {
				http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
