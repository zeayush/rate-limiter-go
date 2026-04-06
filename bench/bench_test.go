package bench

import (
	"context"
	"testing"
	"time"

	"rate-limiter-go/limiter"
	"rate-limiter-go/store"
)

// ─── Algorithm micro-benchmarks ───────────────────────────────────────────────
//
// Run with:
//   go test -bench=. -benchmem -benchtime=5s ./bench/
//
// Expected throughput targets (post-implementation, single-threaded):
//   BenchmarkTokenBucket       ≥ 10 000 000 ops/sec
//   BenchmarkSlidingWindowLog  ≥  3 000 000 ops/sec
//   BenchmarkFixedWindow       ≥ 10 000 000 ops/sec
//   BenchmarkMemoryStore       ≥  2 000 000 ops/sec (includes map lookup)
//
// All benchmarks use a 1-minute window so the rate limit is never hit
// during the run (avoiding the park path which would skew results).

var (
	benchRate   int64         = 10_000_000
	benchWindow time.Duration = time.Minute
)

func BenchmarkTokenBucket(b *testing.B) {
	tb, err := limiter.NewTokenBucket(limiter.Config{
		Rate:   benchRate,
		Window: benchWindow,
	})
	if err != nil {
		b.Fatalf("setup: %v", err)
	}
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tb.Allow(ctx) //nolint:errcheck
	}
}

func BenchmarkTokenBucket_Parallel(b *testing.B) {
	tb, _ := limiter.NewTokenBucket(limiter.Config{Rate: benchRate, Window: benchWindow})
	ctx := context.Background()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			tb.Allow(ctx) //nolint:errcheck
		}
	})
}

func BenchmarkSlidingWindowLog(b *testing.B) {
	sw, err := limiter.NewSlidingWindowLog(limiter.Config{
		Rate:   benchRate,
		Window: benchWindow,
	})
	if err != nil {
		b.Fatalf("setup: %v", err)
	}
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sw.Allow(ctx) //nolint:errcheck
	}
}

func BenchmarkFixedWindow(b *testing.B) {
	fw, err := limiter.NewFixedWindow(limiter.Config{
		Rate:   benchRate,
		Window: benchWindow,
	})
	if err != nil {
		b.Fatalf("setup: %v", err)
	}
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		fw.Allow(ctx) //nolint:errcheck
	}
}

func BenchmarkFixedWindow_Parallel(b *testing.B) {
	fw, _ := limiter.NewFixedWindow(limiter.Config{Rate: benchRate, Window: benchWindow})
	ctx := context.Background()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			fw.Allow(ctx) //nolint:errcheck
		}
	})
}

func BenchmarkMemoryStore_SingleKey(b *testing.B) {
	ms, err := store.NewMemoryStore(func(_ string) (limiter.Limiter, error) {
		return limiter.NewFixedWindow(limiter.Config{Rate: benchRate, Window: benchWindow})
	})
	if err != nil {
		b.Fatalf("setup: %v", err)
	}
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ms.Allow(ctx, "bench-key") //nolint:errcheck
	}
}

func BenchmarkMemoryStore_100Keys(b *testing.B) {
	keys := make([]string, 100)
	for i := range keys {
		keys[i] = "key-" + string(rune('A'+i%26))
	}
	ms, _ := store.NewMemoryStore(func(_ string) (limiter.Limiter, error) {
		return limiter.NewFixedWindow(limiter.Config{Rate: benchRate, Window: benchWindow})
	})
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ms.Allow(ctx, keys[i%100]) //nolint:errcheck
	}
}
