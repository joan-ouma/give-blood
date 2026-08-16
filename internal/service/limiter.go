package service

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

type rateLimitEntry struct {
	attempts  int
	blockedAt time.Time
}

type RateLimiter struct {
	mu      sync.Mutex
	records map[string]*rateLimitEntry
}

func NewRateLimiter() *RateLimiter {
	limiter := &RateLimiter{
		records: make(map[string]*rateLimitEntry),
	}
	go limiter.cleanupLoop()
	return limiter
}

func (rl *RateLimiter) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for key, entry := range rl.records {
			if !entry.blockedAt.IsZero() && now.Sub(entry.blockedAt) > 15*time.Minute {
				delete(rl.records, key)
			} else if entry.blockedAt.IsZero() && now.Sub(entry.blockedAt) > 15*time.Minute { // reset tracking
				delete(rl.records, key)
			}
		}
		rl.mu.Unlock()
	}
}

func (rl *RateLimiter) getIP(r *http.Request) string {
	ip := r.Header.Get("X-Forwarded-For")
	if ip == "" {
		ip = r.Header.Get("X-Real-IP")
	}
	if ip == "" {
		var err error
		ip, _, err = net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			ip = r.RemoteAddr
		}
	}
	return strings.TrimSpace(strings.Split(ip, ",")[0])
}

func (rl *RateLimiter) Allow(r *http.Request, email string) bool {
	ip := rl.getIP(r)
	key := ip + ":" + strings.ToLower(strings.TrimSpace(email))

	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	entry, exists := rl.records[key]
	if !exists {
		rl.records[key] = &rateLimitEntry{
			attempts: 1,
		}
		return true
	}

	if !entry.blockedAt.IsZero() {
		if now.Sub(entry.blockedAt) > 15*time.Minute {
			entry.attempts = 1
			entry.blockedAt = time.Time{}
			return true
		}
		return false
	}

	entry.attempts++
	if entry.attempts > 5 {
		entry.blockedAt = now
		return false
	}

	return true
}
