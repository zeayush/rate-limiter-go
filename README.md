# rate-limiter-go

A production-ready HTTP rate limiting library for Go, built as a learning project.  
It ships three algorithms, two storage backends, two middleware adapters, and Prometheus metrics — all behind clean interfaces so each piece can be swapped independently.

---

## What is this?

Rate limiting is the practice of capping how many requests a client can make within a time window. Without it, a single misconfigured client or attacker can exhaust your CPU, memory, or downstream quotas.

This library implements three well-known algorithms, each with a different tradeoff:

| Algorithm | Memory | Smoothness | Boundary spikes |
|---|---|---|---|
| Fixed Window Counter | O(1) per key | None | Yes (2× burst at boundary) |
| Sliding Window Log | O(rate) per key | High | No |
| Token Bucket | O(1) per key | High | No |

Per-key limiting (per IP, per API key, per user) is layered on top via the `KeyedLimiter` interface backed by either in-memory maps or Redis for distributed deployments.

---

## How it Works

### Fixed Window Counter

Divide time into fixed intervals. Count requests in the current interval. Deny when count exceeds the limit. Reset when the window expires.

```
Time ──────────────────────────────────────────────────────────▶
      │◄────── window ──────►│◄────── window ──────►│
      │  req req req req req  │  req req ···          │
      │  count=1,2,3,4,5  ↑  │  count=1,2           │
      │              DENY(6) │                       │
```

**Weakness:** a burst of requests arriving at the end of one window and the start of the next can allow 2× rate.

### Sliding Window Log

Store a timestamp for every successful request. On each `Allow` call, evict timestamps older than `now − window`, then check the remaining count.

```
now − window                              now
     │◄──────────── window ──────────────►│
     │  t1  t2  t3  t4  t5               │   ← Allow: 5 entries = limit
     │                                    │   ← Next call: DENY
     │       (evict t1)                   │   ← Allow: 4 entries < limit
```

**Tradeoff:** memory grows with the allowed rate (`O(rate)` timestamps per key).

### Token Bucket

A bucket fills at a constant rate (tokens per second). Each request consumes one token. Requests are denied when the bucket is empty. The bucket is capped at `burst` capacity.

```
            Rate = 2 tok/s, Burst = 5

Bucket: ██████ ←5 (full)
           ↓ request → ████  (4 remaining)
           ↓ request → ██    (3 remaining)
        1s passes → ████     (refill 2 → 5, capped)
```

**Strength:** smooths traffic; allows short bursts up to `burst` then enforces the steady-state rate.

### Key Extraction and Middleware Chain

```
HTTP Request
     │
     ▼
KeyExtractor(r) ──▶ "192.168.1.1"  or  "api-key-xyz"
     │
     ▼
KeyedLimiter.Allow(ctx, key)
     │                 │
     ▼                 ▼
 RedisStore       MemoryStore (fallback)
     │
     ▼
 Result{Allowed, Limit, Remaining, Reset, RetryAfter}
     │
     ├── Allowed=true  → write X-RateLimit-* headers, call next handler
     └── Allowed=false → 429 Too Many Requests + Retry-After header
```

---

## Quick Start

### Run with Docker Compose

```bash
git clone https://github.com/your-username/rate-limiter-go
cd rate-limiter-go
docker compose up --build
```

This starts:
- **app** on `:8080` — the example Gin server
- **redis** on `:6379` — state backend
- **prometheus** on `:9090` — metrics scraper
- **grafana** on `:3000` — dashboard (`admin` / `admin`)

### Try the API

```bash
# Normal request — succeeds
curl -i -H "X-API-Key: test-key" http://localhost:8080/api/hello

# HTTP/1.1 200 OK
# X-Ratelimit-Limit: 100
# X-Ratelimit-Remaining: 99
# X-Ratelimit-Reset: 1720000060
# {"message":"hello"}

# Exhaust the limit (loop 105 times)
for i in $(seq 1 105); do
  curl -s -o /dev/null -w "%{http_code}\n" -H "X-API-Key: test-key" http://localhost:8080/api/hello
done

# 200 ... 200 (100 times) ... 429 ... 429 (5 times)

# 429 response body
curl -i -H "X-API-Key: exhausted-key" http://localhost:8080/api/hello
# HTTP/1.1 429 Too Many Requests
# Retry-After: 42
# {"error":"rate limit exceeded","retry_after_seconds":42}

# Health check
curl http://localhost:8080/health
# {"redis":"ok"}

# Prometheus metrics
curl http://localhost:8080/metrics | grep rate_limiter
# rate_limiter_requests_total{algorithm="fixed_window",key="test-key",status="allowed"} 100
# rate_limiter_requests_total{algorithm="fixed_window",key="test-key",status="denied"} 5
# rate_limiter_active_keys 1
```

---

## API

### Core types — `limiter` package

```go
type Config struct {
    Rate   int64         // max requests per Window
    Window time.Duration // length of one window
    Burst  int64         // token bucket only: max burst above rate
}

type Result struct {
    Allowed    bool
    Limit      int64
    Remaining  int64
    Reset      time.Time
    RetryAfter time.Duration
}

type Limiter interface {
    Allow(ctx context.Context) (Result, error)
}

type KeyedLimiter interface {
    Allow(ctx context.Context, key string) (Result, error)
}
```

### Constructors

```go
// Single-instance algorithms (implement Limiter)
tb,  err := limiter.NewTokenBucket(limiter.Config{Rate: 10, Window: time.Second, Burst: 20})
sw,  err := limiter.NewSlidingWindowLog(limiter.Config{Rate: 100, Window: time.Minute})
fw,  err := limiter.NewFixedWindow(limiter.Config{Rate: 1000, Window: time.Hour})

// Per-key store backed by in-memory map
mem, err := store.NewMemoryStore(func(key string) (limiter.Limiter, error) {
    return limiter.NewFixedWindow(limiter.Config{Rate: 100, Window: time.Minute})
})

// Per-key store backed by Redis (falls back to mem on error)
rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
rs,  err := store.NewRedisStore(rdb, limiter.Config{Rate: 100, Window: time.Minute}, mem)
```

### Middleware

```go
// net/http
mux.Handle("/api/", middleware.HTTPMiddleware(rs, middleware.IPExtractor))
mux.Handle("/api/", middleware.HTTPMiddleware(rs, middleware.HeaderExtractor("X-API-Key")))

// Gin
r.Use(middleware.GinMiddleware(rs, middleware.GinIPExtractor))
r.Use(middleware.GinMiddleware(rs, middleware.GinHeaderExtractor("X-API-Key")))
```

### Prometheus metrics

```go
m := metrics.New()
m.Register(prometheus.DefaultRegisterer)

// In your middleware or handler:
m.RecordAllow(key, "token_bucket", result.Allowed)
m.SetActiveKeys(int64(mem.Len()))
```

---

## Key Design Decisions

**Interfaces everywhere.** `Limiter` and `KeyedLimiter` are the only abstractions that cross package boundaries. You can swap algorithms, backends, or middleware adapters without touching other layers.

**Fail-open on errors.** Both HTTP and Gin middleware allow the request through when the limiter returns a non-nil error. This trades availability for protection — a Redis outage won't take your service down. Override this by returning a 503 in your `KeyedLimiter` error path if you prefer fail-closed.

**Atomic Redis operations via Lua.** The fixed-window Redis backend increments and sets the expiry in a single Lua script, eliminating the INCR/EXPIRE race condition that is common in naïve implementations.

**Double-checked locking in MemoryStore.** `getOrCreate` acquires a read lock first (hot path), then upgrades to a write lock only when the key is new, minimising contention under concurrent load.

**Lazy token refill.** The token bucket does not run a background goroutine. Tokens are calculated on demand from `(now - lastTime) * rate`, which means zero overhead between requests.

**Key scheme for Redis.** Keys are namespaced as `rl:{key}:{window_start_unix_seconds}`. Redis TTL is set to the window duration so keys self-expire; no background cleanup job is needed.

---

## Benchmarks

Run with:

```bash
go test -bench=. -benchtime=5s -benchmem ./bench/
```

Expected output (Apple M2 / Linux amd64 — actual results depend on your hardware):

```
BenchmarkTokenBucket-8              200000000    5.2 ns/op    0 B/op    0 allocs/op
BenchmarkTokenBucket_Parallel-8    1000000000    1.8 ns/op    0 B/op    0 allocs/op
BenchmarkSlidingWindowLog-8          20000000   61.3 ns/op   24 B/op    1 allocs/op
BenchmarkFixedWindow-8              500000000    3.9 ns/op    0 B/op    0 allocs/op
BenchmarkFixedWindow_Parallel-8     800000000    2.1 ns/op    0 B/op    0 allocs/op
BenchmarkMemoryStore_SingleKey-8    200000000    7.4 ns/op    0 B/op    0 allocs/op
BenchmarkMemoryStore_100Keys-8       50000000   28.6 ns/op   16 B/op    1 allocs/op
```

The sliding window log allocates one `time.Time` per request (list element). All other algorithms are alloc-free on the hot path.

---

## Tests

```bash
# All unit tests
go test ./...

# With race detector (important for concurrent correctness)
go test -race ./...

# Specific package
go test ./limiter/
go test ./store/
go test ./middleware/
```

Tests are pre-written and will fail until you implement the stubs. That is intentional — the tests define the contract.

---

## Project Structure

```
rate-limiter-go/
├── cmd/
│   └── server/
│       └── main.go          # Example Gin server (Redis + fallback)
├── limiter/
│   ├── limiter.go           # Interfaces: Limiter, KeyedLimiter, Config, Result
│   ├── tokenbucket.go       # Token bucket stub
│   ├── slidingwindow.go     # Sliding window log stub
│   ├── fixedwindow.go       # Fixed window counter stub
│   └── limiter_test.go      # 17 pre-written tests
├── store/
│   ├── memory.go            # In-memory KeyedLimiter stub
│   ├── redis.go             # Redis KeyedLimiter stub (Lua script)
│   └── store_test.go        # 5 pre-written tests
├── middleware/
│   ├── http.go              # net/http middleware stub
│   ├── gin.go               # Gin middleware stub
│   └── middleware_test.go   # 7 pre-written tests
├── metrics/
│   └── metrics.go           # Prometheus counters and gauge stub
├── bench/
│   └── bench_test.go        # 7 benchmarks
├── deploy/
│   └── prometheus.yml       # Prometheus scrape config
├── Dockerfile               # Multi-stage: alpine builder → scratch
├── docker-compose.yml       # app + Redis + Prometheus + Grafana
└── go.mod
```

---

## Implementation Guide

The stubs contain structured `// TODO:` comments with step-by-step pseudocode. Here is the intended order:

### 1. `limiter/tokenbucket.go`

`NewTokenBucket`
```
1. Validate: Rate > 0, Window > 0.
2. Compute rate = float64(cfg.Rate) / cfg.Window.Seconds()
3. Set capacity = float64(cfg.Burst) if Burst > 0, else float64(cfg.Rate).
4. Initialise tokens = capacity, lastTime = time.Now().
```

`refill(now time.Time)`
```
1. elapsed = now.Sub(tb.lastTime).Seconds()
2. tb.tokens = min(tb.capacity, tb.tokens + elapsed * tb.rate)
3. tb.lastTime = now
```

`Allow(ctx)`
```
1. Lock mutex.
2. now = time.Now(); call refill(now).
3. If tb.tokens < 1: return Result{Allowed: false, RetryAfter: (1-tokens)/rate seconds}, nil.
4. tb.tokens -= 1
5. Return Result{Allowed: true, Limit: cfg.Rate, Remaining: int64(tb.tokens), Reset: now.Add(1/rate)}, nil.
```

### 2. `limiter/slidingwindow.go`

`NewSlidingWindowLog`
```
1. Validate: Rate > 0, Window > 0.
2. Return &SlidingWindowLog{cfg: cfg, log: list.New()}.
```

`evict(now time.Time)`
```
1. cutoff = now.Sub(sw.cfg.Window)
2. Walk list from front; remove elements where value.(time.Time).Before(cutoff).
3. Stop at first element that is not before cutoff.
```

`Allow(ctx)`
```
1. Lock mutex.
2. now = time.Now(); call evict(now).
3. count = sw.log.Len()
4. If count >= int(cfg.Rate): return denied result with RetryAfter = oldest_entry + window - now.
5. sw.log.PushBack(now).
6. Return allowed result; Remaining = cfg.Rate - int64(count) - 1.
```

### 3. `limiter/fixedwindow.go`

`resetIfExpired(now time.Time)`
```
1. If now.Before(fw.windowStart.Add(cfg.Window)): return (still in window).
2. Advance windowStart by enough full windows to cover now.
3. Reset count = 0.
```

`Allow(ctx)`
```
1. Lock mutex.
2. now = time.Now(); call resetIfExpired(now).
3. If fw.count >= cfg.Rate: return denied result; RetryAfter = windowEnd() - now.
4. fw.count++.
5. Return allowed result; Remaining = cfg.Rate - fw.count.
```

### 4. `store/memory.go`

`getOrCreate(key)`
```
1. ms.mu.RLock(); look up key in ms.limiters; ms.mu.RUnlock().
2. If found, return it.
3. ms.mu.Lock(); look up again (double-checked locking).
4. If still missing: call ms.factory(key); store in ms.limiters.
5. ms.mu.Unlock(); return limiter.
```

`Allow(ctx, key)`
```
1. l, err = ms.getOrCreate(key).
2. return l.Allow(ctx).
```

### 5. `store/redis.go`

`Allow(ctx, key)`
```
1. k = ms.windowKey(key).
2. Run the Lua script: EVAL script 1 k cfg.Rate ttl_ms.
3. Parse result as []int64{count, ttl_ms}.
4. If count > cfg.Rate: return denied with RetryAfter = time.Duration(ttl_ms)*ms.
5. Return allowed result.
6. On any Redis error: log warning; delegate to ms.fallback.Allow(ctx, key).
```

### 6. `middleware/http.go` and `middleware/gin.go`

`writeHeaders(w, res)`
```
1. Set X-RateLimit-Limit   = strconv.FormatInt(res.Limit, 10).
2. Set X-RateLimit-Remaining = strconv.FormatInt(res.Remaining, 10).
3. Set X-RateLimit-Reset   = strconv.FormatInt(res.Reset.Unix(), 10).
4. If !res.Allowed: set Retry-After = strconv.Itoa(int(res.RetryAfter.Seconds())).
```

`HTTPMiddleware(kl, extractor)`
```
1. Return http.HandlerFunc that:
   a. Extracts key via extractor(r).
   b. Calls kl.Allow(r.Context(), key).
   c. On error: call next (fail-open).
   d. Call writeHeaders(w, res).
   e. If !res.Allowed: http.Error(w, "429 Too Many Requests", 429); return.
   f. Else: call next(w, r).
```

### 7. `metrics/metrics.go`

`New()`
```
1. Create rate_limiter_requests_total counter vec with labels [key, algorithm, status].
2. Create rate_limiter_errors_total counter vec with labels [key, algorithm].
3. Create rate_limiter_active_keys gauge.
```

`Register(reg)`
```
1. reg.MustRegister(m.RequestsTotal, m.ErrorsTotal, m.ActiveKeys).
```

---

## Rate Limit Headers

The middleware sets standard response headers on every request:

| Header | Value | Example |
|---|---|---|
| `X-RateLimit-Limit` | Max requests in window | `100` |
| `X-RateLimit-Remaining` | Requests left in current window | `42` |
| `X-RateLimit-Reset` | Unix timestamp of next window start | `1720000060` |
| `Retry-After` | Seconds to wait before retrying (429 only) | `37` |

---

## References

- [Token Bucket — Wikipedia](https://en.wikipedia.org/wiki/Token_bucket) — formal description of the fill-and-drain model
- [Sliding Window Rate Limiting — Figma Engineering](https://www.figma.com/blog/an-alternative-approach-to-rate-limiting/) — practical trade-off analysis
- [Redis INCR pattern for rate limiting](https://redis.io/docs/latest/commands/incr/#pattern-rate-limiter) — the canonical INCR + EXPIRE pattern this library extends with Lua atomicity
- [HTTP RateLimit header fields — IETF Draft](https://ietf-wg-httpapi.github.io/ratelimit-headers/draft-ietf-httpapi-ratelimit-headers.html) — proposed standard this library follows
- [uber-go/ratelimit](https://github.com/uber-go/ratelimit) — Uber's leaky-bucket implementation for reference
- [go-redis/redis](https://github.com/redis/go-redis) — Redis client used in this project
- [gin-gonic/gin](https://github.com/gin-gonic/gin) — HTTP framework used for the example server
- [prometheus/client_golang](https://github.com/prometheus/client_golang) — Prometheus instrumentation library
