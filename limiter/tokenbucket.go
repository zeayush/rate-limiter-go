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
	// TODO:
	//  1. Validate: cfg.Rate <= 0  → return nil, errors.New("rate must be > 0")
	//  2. Validate: cfg.Window <= 0 → return nil, errors.New("window must be > 0")
	//  3. capacity := float64(cfg.Rate + cfg.Burst)
	//  4. rate := float64(cfg.Rate) / float64(cfg.Window) // tokens per nanosecond unit
	//  5. Return &TokenBucket{
	//         cfg:      cfg,
	//         tokens:   capacity,   // start full
	//         lastTime: time.Now(),
	//         capacity: capacity,
	//         rate:     rate,
	//     }, nil

	_ = errors.New // hint
	return nil, errors.New("not implemented")
}

// refill adds tokens proportional to elapsed time since the last call.
// Must be called while holding tb.mu.
func (tb *TokenBucket) refill() {
	// TODO:
	//  1. now := time.Now()
	//  2. elapsed := now.Sub(tb.lastTime)
	//  3. tb.tokens += tb.rate * float64(elapsed)
	//  4. if tb.tokens > tb.capacity { tb.tokens = tb.capacity }
	//  5. tb.lastTime = now
}

// Allow implements Limiter.
// It refills tokens based on elapsed time, then checks and consumes one token.
func (tb *TokenBucket) Allow(_ context.Context) (Result, error) {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	// TODO step 1: call tb.refill()

	// TODO step 2: check tokens
	//   if tb.tokens < 1 {
	//       timeToNextToken := time.Duration((1 - tb.tokens) / tb.rate)
	//       return Result{
	//           Allowed:    false,
	//           Limit:      tb.cfg.Rate,
	//           Remaining:  0,
	//           Reset:      time.Now().Add(timeToNextToken),
	//           RetryAfter: timeToNextToken,
	//       }, nil
	//   }

	// TODO step 3: consume one token and return success
	//   tb.tokens--
	//   return Result{
	//       Allowed:   true,
	//       Limit:     tb.cfg.Rate,
	//       Remaining: int64(tb.tokens),
	//       Reset:     time.Now().Add(tb.cfg.Window),
	//   }, nil

	return Result{}, errors.New("not implemented")
}
