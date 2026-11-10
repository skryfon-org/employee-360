package middleware

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/skryfon-org/employee-360/backend/internal/delivery/http/response"
	"github.com/skryfon-org/employee-360/backend/internal/domain/service"
)

// maxThrottleBodyBytes caps how much of a request body is inspected for the
// email; larger bodies are not parsed (the per-IP limit still applies).
const maxThrottleBodyBytes = 64 << 10

// AuthThrottleConfig configures the strict limiter for unauthenticated auth
// endpoints (login, forgot/reset password, invitation accept/validate).
type AuthThrottleConfig struct {
	// PerAccountMax is the max attempts per Window for one (endpoint, email).
	PerAccountMax int
	// PerIPMax is the max attempts per Window for one client IP across all
	// throttled endpoints.
	PerIPMax int
	// Window is the fixed counting window; it is also the maximum lockout length.
	Window time.Duration
	// Cache is the optional shared counter store (Redis). When nil or failing,
	// counters fall back to the in-process store.
	Cache service.Cache
	// ClientIP resolves the key IP; nil uses c.ClientIP().
	ClientIP ClientIPFunc
}

type windowEntry struct {
	count   int
	resetAt time.Time
}

// memCounter is a fixed-window in-memory counter used when Redis is absent.
type memCounter struct {
	mu      sync.Mutex
	entries map[string]*windowEntry
	now     func() time.Time
}

func newMemCounter() *memCounter {
	return &memCounter{entries: make(map[string]*windowEntry), now: time.Now}
}

// incr counts one attempt and returns the new count and the window remainder.
func (m *memCounter) incr(key string, window time.Duration) (int, time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := m.now()
	// Opportunistic sweep keeps memory bounded without a goroutine.
	if len(m.entries) > 1024 {
		for k, e := range m.entries {
			if !now.Before(e.resetAt) {
				delete(m.entries, k)
			}
		}
	}
	e, ok := m.entries[key]
	if !ok || !now.Before(e.resetAt) {
		e = &windowEntry{resetAt: now.Add(window)}
		m.entries[key] = e
	}
	e.count++
	return e.count, e.resetAt.Sub(now)
}

type authThrottle struct {
	cfg AuthThrottleConfig
	mem *memCounter
}

// hit counts an attempt and reports whether it exceeds max, with a retry hint.
func (t *authThrottle) hit(ctx context.Context, key string, max int) (bool, time.Duration) {
	if t.cfg.Cache != nil {
		n, err := t.cfg.Cache.Incr(ctx, key, t.cfg.Window)
		if err == nil {
			// The remaining TTL is not exposed by the port; the full window is a safe upper bound.
			return n > int64(max), t.cfg.Window
		}
	}
	n, remaining := t.mem.incr(key, t.cfg.Window)
	return n > max, remaining
}

func hashPart(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:16])
}

func throttleKey(kind, part string) string {
	k, err := service.GlobalCacheKey("auththrottle", kind+"-"+hashPart(part))
	if err != nil {
		return "e360:global:auththrottle:" + kind + "-" + hashPart(part)
	}
	return k
}

// NormalizeThrottleEmail lowercases and trims an email for use as a throttle key.
func NormalizeThrottleEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// peekEmail reads the JSON body's "email" without consuming it.
func peekEmail(c *gin.Context) string {
	if c.Request.Body == nil || c.Request.Method != http.MethodPost {
		return ""
	}
	raw, err := io.ReadAll(io.LimitReader(c.Request.Body, maxThrottleBodyBytes+1))
	// Restore the full body for the handler (including any unread remainder).
	c.Request.Body = struct {
		io.Reader
		io.Closer
	}{io.MultiReader(bytes.NewReader(raw), c.Request.Body), c.Request.Body}
	if err != nil || len(raw) > maxThrottleBodyBytes {
		return ""
	}
	var body struct {
		Email string `json:"email"`
	}
	if json.Unmarshal(raw, &body) != nil {
		return ""
	}
	return NormalizeThrottleEmail(body.Email)
}

// AuthThrottle returns a strict limiter for unauthenticated auth endpoints. It
// counts every attempt per client IP and, when the JSON body carries an email,
// per (endpoint, normalized email). Both checks are applied to every request
// and blocked requests get the same 429 regardless of whether the account
// exists, so it reveals nothing about accounts. Counters are fixed-window, so a
// lockout never lasts longer than Window. Client IP comes from cfg.ClientIP (see NewClientIPResolver).
func AuthThrottle(cfg AuthThrottleConfig) gin.HandlerFunc {
	t := &authThrottle{cfg: cfg, mem: newMemCounter()}
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		path := strings.TrimSuffix(c.Request.URL.Path, "/")

		blocked, wait := t.hit(ctx, throttleKey("ip", resolveIP(cfg.ClientIP, c)), cfg.PerIPMax)
		if email := peekEmail(c); email != "" {
			if b, w := t.hit(ctx, throttleKey("acct", path+"|"+email), cfg.PerAccountMax); b {
				blocked = true
				if w > wait {
					wait = w
				}
			}
		}
		if !blocked {
			c.Next()
			return
		}
		secs := int(math.Ceil(wait.Seconds()))
		if secs < 1 {
			secs = 1
		}
		c.Header("Retry-After", strconv.Itoa(secs))
		response.Error(c, http.StatusTooManyRequests, "RATE_LIMITED", "too many requests, please retry later")
		c.Abort()
	}
}
