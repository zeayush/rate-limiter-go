package limiter

import (
	"context"
	"errors"
	"sync"
	"time"
)

// ─── Fixed Window Counter ─────────────────────────────────────────────────────
//
// The fixed window algorithm divides time into equal-sized buckets aligned to
// the clock (e.g. the current minute or hour). A counter increments for each
// request; when the counter exceeds the rate the request is denied. At the
// start of the next window the counter resets to 0.
//
// Visualisation:
//
//	window = 1 min, rate = 5
//	│  :00–:59  │  :00–:59  │  :00–:59  │
//	│  ● ● ● ●  │ ● ● ● ● ●│  ● ● ✗ ✗  │
//	counter=4      =5(full)    =2 (new window resets)
//
// Known limitation: two bursts of `rate` requests can occur back-to-back
// at the window boundary (last second of window N + first second of window N+1).
//
// Properties:
//   - Memory: O(1) — one counter and one timestamp per instance.
//   - Best for: API quotas where small boundary spikes are acceptable.
//
// Thread-safety: protected by an internal sync.Mutex.

// FixedWindow is a rate limiter using the fixed-window-counter algorithm.
type FixedWindow struct {
	mu          sync.Mutex
	cfg         Config
	count       int64     // requests in the current window
	windowStart time.Time // when the current window began
}

// NewFixedWindow creates a new FixedWindow limiter.
//
// Rules:
//   - cfg.Rate must be > 0.
//   - cfg.Window must be > 0.
//
// The window starts immediately on construction.
func NewFixedWindow(cfg Config) (*FixedWindow, error) {
	// TODO:
	//  1. Validate cfg.Rate and cfg.Window.
	//  2. Return &FixedWindow{
	//         cfg:         cfg,
	//         windowStart: time.Now(),
	//     }, nil

	return nil, errors.New("not implemented")
}

// resetIfExpired checks whether the current window has expired and, if so,
// resets the counter and records the new window start.
// Must be called while holding fw.mu.
func (fw *FixedWindow) resetIfExpired(now time.Time) {
	// TODO:
	//  if now.Sub(fw.windowStart) >= fw.cfg.Window {
	//      fw.count = 0
	//      fw.windowStart = now
	//  }
}

// Allow implements Limiter.
// It resets the window if expired, then checks and increments the counter.
func (fw *FixedWindow) Allow(_ context.Context) (Result, error) {
	fw.mu.Lock()
	defer fw.mu.Unlock()

	// TODO step 1: now := time.Now(); fw.resetIfExpired(now)

	// TODO step 2: check the counter
	//   windowEnd := fw.windowStart.Add(fw.cfg.Window)
	//   if fw.count >= fw.cfg.Rate {
	//       return Result{
	//           Allowed:    false,
	//           Limit:      fw.cfg.Rate,
	//           Remaining:  0,
	//           Reset:      windowEnd,
	//           RetryAfter: windowEnd.Sub(now),
	//       }, nil
	//   }

	// TODO step 3: increment counter and return success
	//   fw.count++
	//   return Result{
	//       Allowed:   true,
	//       Limit:     fw.cfg.Rate,
	//       Remaining: fw.cfg.Rate - fw.count,
	//       Reset:     windowEnd,
	//   }, nil

	return Result{}, errors.New("not implemented")
}

// windowEnd returns the time at which the current window closes.
// Must be called while holding fw.mu.
func (fw *FixedWindow) windowEnd() time.Time {
	// TODO: return fw.windowStart.Add(fw.cfg.Window)
	return time.Time{}
}
