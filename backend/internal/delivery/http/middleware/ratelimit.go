package middleware

import (
	"math"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/skryfon-org/employee-360/backend/internal/delivery/http/response"
	"golang.org/x/time/rate"
)

// RateLimitConfig configures the per-IP limiter.
type RateLimitConfig struct {
	RequestsPerSecond float64
	Burst             int
	CleanupInterval   time.Duration
	IdleTTL           time.Duration
	// ExemptPaths are exact request paths that bypass limiting (e.g. probes).
	ExemptPaths []string
	// ClientIP resolves the key IP; nil uses c.ClientIP().
	ClientIP ClientIPFunc
}

type ipEntry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// ipLimiter holds one token bucket per client IP and evicts idle ones.
type ipLimiter struct {
	mu      sync.Mutex
	entries map[string]*ipEntry
	rps     rate.Limit
	burst   int
	ttl     time.Duration
	now     func() time.Time
}

func newIPLimiter(cfg RateLimitConfig) *ipLimiter {
	return &ipLimiter{
		entries: make(map[string]*ipEntry),
		rps:     rate.Limit(cfg.RequestsPerSecond),
		burst:   cfg.Burst,
		ttl:     cfg.IdleTTL,
		now:     time.Now,
	}
}

// reserve consumes a token for ip. It returns ok=true when allowed, otherwise
// the duration after which a retry would succeed.
func (l *ipLimiter) reserve(ip string) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	e, found := l.entries[ip]
	if !found {
		e = &ipEntry{limiter: rate.NewLimiter(l.rps, l.burst)}
		l.entries[ip] = e
	}
	e.lastSeen = now
	r := e.limiter.ReserveN(now, 1)
	if !r.OK() {
		return false, time.Second
	}
	if d := r.DelayFrom(now); d > 0 {
		r.CancelAt(now) // do not consume a token for a rejected request
		return false, d
	}
	return true, 0
}

// sweep removes limiters idle for longer than the TTL and returns the count removed.
func (l *ipLimiter) sweep() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	cutoff := l.now().Add(-l.ttl)
	n := 0
	for ip, e := range l.entries {
		if e.lastSeen.Before(cutoff) {
			delete(l.entries, ip)
			n++
		}
	}
	return n
}

func (l *ipLimiter) size() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.entries)
}

// RateLimit returns a global per-client-IP token-bucket middleware. The
// returned stop func halts the background cleanup goroutine (may be ignored
// for process-lifetime use). Client IP comes from c.ClientIP(), so configure
// the engine's trusted proxies to keep X-Forwarded-For from being spoofed.
func RateLimit(cfg RateLimitConfig) (gin.HandlerFunc, func()) {
	if cfg.CleanupInterval <= 0 {
		cfg.CleanupInterval = time.Minute
	}
	if cfg.IdleTTL <= 0 {
		cfg.IdleTTL = 10 * time.Minute
	}
	lim := newIPLimiter(cfg)
	exempt := make(map[string]struct{}, len(cfg.ExemptPaths))
	for _, p := range cfg.ExemptPaths {
		exempt[p] = struct{}{}
	}

	done := make(chan struct{})
	var once sync.Once
	go func() {
		t := time.NewTicker(cfg.CleanupInterval)
		defer t.Stop()
		for {
			select {
			case <-t.C:
				lim.sweep()
			case <-done:
				return
			}
		}
	}()
	stop := func() { once.Do(func() { close(done) }) }

	handler := func(c *gin.Context) {
		if _, ok := exempt[strings.TrimSuffix(c.Request.URL.Path, "/")]; ok {
			c.Next()
			return
		}
		ok, wait := lim.reserve(resolveIP(cfg.ClientIP, c))
		if ok {
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
	return handler, stop
}
