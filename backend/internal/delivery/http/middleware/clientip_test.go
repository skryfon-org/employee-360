package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func resolveWith(f ClientIPFunc, trusted []string, remote, realIP string) string {
	gin.SetMode(gin.TestMode)
	e := gin.New()
	_ = e.SetTrustedProxies(trusted)
	var got string
	e.GET("/", func(c *gin.Context) { got = f(c) })
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = remote
	if realIP != "" {
		req.Header.Set("X-Real-IP", realIP)
		req.Header.Set("X-Forwarded-For", realIP)
	}
	e.ServeHTTP(httptest.NewRecorder(), req)
	return got
}

func TestClientIPResolver(t *testing.T) {
	trusted := []string{"10.89.0.0/24"}
	f := NewClientIPResolver(trusted, zerolog.Nop())

	assert.Equal(t, "203.0.113.9", resolveWith(f, trusted, "10.89.0.1:5000", "203.0.113.9"), "trusted peer: X-Real-IP used")
	assert.Equal(t, "10.89.0.1", resolveWith(f, trusted, "10.89.0.1:5000", ""), "trusted peer, no header")
	assert.Equal(t, "10.89.0.1", resolveWith(f, trusted, "10.89.0.1:5000", "not-an-ip"), "invalid header ignored")
	assert.Equal(t, "198.51.100.7", resolveWith(f, trusted, "198.51.100.7:5000", "1.2.3.4"), "spoofed X-Real-IP from untrusted peer ignored")
	assert.Equal(t, "198.51.100.7", resolveWith(NewClientIPResolver(nil, zerolog.Nop()), nil, "198.51.100.7:5000", "1.2.3.4"), "no trusted proxies: header ignored")
	single := NewClientIPResolver([]string{"10.89.0.1"}, zerolog.Nop())
	assert.Equal(t, "5.5.5.5", resolveWith(single, []string{"10.89.0.1"}, "10.89.0.1:1", "5.5.5.5"), "single-IP entry")
}

func TestLimiterNotBypassedBySpoofedRealIP(t *testing.T) {
	gin.SetMode(gin.TestMode)
	f := NewClientIPResolver(nil, zerolog.Nop())
	h, stop := RateLimit(RateLimitConfig{RequestsPerSecond: 0.001, Burst: 1, ClientIP: f})
	defer stop()
	e := gin.New()
	_ = e.SetTrustedProxies(nil)
	e.Use(h)
	e.GET("/", func(c *gin.Context) { c.Status(200) })
	codes := []int{}
	for _, spoof := range []string{"1.1.1.1", "2.2.2.2"} {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "198.51.100.7:1"
		req.Header.Set("X-Real-IP", spoof)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		codes = append(codes, rec.Code)
	}
	assert.Equal(t, []int{200, 429}, codes)
}

func TestLimiterSeparatesClientsBehindTrustedProxy(t *testing.T) {
	gin.SetMode(gin.TestMode)
	trusted := []string{"10.89.0.0/24"}
	h, stop := RateLimit(RateLimitConfig{RequestsPerSecond: 0.001, Burst: 1, ClientIP: NewClientIPResolver(trusted, zerolog.Nop())})
	defer stop()
	e := gin.New()
	_ = e.SetTrustedProxies(trusted)
	e.Use(h)
	e.GET("/", func(c *gin.Context) { c.Status(200) })
	codes := []int{}
	for _, ip := range []string{"1.1.1.1", "2.2.2.2", "1.1.1.1"} {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "10.89.0.1:1"
		req.Header.Set("X-Real-IP", ip)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		codes = append(codes, rec.Code)
	}
	assert.Equal(t, []int{200, 200, 429}, codes)
}

func TestClientIPMisconfigWarning(t *testing.T) {
	var buf safeBuf
	log := zerolog.New(&buf)
	r := &clientIPResolver{log: log, now: time.Now}
	for _, ip := range []string{"1.1.1.1", "1.1.1.2", "1.1.1.3", "1.1.1.4", "1.1.1.5", "1.1.1.6", "1.1.1.7"} {
		r.observe("10.0.0.1", ip)
	}
	out := buf.String()
	assert.Contains(t, out, "SERVER_TRUSTED_PROXIES")
	assert.Equal(t, 1, countOccurrences(out, "SERVER_TRUSTED_PROXIES"), "warning is rate-limited")
}

type safeBuf struct{ b []byte }

func (s *safeBuf) Write(p []byte) (int, error) { s.b = append(s.b, p...); return len(p), nil }
func (s *safeBuf) String() string              { return string(s.b) }

func countOccurrences(s, sub string) int {
	n := 0
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			n++
		}
	}
	return n
}
