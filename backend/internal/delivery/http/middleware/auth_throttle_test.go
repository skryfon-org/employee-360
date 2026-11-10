package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeCache struct {
	counts map[string]int64
	fail   bool
}

func (f *fakeCache) Get(context.Context, string) (string, error)              { return "", errors.New("x") }
func (f *fakeCache) Set(context.Context, string, string, time.Duration) error { return nil }
func (f *fakeCache) Delete(context.Context, string) error                     { return nil }
func (f *fakeCache) Ping(context.Context) error                               { return nil }
func (f *fakeCache) Incr(_ context.Context, k string, _ time.Duration) (int64, error) {
	if f.fail {
		return 0, errors.New("down")
	}
	f.counts[k]++
	return f.counts[k], nil
}

func throttleEngine(cfg AuthThrottleConfig) (*gin.Engine, *int) {
	gin.SetMode(gin.TestMode)
	e := gin.New()
	_ = e.SetTrustedProxies(nil)
	e.Use(AuthThrottle(cfg))
	reached := 0
	e.POST("/login", func(c *gin.Context) {
		var b struct {
			Email string `json:"email"`
		}
		_ = c.ShouldBindJSON(&b) // proves the body was not consumed
		reached++
		c.String(200, b.Email)
	})
	e.POST("/other", func(c *gin.Context) { c.String(200, "ok") })
	return e, &reached
}

func post(e *gin.Engine, path, ip, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = ip + ":1"
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

func TestAuthThrottle_PerAccountBlocksAcrossIPsAndOthersUnaffected(t *testing.T) {
	e, reached := throttleEngine(AuthThrottleConfig{PerAccountMax: 3, PerIPMax: 100, Window: time.Minute})
	for i := 0; i < 3; i++ {
		rec := post(e, "/login", "10.0.0."+strconv.Itoa(i+1), `{"email":"A@x.com"}`)
		require.Equal(t, 200, rec.Code)
		assert.Equal(t, "a@x.com", strings.ToLower(rec.Body.String()), "body must reach handler intact")
	}
	// Case/whitespace variants and a new IP still hit the same account bucket.
	rec := post(e, "/login", "10.0.0.99", `{"email":"  a@X.com "}`)
	assert.Equal(t, 429, rec.Code)
	assert.NotEmpty(t, rec.Header().Get("Retry-After"))
	assert.Contains(t, rec.Body.String(), "RATE_LIMITED")
	assert.Equal(t, 3, *reached)
	// A different account from a fresh IP is unaffected.
	assert.Equal(t, 200, post(e, "/login", "10.0.0.50", `{"email":"b@x.com"}`).Code)
}

func TestAuthThrottle_PerIPBlocksAcrossAccounts(t *testing.T) {
	e, _ := throttleEngine(AuthThrottleConfig{PerAccountMax: 100, PerIPMax: 3, Window: time.Minute})
	for i := 0; i < 3; i++ {
		require.Equal(t, 200, post(e, "/login", "1.1.1.1", `{"email":"u`+strconv.Itoa(i)+`@x.com"}`).Code)
	}
	assert.Equal(t, 429, post(e, "/login", "1.1.1.1", `{"email":"fresh@x.com"}`).Code)
	assert.Equal(t, 429, post(e, "/other", "1.1.1.1", `{}`).Code, "IP bucket spans throttled endpoints")
	assert.Equal(t, 200, post(e, "/login", "2.2.2.2", `{"email":"fresh@x.com"}`).Code)
}

func TestAuthThrottle_AccountBucketIsPerEndpoint(t *testing.T) {
	e, _ := throttleEngine(AuthThrottleConfig{PerAccountMax: 1, PerIPMax: 100, Window: time.Minute})
	require.Equal(t, 200, post(e, "/login", "1.1.1.1", `{"email":"a@x.com"}`).Code)
	assert.Equal(t, 429, post(e, "/login", "1.1.1.1", `{"email":"a@x.com"}`).Code)
	assert.Equal(t, 200, post(e, "/other", "1.1.1.1", `{"email":"a@x.com"}`).Code)
}

func TestAuthThrottle_WindowExpiry(t *testing.T) {
	cfg := AuthThrottleConfig{PerAccountMax: 2, PerIPMax: 100, Window: time.Minute}
	gin.SetMode(gin.TestMode)
	// Build manually to control the clock.
	th := &authThrottle{cfg: cfg, mem: newMemCounter()}
	now := time.Unix(1000, 0)
	th.mem.now = func() time.Time { return now }
	for i := 0; i < 2; i++ {
		b, _ := th.hit(context.Background(), "k", 2)
		require.False(t, b)
	}
	b, wait := th.hit(context.Background(), "k", 2)
	assert.True(t, b)
	assert.Equal(t, time.Minute, wait)
	now = now.Add(61 * time.Second)
	b, _ = th.hit(context.Background(), "k", 2)
	assert.False(t, b, "window expired, attempts allowed again")
}

func TestAuthThrottle_UsesCacheAndFallsBackToMemory(t *testing.T) {
	fc := &fakeCache{counts: map[string]int64{}}
	e, _ := throttleEngine(AuthThrottleConfig{PerAccountMax: 1, PerIPMax: 100, Window: time.Minute, Cache: fc})
	require.Equal(t, 200, post(e, "/login", "1.1.1.1", `{"email":"a@x.com"}`).Code)
	assert.Equal(t, 429, post(e, "/login", "1.1.1.1", `{"email":"a@x.com"}`).Code)
	assert.NotEmpty(t, fc.counts)
	for k := range fc.counts {
		assert.NotContains(t, k, "a@x.com", "no raw PII in cache keys")
	}

	fc2 := &fakeCache{fail: true}
	e2, _ := throttleEngine(AuthThrottleConfig{PerAccountMax: 1, PerIPMax: 100, Window: time.Minute, Cache: fc2})
	require.Equal(t, 200, post(e2, "/login", "1.1.1.1", `{"email":"a@x.com"}`).Code)
	assert.Equal(t, 429, post(e2, "/login", "1.1.1.1", `{"email":"a@x.com"}`).Code)
}

func TestAuthThrottle_NonJSONBodyOnlyIPLimited(t *testing.T) {
	e, _ := throttleEngine(AuthThrottleConfig{PerAccountMax: 1, PerIPMax: 100, Window: time.Minute})
	for i := 0; i < 3; i++ {
		assert.Equal(t, 200, post(e, "/other", "1.1.1.1", `not json`).Code)
	}
}
