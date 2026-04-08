package limiter

import (
	"context"
	"errors"
	"sync"
	"time"
)

// ─── Token Bucket ─────────────────────────────────────────────────────────────
//
// The token bucket algorithm allows short bursts while enforcing a long-term
// average rate.
//
// Visualisation:
//
//	capacity   ┌──────────────────┐
//	           │ ● ● ● ● ● ●      │  ← tokens accumulate at rate r/s
//	           └──────────────────┘
//	              ↑  each request takes one token
//
// Parameters:
//   - rate   : tokens added per window (e.g. 100 tokens/minute)
//   - window : the refill period (e.g. time.Minute)
//   - burst  : maximum bucket capacity (>= rate; allows short spikes)
//
// Refill happens lazily on each Allow call — no background goroutine needed.
//
// Thread-safety: protected by an internal sync.Mutex.

// TokenBucket is a rate limiter using the token-bucket algorithm.
type TokenBucket struct {
	mu       sync.Mutex
	cfg      Config
	tokens   float64   // current token count (float to support fractional refill)
	lastTime time.Time // last time tokens were refilled
	capacity float64   // maximum tokens (cfg.Rate + cfg.Burst)
	rate     float64   // tokens per nanosecond
}

// NewTokenBucket creates a new TokenBucket limiter.
//
// Rules:
//   - cfg.Rate must be > 0.
//   - cfg.Window must be > 0.
//   - cfg.Burst >= 0; the bucket capacity is cfg.Rate + cfg.Burst.
//
// The bucket starts full (capacity tokens available).
func NewTokenBucket(cfg Config) (*TokenBucket, error) {
	if cfg.Rate <= 0 {
		return nil, errors.New("rate must be > 0")
	}
	if cfg.Window <= 0 {
		return nil, errors.New("window must be > 0")
	}
	capacity := float64(cfg.Rate + cfg.Burst)
	rate := float64(cfg.Rate) / float64(cfg.Window)
	return &TokenBucket{
		cfg:      cfg,
		tokens:   capacity,
		lastTime: time.Now(),
		capacity: capacity,
		rate:     rate,
	}, nil
}

// refill adds tokens proportional to elapsed time since the last call.
// Must be called while holding tb.mu.
func (tb *TokenBucket) refill() {
	now := time.Now()
	elapsed := now.Sub(tb.lastTime)
	tb.tokens += tb.rate * float64(elapsed)
	if tb.tokens > tb.capacity {
		tb.tokens = tb.capacity
	}
	tb.lastTime = now
}

// Allow implements Limiter.
// It refills tokens based on elapsed time, then checks and consumes one token.
func (tb *TokenBucket) Allow(_ context.Context) (Result, error) {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	tb.refill()

	if tb.tokens < 1 {
		timeToNextToken := time.Duration((1 - tb.tokens) / tb.rate)
		return Result{
			Allowed:    false,
			Limit:      tb.cfg.Rate,
			Remaining:  0,
			Reset:      time.Now().Add(timeToNextToken),
			RetryAfter: timeToNextToken,
		}, nil
	}

	tb.tokens--
	return Result{
		Allowed:   true,
		Limit:     tb.cfg.Rate,
		Remaining: int64(tb.tokens),
		Reset:     time.Now().Add(tb.cfg.Window),
	}, nil
}
