// Package limiter provides rate limiting algorithms and shared types.
//
// Three algorithms are provided:
//
//   - TokenBucket  — smooth limiting with burst allowance
//   - SlidingWindowLog — precise per-request accuracy; higher memory cost
//   - FixedWindow  — lowest memory; slight boundary spikiness
//
// Each algorithm implements the [Limiter] interface.
// Per-key limiting is layered on top via [KeyedLimiter].
package limiter

import (
	"context"
	"time"
)

// Result is returned by every Allow call.
// It carries enough information to populate the standard rate-limit headers.
type Result struct {
	// Allowed is true when the request may proceed.
	Allowed bool

	// Limit is the maximum number of requests allowed in the window.
	Limit int64

	// Remaining is how many requests can still be made in the current window.
	// May be 0 even when Allowed is true (last token consumed).
	Remaining int64

	// Reset is the moment the window (or bucket) fully resets.
	// Format as Unix seconds for the X-RateLimit-Reset header.
	Reset time.Time

	// RetryAfter is meaningful only when Allowed is false.
	// It is the minimum duration the caller should wait before retrying.
	RetryAfter time.Duration
}

// Limiter is the common interface implemented by all three algorithms.
// Implementations must be safe for concurrent use.
type Limiter interface {
	// Allow checks whether one request token is available and, if so,
	// consumes it. ctx carries a deadline/timeout for Redis calls.
	Allow(ctx context.Context) (Result, error)
}

// KeyedLimiter wraps a factory function so that each unique key gets its
// own independent Limiter instance.
// Implementations must be safe for concurrent use.
type KeyedLimiter interface {
	// Allow checks the rate limit for the given key.
	Allow(ctx context.Context, key string) (Result, error)
}

// LimiterFactory creates a new Limiter for the given key.
// The key is provided so implementations can vary config per tenant if needed.
type LimiterFactory func(key string) (Limiter, error)

// Config holds the parameters common to all algorithms.
type Config struct {
	// Rate is the maximum number of requests allowed per Window.
	Rate int64

	// Window is the time period over which Rate applies.
	Window time.Duration

	// Burst is only used by TokenBucket; it sets the maximum number of
	// tokens that can accumulate above Rate. Zero means no extra burst.
	Burst int64
}
