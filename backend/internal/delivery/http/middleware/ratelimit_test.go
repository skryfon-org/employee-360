package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func rlEngine(cfg RateLimitConfig) *gin.Engine {
	gin.SetMode(gin.TestMode)
	e := gin.New()
	_ = e.SetTrustedProxies(nil)
	h, stop := RateLimit(cfg)
	_ = stop
	e.Use(h)
	e.GET("/x", func(c *gin.Context) { c.String(200, "ok") })
	e.GET("/health", func(c *gin.Context) { c.String(200, "ok") })
	return e
}

func hit(e *gin.Engine, path, remote string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.RemoteAddr = remote + ":1234"
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

func TestRateLimit_AllowsUnderLimit(t *testing.T) {
	e := rlEngine(RateLimitConfig{RequestsPerSecond: 1, Burst: 3})
	for i := 0; i < 3; i++ {
		assert.Equal(t, 200, hit(e, "/x", "1.1.1.1").Code)
	}
}

func TestRateLimit_429AfterBurstWithRetryAfterAndEnvelope(t *testing.T) {
	e := rlEngine(RateLimitConfig{RequestsPerSecond: 1, Burst: 2})
	hit(e, "/x", "1.1.1.1")
	hit(e, "/x", "1.1.1.1")
	rec := hit(e, "/x", "1.1.1.1")
	require.Equal(t, http.StatusTooManyRequests, rec.Code)
	ra, err := strconv.Atoi(rec.Header().Get("Retry-After"))
	require.NoError(t, err)
	assert.GreaterOrEqual(t, ra, 1)
	var body struct {
		Success bool `json:"success"`
		Error   struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.False(t, body.Success)
	assert.Equal(t, "RATE_LIMITED", body.Error.Code)
}

func TestRateLimit_PerIPIsolation(t *testing.T) {
	e := rlEngine(RateLimitConfig{RequestsPerSecond: 0.001, Burst: 1})
	assert.Equal(t, 200, hit(e, "/x", "1.1.1.1").Code)
	assert.Equal(t, 429, hit(e, "/x", "1.1.1.1").Code)
	assert.Equal(t, 200, hit(e, "/x", "2.2.2.2").Code)
}

func TestRateLimit_XForwardedForNotTrustedByDefault(t *testing.T) {
	e := rlEngine(RateLimitConfig{RequestsPerSecond: 0.001, Burst: 1})
	send := func(xff string) int {
		req := httptest.NewRequest(http.MethodGet, "/x", nil)
		req.RemoteAddr = "1.1.1.1:1"
		req.Header.Set("X-Forwarded-For", xff)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		return rec.Code
	}
	assert.Equal(t, 200, send("9.9.9.9"))
	assert.Equal(t, 429, send("8.8.8.8"))
}

func TestRateLimit_ExemptPaths(t *testing.T) {
	e := rlEngine(RateLimitConfig{RequestsPerSecond: 0.001, Burst: 1, ExemptPaths: []string{"/health"}})
	for i := 0; i < 5; i++ {
		assert.Equal(t, 200, hit(e, "/health", "1.1.1.1").Code)
	}
}

func TestIPLimiter_StaleCleanup(t *testing.T) {
	now := time.Now()
	l := newIPLimiter(RateLimitConfig{RequestsPerSecond: 1, Burst: 1, IdleTTL: time.Minute})
	l.now = func() time.Time { return now }
	l.reserve("a")
	now = now.Add(30 * time.Second)
	l.reserve("b")
	assert.Equal(t, 2, l.size())
	now = now.Add(45 * time.Second) // a idle 75s, b idle 45s
	assert.Equal(t, 1, l.sweep())
	assert.Equal(t, 1, l.size())
}

func TestIPLimiter_ConcurrentSafe(t *testing.T) {
	l := newIPLimiter(RateLimitConfig{RequestsPerSecond: 1000, Burst: 1000, IdleTTL: time.Minute})
	done := make(chan struct{})
	for i := 0; i < 8; i++ {
		go func() {
			for j := 0; j < 200; j++ {
				l.reserve("ip" + strconv.Itoa(j%5))
				l.sweep()
			}
			done <- struct{}{}
		}()
	}
	for i := 0; i < 8; i++ {
		<-done
	}
}
