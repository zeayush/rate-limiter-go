package store

import (
	"context"
	"testing"
	"time"

	"github.com/zeayush/rate-limiter-go/limiter"
)

// ─── MemoryStore Tests ────────────────────────────────────────────────────────

func fixedWindowFactory(rate int64, window time.Duration) limiter.LimiterFactory {
	return func(_ string) (limiter.Limiter, error) {
		return limiter.NewFixedWindow(limiter.Config{Rate: rate, Window: window})
	}
}

func TestNewMemoryStore_NilFactory(t *testing.T) {
	_, err := NewMemoryStore(nil)
	if err == nil {
		t.Fatal("expected error for nil factory, got nil")
	}
}

func TestMemoryStore_CreatesLimiterPerKey(t *testing.T) {
	ms, err := NewMemoryStore(fixedWindowFactory(10, time.Minute))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ctx := context.Background()
	ms.Allow(ctx, "user:1") //nolint:errcheck
	ms.Allow(ctx, "user:2") //nolint:errcheck

	if n := ms.Len(); n != 2 {
		t.Errorf("expected 2 tracked keys, got %d", n)
	}
}

func TestMemoryStore_IndependentCounters(t *testing.T) {
	// rate=1 so second call on same key is denied; different keys unaffected.
	ms, _ := NewMemoryStore(fixedWindowFactory(1, time.Minute))
	ctx := context.Background()

	res, _ := ms.Allow(ctx, "a")
	if !res.Allowed {
		t.Fatal("first request for 'a' should be allowed")
	}
	res, _ = ms.Allow(ctx, "a")
	if res.Allowed {
		t.Fatal("second request for 'a' should be denied")
	}

	// 'b' has its own independent counter — should still be allowed.
	res, _ = ms.Allow(ctx, "b")
	if !res.Allowed {
		t.Fatal("first request for 'b' should be allowed regardless of 'a'")
	}
}

func TestMemoryStore_ConcurrentAccess(t *testing.T) {
	ms, _ := NewMemoryStore(fixedWindowFactory(1000, time.Minute))
	ctx := context.Background()

	done := make(chan struct{})
	for i := 0; i < 50; i++ {
		go func() {
			defer func() { done <- struct{}{} }()
			for j := 0; j < 100; j++ {
				ms.Allow(ctx, "shared-key") //nolint:errcheck
			}
		}()
	}
	for i := 0; i < 50; i++ {
		<-done
	}
}

func TestMemoryStore_Len(t *testing.T) {
	ms, _ := NewMemoryStore(fixedWindowFactory(10, time.Minute))
	ctx := context.Background()

	if ms.Len() != 0 {
		t.Fatal("fresh store should have 0 keys")
	}
	ms.Allow(ctx, "x") //nolint:errcheck
	ms.Allow(ctx, "y") //nolint:errcheck
	ms.Allow(ctx, "x") //nolint:errcheck // same key again — no new entry

	if ms.Len() != 2 {
		t.Errorf("expected 2 keys, got %d", ms.Len())
	}
}
