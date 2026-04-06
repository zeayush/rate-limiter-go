package limiter

import (
	"container/list"
	"context"
	"errors"
	"sync"
	"time"
)

// ─── Sliding Window Log ───────────────────────────────────────────────────────
//
// The sliding window log algorithm records the exact timestamp of every
// request in a sorted log. On each Allow call it evicts timestamps older than
// (now - window) and checks whether the remaining count is below the rate.
//
// Visualisation:
//
//	window = 1 min, rate = 5
//	time ──────────────────────────────────────────────→
//	         [t1][t2]   [t3][t4]  [t5]       [t6?]
//	         ←────────── 1 min ──────────────→
//	                                          ^ log has 5 entries → deny
//
// Properties:
//   - Exact: no edge-spike at window boundaries.
//   - Memory: O(rate) timestamps kept per key/instance.
//   - Not suitable for very high rates (>10 000/s) per key.
//
// Thread-safety: protected by an internal sync.Mutex.

// SlidingWindowLog is a rate limiter using the sliding-window-log algorithm.
type SlidingWindowLog struct {
	mu  sync.Mutex
	cfg Config
	log *list.List // ordered list of time.Time values (oldest at front)
}

// NewSlidingWindowLog creates a new SlidingWindowLog limiter.
//
// Rules:
//   - cfg.Rate must be > 0.
//   - cfg.Window must be > 0.
func NewSlidingWindowLog(cfg Config) (*SlidingWindowLog, error) {
	// TODO:
	//  1. Validate cfg.Rate and cfg.Window (same pattern as NewTokenBucket).
	//  2. Return &SlidingWindowLog{cfg: cfg, log: list.New()}, nil

	_ = list.New // hint
	return nil, errors.New("not implemented")
}

// evict removes all log entries older than (now - window).
// Must be called while holding sw.mu.
func (sw *SlidingWindowLog) evict(now time.Time) {
	// TODO:
	//  cutoff := now.Add(-sw.cfg.Window)
	//  for {
	//      front := sw.log.Front()
	//      if front == nil { break }
	//      if front.Value.(time.Time).After(cutoff) { break }
	//      sw.log.Remove(front)
	//  }
}

// Allow implements Limiter.
// It evicts old entries, checks the count, and appends the current timestamp if allowed.
func (sw *SlidingWindowLog) Allow(_ context.Context) (Result, error) {
	sw.mu.Lock()
	defer sw.mu.Unlock()

	// TODO step 1: now := time.Now(); sw.evict(now)

	// TODO step 2: count := int64(sw.log.Len())
	//   if count >= sw.cfg.Rate {
	//       // oldest entry in the log tells us when the next slot opens
	//       oldest := sw.log.Front().Value.(time.Time)
	//       retryAt := oldest.Add(sw.cfg.Window).Sub(now)
	//       return Result{
	//           Allowed:    false,
	//           Limit:      sw.cfg.Rate,
	//           Remaining:  0,
	//           Reset:      oldest.Add(sw.cfg.Window),
	//           RetryAfter: retryAt,
	//       }, nil
	//   }

	// TODO step 3: append now and return success
	//   sw.log.PushBack(now)
	//   return Result{
	//       Allowed:   true,
	//       Limit:     sw.cfg.Rate,
	//       Remaining: sw.cfg.Rate - int64(sw.log.Len()),
	//       Reset:     now.Add(sw.cfg.Window),
	//   }, nil

	return Result{}, errors.New("not implemented")
}
