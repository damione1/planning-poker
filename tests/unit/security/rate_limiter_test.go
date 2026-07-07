package security_test

import (
	"sync"
	"testing"
	"time"

	"github.com/damione1/planning-poker/internal/security"
)

func TestIPRateLimiter_AllowsUpToLimit(t *testing.T) {
	limiter := security.NewIPRateLimiter(3, time.Minute)

	for i := 0; i < 3; i++ {
		if !limiter.Allow("1.2.3.4") {
			t.Fatalf("request %d: expected allowed, got denied", i+1)
		}
	}

	if limiter.Allow("1.2.3.4") {
		t.Fatal("expected 4th request within the window to be denied")
	}
}

func TestIPRateLimiter_PerIPIsolation(t *testing.T) {
	limiter := security.NewIPRateLimiter(1, time.Minute)

	if !limiter.Allow("1.1.1.1") {
		t.Fatal("expected first request from 1.1.1.1 to be allowed")
	}
	if limiter.Allow("1.1.1.1") {
		t.Fatal("expected second request from 1.1.1.1 to be denied")
	}

	// A different IP must have its own independent counter.
	if !limiter.Allow("2.2.2.2") {
		t.Fatal("expected first request from 2.2.2.2 to be allowed despite 1.1.1.1 being exhausted")
	}
}

func TestIPRateLimiter_WindowRollover(t *testing.T) {
	window := 50 * time.Millisecond
	limiter := security.NewIPRateLimiter(1, window)

	if !limiter.Allow("9.9.9.9") {
		t.Fatal("expected first request to be allowed")
	}
	if limiter.Allow("9.9.9.9") {
		t.Fatal("expected second request within the same window to be denied")
	}

	time.Sleep(window * 3)

	if !limiter.Allow("9.9.9.9") {
		t.Fatal("expected request after window rollover to be allowed again")
	}
}

func TestIPRateLimiter_ConcurrentAccess(t *testing.T) {
	const limit = 100
	limiter := security.NewIPRateLimiter(limit, time.Minute)

	const goroutines = 20
	const attemptsPerGoroutine = 20 // 400 total attempts against a limit of 100

	var wg sync.WaitGroup
	var mu sync.Mutex
	allowedCount := 0

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < attemptsPerGoroutine; j++ {
				if limiter.Allow("shared-ip") {
					mu.Lock()
					allowedCount++
					mu.Unlock()
				}
			}
		}()
	}

	// Concurrently hammer a handful of other IPs too, to exercise the
	// pruning path and map access alongside the shared-IP contention above.
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			ip := "isolated-ip"
			for j := 0; j < attemptsPerGoroutine; j++ {
				limiter.Allow(ip)
			}
		}(i)
	}

	wg.Wait()

	if allowedCount != limit {
		t.Fatalf("expected exactly %d allowed requests under concurrency, got %d", limit, allowedCount)
	}
}
