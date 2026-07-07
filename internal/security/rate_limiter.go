package security

import (
	"sync"
	"time"
)

// visitorState tracks one IP's request count within its current window.
type visitorState struct {
	count       int
	windowStart time.Time
}

// IPRateLimiter is a concurrent-safe, per-IP fixed-window request limiter.
// Each IP gets its own counter that resets once the window has elapsed
// since that IP's first request in the current window.
//
// Stale entries are pruned lazily on access (see Allow) rather than via a
// background goroutine, so there is no janitor to leak or shut down -
// callers never need a Stop method.
type IPRateLimiter struct {
	mu       sync.Mutex
	limit    int
	window   time.Duration
	visitors map[string]*visitorState
}

// NewIPRateLimiter creates a limiter that allows up to limit requests per
// window duration, tracked independently per IP address.
func NewIPRateLimiter(limit int, window time.Duration) *IPRateLimiter {
	return &IPRateLimiter{
		limit:    limit,
		window:   window,
		visitors: make(map[string]*visitorState),
	}
}

// Allow reports whether a request from ip should be permitted, recording it
// if so. Concurrent calls (including for the same ip) are safe.
func (l *IPRateLimiter) Allow(ip string) bool {
	now := time.Now()

	l.mu.Lock()
	defer l.mu.Unlock()

	l.pruneLocked(now)

	v, ok := l.visitors[ip]
	if !ok || now.Sub(v.windowStart) >= l.window {
		l.visitors[ip] = &visitorState{count: 1, windowStart: now}
		return true
	}

	if v.count >= l.limit {
		return false
	}

	v.count++
	return true
}

// pruneLocked removes visitor entries whose window expired well in the
// past, keeping the map from growing unbounded across many distinct
// one-off IPs. Must be called with l.mu held.
func (l *IPRateLimiter) pruneLocked(now time.Time) {
	for ip, v := range l.visitors {
		if now.Sub(v.windowStart) >= 2*l.window {
			delete(l.visitors, ip)
		}
	}
}
