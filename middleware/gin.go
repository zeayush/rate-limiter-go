package middleware

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"rate-limiter-go/limiter"
)

// ─── Gin middleware ───────────────────────────────────────────────────────────

// GinKeyExtractor extracts the rate-limit key from a Gin context.
// Common usage: use c.ClientIP(), c.GetHeader("X-API-Key"), or c.Param(":userID").
type GinKeyExtractor func(c *gin.Context) string

// GinIPExtractor returns the client's IP address using Gin's built-in ClientIP().
//
// TODO:
//
//	func GinIPExtractor(c *gin.Context) string {
//	    return c.ClientIP()
//	}
func GinIPExtractor(c *gin.Context) string {
	// TODO: return c.ClientIP()
	return c.RemoteIP()
}

// GinHeaderExtractor returns a GinKeyExtractor that uses the value of a
// specific HTTP header (e.g. "X-API-Key") and falls back to ClientIP().
//
// TODO:
//
//	func GinHeaderExtractor(header string) GinKeyExtractor {
//	    return func(c *gin.Context) string {
//	        if v := c.GetHeader(header); v != "" {
//	            return v
//	        }
//	        return c.ClientIP()
//	    }
//	}
func GinHeaderExtractor(header string) GinKeyExtractor {
	// TODO: implement as described above
	_ = header
	return GinIPExtractor
}

// GinMiddleware returns a Gin middleware HandlerFunc that rate-limits requests.
//
// Usage:
//
//	r := gin.New()
//	r.Use(middleware.GinMiddleware(store, middleware.GinIPExtractor))
//
// When a request is denied the middleware aborts with HTTP 429 and sets the
// standard rate-limit headers. It also exposes the Result on the Gin context
// under the key "rl_result" for downstream handlers that want to inspect it.
func GinMiddleware(kl limiter.KeyedLimiter, extractor GinKeyExtractor) gin.HandlerFunc {
	// TODO:
	//  return func(c *gin.Context) {
	//      key := extractor(c)
	//      res, err := kl.Allow(c.Request.Context(), key)
	//      if err != nil {
	//          // Fail open: don't block traffic on limiter errors.
	//          c.Next()
	//          return
	//      }
	//      // Set headers on every response.
	//      c.Header("X-RateLimit-Limit",     strconv.FormatInt(res.Limit, 10))
	//      c.Header("X-RateLimit-Remaining", strconv.FormatInt(res.Remaining, 10))
	//      c.Header("X-RateLimit-Reset",     strconv.FormatInt(res.Reset.Unix(), 10))
	//      if !res.Allowed {
	//          retrySeconds := int64(res.RetryAfter.Seconds()) + 1
	//          c.Header("Retry-After", strconv.FormatInt(retrySeconds, 10))
	//          c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
	//              "error":       "rate limit exceeded",
	//              "retry_after": retrySeconds,
	//          })
	//          return
	//      }
	//      c.Set("rl_result", res)
	//      c.Next()
	//  }
	_ = strconv.FormatInt // hint
	_ = http.StatusTooManyRequests // hint
	return func(c *gin.Context) {
		c.Next()
	}
}
