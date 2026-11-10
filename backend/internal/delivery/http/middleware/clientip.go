package middleware

import (
	"net"
	"net/netip"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

// ClientIPFunc resolves the client address used to key per-IP limiters.
type ClientIPFunc func(c *gin.Context) string

const (
	// misconfigWindow is the observation window for the shared-peer heuristic.
	misconfigWindow = time.Minute
	// misconfigDistinct is how many different claimed client IPs behind one
	// untrusted peer, within a window, trigger the warning.
	misconfigDistinct = 5
	// misconfigMaxTracked bounds memory used by the heuristic.
	misconfigMaxTracked = 64
	// misconfigWarnEvery rate-limits the warning itself.
	misconfigWarnEvery = 10 * time.Minute
)

type clientIPResolver struct {
	trusted []netip.Prefix
	log     zerolog.Logger
	now     func() time.Time

	mu         sync.Mutex
	windowEnd  time.Time
	claimed    map[string]struct{} // distinct X-Real-IP values seen behind one untrusted peer
	claimPeer  string
	lastWarned time.Time
}

// NewClientIPResolver builds the single client-IP resolver shared by the global
// limiter and AuthThrottle. trustedProxies are IPs or CIDRs (same syntax as
// SERVER_TRUSTED_PROXIES). X-Real-IP is honoured only when the direct TCP peer
// (RemoteAddr) is a trusted proxy and the header parses as an IP; otherwise the
// resolver falls back to c.ClientIP(), which never trusts forwarded headers
// from untrusted peers when the engine's trusted proxies match this list.
//
// It also logs a rate-limited warning when several distinct clients appear
// behind one untrusted peer, which means the trusted proxy list is unset or
// wrong and every user would otherwise share one rate-limit bucket.
func NewClientIPResolver(trustedProxies []string, log zerolog.Logger) ClientIPFunc {
	r := &clientIPResolver{log: log, now: time.Now}
	for _, s := range trustedProxies {
		s = strings.TrimSpace(s)
		if p, err := netip.ParsePrefix(s); err == nil {
			r.trusted = append(r.trusted, p.Masked())
		} else if a, err := netip.ParseAddr(s); err == nil {
			r.trusted = append(r.trusted, netip.PrefixFrom(a.Unmap(), a.Unmap().BitLen()))
		}
	}
	return r.resolve
}

func (r *clientIPResolver) isTrusted(a netip.Addr) bool {
	a = a.Unmap()
	for _, p := range r.trusted {
		if p.Contains(a) {
			return true
		}
	}
	return false
}

func (r *clientIPResolver) resolve(c *gin.Context) string {
	host, _, err := net.SplitHostPort(c.Request.RemoteAddr)
	if err != nil {
		host = c.Request.RemoteAddr
	}
	peer, perr := netip.ParseAddr(host)
	real := strings.TrimSpace(c.GetHeader("X-Real-IP"))
	realAddr, rerr := netip.ParseAddr(real)

	if perr == nil && r.isTrusted(peer) {
		if rerr == nil {
			return realAddr.Unmap().String()
		}
		return c.ClientIP()
	}
	// Untrusted peer: never use forwarded headers; only watch for misconfiguration.
	if perr == nil && rerr == nil && realAddr.Unmap() != peer.Unmap() {
		r.observe(peer.Unmap().String(), realAddr.Unmap().String())
	}
	if perr == nil {
		return peer.Unmap().String()
	}
	return c.ClientIP()
}

func (r *clientIPResolver) observe(peer, claimed string) {
	now := r.now()
	r.mu.Lock()
	if now.After(r.windowEnd) || r.claimPeer != peer {
		r.windowEnd = now.Add(misconfigWindow)
		r.claimPeer = peer
		r.claimed = make(map[string]struct{})
	}
	if len(r.claimed) < misconfigMaxTracked {
		r.claimed[claimed] = struct{}{}
	}
	warn := len(r.claimed) >= misconfigDistinct &&
		(r.lastWarned.IsZero() || now.Sub(r.lastWarned) >= misconfigWarnEvery)
	if warn {
		r.lastWarned = now
	}
	r.mu.Unlock()
	if warn {
		r.log.Warn().Str("peer", peer).Int("distinct_client_ips", misconfigDistinct).
			Msg("many distinct clients (X-Real-IP) share one untrusted source address; " +
				"per-IP rate limits treat them as a single client. Set SERVER_TRUSTED_PROXIES " +
				"to the address or CIDR nginx appears as (see docs/self-hosting.md)")
	}
}

func resolveIP(f ClientIPFunc, c *gin.Context) string {
	if f == nil {
		return c.ClientIP()
	}
	return f(c)
}
