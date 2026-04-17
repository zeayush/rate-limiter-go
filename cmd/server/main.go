package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"

	"rate-limiter-go/limiter"
	"rate-limiter-go/metrics"
	"rate-limiter-go/middleware"
	"rate-limiter-go/store"
)

func main() {
	// ── Redis client ──────────────────────────────────────────────────────────
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}
	rdb := redis.NewClient(&redis.Options{Addr: redisAddr})

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Printf("warning: Redis unavailable (%v) — using in-memory fallback", err)
	}

	// ── Rate limit config ─────────────────────────────────────────────────────
	cfg := limiter.Config{
		Rate:   100,          // 100 requests …
		Window: time.Minute,  // … per minute per key
	}

	// ── In-memory fallback store ──────────────────────────────────────────────
	memStore, err := store.NewMemoryStore(func(_ string) (limiter.Limiter, error) {
		return limiter.NewFixedWindow(cfg)
	})
	if err != nil {
		log.Fatalf("memory store: %v", err)
	}

	// ── Redis store ───────────────────────────────────────────────────────────
	redisStore, err := store.NewRedisStore(rdb, cfg, memStore)
	if err != nil {
		log.Fatalf("redis store: %v", err)
	}

	// ── Prometheus metrics ────────────────────────────────────────────────────
	m := metrics.New()
	if err := m.Register(prometheus.DefaultRegisterer); err != nil {
		log.Fatalf("metrics: %v", err)
	}

	// ── Gin router ────────────────────────────────────────────────────────────
	r := gin.New()
	r.Use(gin.Recovery())

	// Apply rate limiting to /api routes, keyed by X-API-Key header.
	keyExtractor := middleware.GinHeaderExtractor("X-API-Key")
	api := r.Group("/api")
	api.Use(func(c *gin.Context) {
		key := keyExtractor(c)
		c.Next()

		m.SetActiveKeys(memStore.Len())
		if _, hasErr := c.Get("rl_error"); hasErr {
			m.RecordError(key, "fixed_window")
			return
		}
		allowed := c.Writer.Status() != http.StatusTooManyRequests
		m.RecordAllow(key, "fixed_window", allowed)
	})
	api.Use(middleware.GinMiddleware(redisStore, keyExtractor))
	{
		api.GET("/hello", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "hello, world"})
		})
	}

	// Health check — not rate-limited.
	r.GET("/health", func(c *gin.Context) {
		pingErr := redisStore.Ping(c.Request.Context())
		if pingErr != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"redis": "unavailable",
				"error": pingErr.Error(),
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{"redis": "ok", "keys": memStore.Len()})
	})

	// Prometheus metrics — not rate-limited.
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// ─────────────────────────────────────────────────────────────────────────
	addr := ":8080"
	log.Printf("rate-limiter-go listening on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
