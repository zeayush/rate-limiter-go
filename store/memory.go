package store

import (
	"context"
	"errors"
	"sync"

	"rate-limiter-go/limiter"
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
//
// Thread-safety: protected by an internal sync.RWMutex.

// MemoryStore is an in-memory KeyedLimiter.
type MemoryStore struct {
	mu       sync.RWMutex
	limiters map[string]limiter.Limiter
	factory  limiter.LimiterFactory
}

// NewMemoryStore creates a MemoryStore that uses factory to create per-key limiters.
//
// factory must not be nil.
func NewMemoryStore(factory limiter.LimiterFactory) (*MemoryStore, error) {
	if factory == nil {
		return nil, errors.New("factory must not be nil")
	}
	return &MemoryStore{
		limiters: make(map[string]limiter.Limiter),
		factory:  factory,
	}, nil
}

// getOrCreate returns the limiter for key, creating it via the factory if absent.
// Uses a double-checked lock pattern for efficiency.
func (m *MemoryStore) getOrCreate(key string) (limiter.Limiter, error) {
	m.mu.RLock()
	if l, ok := m.limiters[key]; ok {
		m.mu.RUnlock()
		return l, nil
	}
	m.mu.RUnlock()

	m.mu.Lock()
	defer m.mu.Unlock()
	if l, ok := m.limiters[key]; ok {
		return l, nil
	}
	l, err := m.factory(key)
	if err != nil {
		return nil, err
	}
	m.limiters[key] = l
	return l, nil
}

// Allow implements limiter.KeyedLimiter.
func (m *MemoryStore) Allow(ctx context.Context, key string) (limiter.Result, error) {
	l, err := m.getOrCreate(key)
	if err != nil {
		return limiter.Result{}, err
	}
	return l.Allow(ctx)
}

// Len returns the number of tracked keys (useful in tests and metrics).
func (m *MemoryStore) Len() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.limiters)
}

