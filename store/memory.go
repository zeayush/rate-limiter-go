package limiter

import (
	"context"
	"errors"
	"sync"
)

// ─── In-Memory Per-Key Limiter ────────────────────────────────────────────────
//
// MemoryStore implements KeyedLimiter using a plain Go map.
// Each key gets its own independent Limiter instance created by a factory func.
//
// Use this when:
//   - You have a single-instance deployment (no Redis).
//   - You want an automatic fallback when Redis is unavailable.
//
// Eviction: limiters are never evicted automatically. For long-running services
// with unbounded key spaces, call Prune() on a schedule to reclaim memory.
// (Prune is a stretch goal — see the TODO below.)
//
// Thread-safety: protected by an internal sync.RWMutex.

// LimiterFactory is a function that constructs a new Limiter for a given key.
// The key is provided so implementations can vary config per tenant if needed.
type LimiterFactory func(key string) (Limiter, error)

// MemoryStore is an in-memory KeyedLimiter.
type MemoryStore struct {
	mu       sync.RWMutex
	limiters map[string]Limiter
	factory  LimiterFactory
}

// NewMemoryStore creates a MemoryStore that uses factory to create per-key limiters.
//
// factory must not be nil.
func NewMemoryStore(factory LimiterFactory) (*MemoryStore, error) {
	// TODO:
	//  1. if factory == nil { return nil, errors.New("factory must not be nil") }
	//  2. Return &MemoryStore{
	//         limiters: make(map[string]Limiter),
	//         factory:  factory,
	//     }, nil

	_ = errors.New // hint
	return nil, errors.New("not implemented")
}

// getOrCreate returns the limiter for key, creating it via the factory if absent.
// It uses a double-checked lock pattern for efficiency:
//   - Read lock first to avoid lock contention on the hot path.
//   - Upgrade to write lock only when the key is missing.
func (m *MemoryStore) getOrCreate(key string) (Limiter, error) {
	// TODO — double-checked locking:
	//
	//  Phase 1 (read lock — fast path for existing keys):
	//    m.mu.RLock()
	//    if l, ok := m.limiters[key]; ok {
	//        m.mu.RUnlock()
	//        return l, nil
	//    }
	//    m.mu.RUnlock()
	//
	//  Phase 2 (write lock — slow path for new keys):
	//    m.mu.Lock()
	//    defer m.mu.Unlock()
	//    // Re-check after acquiring write lock (another goroutine may have created it).
	//    if l, ok := m.limiters[key]; ok {
	//        return l, nil
	//    }
	//    l, err := m.factory(key)
	//    if err != nil { return nil, err }
	//    m.limiters[key] = l
	//    return l, nil

	return nil, errors.New("not implemented")
}

// Allow implements KeyedLimiter.
func (m *MemoryStore) Allow(ctx context.Context, key string) (Result, error) {
	// TODO:
	//  l, err := m.getOrCreate(key)
	//  if err != nil { return Result{}, err }
	//  return l.Allow(ctx)
	return Result{}, errors.New("not implemented")
}

// Len returns the number of tracked keys (useful in tests and metrics).
func (m *MemoryStore) Len() int {
	// TODO:
	//  m.mu.RLock()
	//  defer m.mu.RUnlock()
	//  return len(m.limiters)
	return 0
}
