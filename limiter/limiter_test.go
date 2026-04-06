package limiter

import (
	"context"
	"testing"
	"time"
)

// ─── Token Bucket Tests ───────────────────────────────────────────────────────

func TestNewTokenBucket_InvalidRate(t *testing.T) {
	_, err := NewTokenBucket(Config{Rate: 0, Window: time.Second})
	if err == nil {
		t.Fatal("expected error for zero rate, got nil")
	}
}

func TestNewTokenBucket_InvalidWindow(t *testing.T) {
	_, err := NewTokenBucket(Config{Rate: 10, Window: 0})
	if err == nil {
		t.Fatal("expected error for zero window, got nil")
	}
}

func TestTokenBucket_AllowsUpToRate(t *testing.T) {
	const rate = 5
	tb, err := NewTokenBucket(Config{Rate: rate, Window: time.Second})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ctx := context.Background()
	for i := 0; i < rate; i++ {
		res, err := tb.Allow(ctx)
		if err != nil {
			t.Fatalf("request %d: unexpected error: %v", i, err)
		}
		if !res.Allowed {
			t.Fatalf("request %d should be allowed (bucket had tokens)", i)
		}
	}
}

func TestTokenBucket_DeniesWhenExhausted(t *testing.T) {
	tb, _ := NewTokenBucket(Config{Rate: 3, Window: time.Minute})
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		tb.Allow(ctx) //nolint:errcheck
	}

	res, err := tb.Allow(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Allowed {
		t.Fatal("4th request should be denied but was allowed")
	}
	if res.RetryAfter <= 0 {
		t.Error("denied result should have positive RetryAfter")
	}
}

func TestTokenBucket_BurstAllowsExtra(t *testing.T) {
	// rate=3/min, burst=2 → capacity=5
	tb, _ := NewTokenBucket(Config{Rate: 3, Burst: 2, Window: time.Minute})
	ctx := context.Background()

	allowed := 0
	for i := 0; i < 10; i++ {
		res, _ := tb.Allow(ctx)
		if res.Allowed {
			allowed++
		}
	}
	if allowed != 5 {
		t.Errorf("expected 5 allowed (rate+burst), got %d", allowed)
	}
}

func TestTokenBucket_HeaderFields(t *testing.T) {
	tb, _ := NewTokenBucket(Config{Rate: 10, Window: time.Second})
	res, err := tb.Allow(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if res.Limit != 10 {
		t.Errorf("expected Limit=10, got %d", res.Limit)
	}
	if res.Reset.IsZero() {
		t.Error("Reset must not be zero")
	}
}

func TestTokenBucket_RefillOverTime(t *testing.T) {
	// Start with 1 token/100ms, exhaust it, then wait for refill.
	tb, _ := NewTokenBucket(Config{Rate: 1, Window: 100 * time.Millisecond})
	ctx := context.Background()

	res, _ := tb.Allow(ctx)
	if !res.Allowed {
		t.Fatal("first request should be allowed")
	}
	res, _ = tb.Allow(ctx)
	if res.Allowed {
		t.Fatal("second request should be denied (bucket empty)")
	}

	time.Sleep(120 * time.Millisecond) // wait for refill

	res, _ = tb.Allow(ctx)
	if !res.Allowed {
		t.Fatal("request after refill should be allowed")
	}
}

// ─── Sliding Window Log Tests ─────────────────────────────────────────────────

func TestNewSlidingWindowLog_InvalidRate(t *testing.T) {
	_, err := NewSlidingWindowLog(Config{Rate: 0, Window: time.Second})
	if err == nil {
		t.Fatal("expected error for zero rate")
	}
}

func TestSlidingWindowLog_AllowsUpToRate(t *testing.T) {
	const rate = 5
	sw, _ := NewSlidingWindowLog(Config{Rate: rate, Window: time.Second})
	ctx := context.Background()

	for i := 0; i < rate; i++ {
		res, _ := sw.Allow(ctx)
		if !res.Allowed {
			t.Fatalf("request %d should be allowed", i)
		}
	}
}

func TestSlidingWindowLog_DeniesWhenFull(t *testing.T) {
	sw, _ := NewSlidingWindowLog(Config{Rate: 3, Window: time.Second})
	ctx := context.Background()
	for i := 0; i < 3; i++ {
		sw.Allow(ctx) //nolint:errcheck
	}

	res, _ := sw.Allow(ctx)
	if res.Allowed {
		t.Fatal("4th request should be denied")
	}
}

func TestSlidingWindowLog_RemainderDecreases(t *testing.T) {
	sw, _ := NewSlidingWindowLog(Config{Rate: 5, Window: time.Second})
	ctx := context.Background()

	prev := int64(5)
	for i := 0; i < 5; i++ {
		res, _ := sw.Allow(ctx)
		if !res.Allowed {
			t.Fatalf("request %d should be allowed", i)
		}
		if res.Remaining >= prev {
			t.Errorf("remaining should decrease: was %d, now %d", prev, res.Remaining)
		}
		prev = res.Remaining
	}
}

func TestSlidingWindowLog_SlideExpiry(t *testing.T) {
	sw, _ := NewSlidingWindowLog(Config{Rate: 2, Window: 100 * time.Millisecond})
	ctx := context.Background()

	sw.Allow(ctx) //nolint:errcheck
	sw.Allow(ctx) //nolint:errcheck
	res, _ := sw.Allow(ctx)
	if res.Allowed {
		t.Fatal("3rd request should be denied immediately")
	}

	time.Sleep(120 * time.Millisecond)

	res, _ = sw.Allow(ctx)
	if !res.Allowed {
		t.Fatal("request after window slide should be allowed")
	}
}

// ─── Fixed Window Tests ───────────────────────────────────────────────────────

func TestNewFixedWindow_InvalidConfig(t *testing.T) {
	_, err := NewFixedWindow(Config{Rate: 0, Window: time.Second})
	if err == nil {
		t.Fatal("expected error for zero rate")
	}
}

func TestFixedWindow_AllowsUpToRate(t *testing.T) {
	const rate = 5
	fw, _ := NewFixedWindow(Config{Rate: rate, Window: time.Second})
	ctx := context.Background()

	for i := 0; i < rate; i++ {
		res, _ := fw.Allow(ctx)
		if !res.Allowed {
			t.Fatalf("request %d should be allowed", i)
		}
	}
}

func TestFixedWindow_DeniesWhenExceeded(t *testing.T) {
	fw, _ := NewFixedWindow(Config{Rate: 3, Window: time.Second})
	ctx := context.Background()
	for i := 0; i < 3; i++ {
		fw.Allow(ctx) //nolint:errcheck
	}
	res, _ := fw.Allow(ctx)
	if res.Allowed {
		t.Fatal("4th request should be denied")
	}
}

func TestFixedWindow_ResetsAtWindowEdge(t *testing.T) {
	fw, _ := NewFixedWindow(Config{Rate: 2, Window: 100 * time.Millisecond})
	ctx := context.Background()

	fw.Allow(ctx) //nolint:errcheck
	fw.Allow(ctx) //nolint:errcheck
	res, _ := fw.Allow(ctx)
	if res.Allowed {
		t.Fatal("3rd request should be denied")
	}

	time.Sleep(110 * time.Millisecond)

	res, _ = fw.Allow(ctx)
	if !res.Allowed {
		t.Fatal("first request in new window should be allowed")
	}
}

func TestFixedWindow_ResetTimeIsInFuture(t *testing.T) {
	fw, _ := NewFixedWindow(Config{Rate: 10, Window: time.Second})
	res, _ := fw.Allow(context.Background())
	if !res.Reset.After(time.Now()) {
		t.Error("Reset must be in the future")
	}
}
