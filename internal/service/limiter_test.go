package service

import (
	"net/http/httptest"
	"testing"
	"time"
)

func TestRateLimiterIPExtraction(t *testing.T) {
	rl := NewRateLimiter()

	t.Run("X-Forwarded-For Header", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/login", nil)
		req.Header.Set("X-Forwarded-For", "203.0.113.195, 70.41.3.18")
		ip := rl.getIP(req)
		if ip != "203.0.113.195" {
			t.Errorf("expected 203.0.113.195, got %s", ip)
		}
	})

	t.Run("X-Real-IP Header", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/login", nil)
		req.Header.Set("X-Real-IP", "198.51.100.1")
		ip := rl.getIP(req)
		if ip != "198.51.100.1" {
			t.Errorf("expected 198.51.100.1, got %s", ip)
		}
	})

	t.Run("RemoteAddr Fallback", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/login", nil)
		req.RemoteAddr = "192.0.2.1:12345"
		ip := rl.getIP(req)
		if ip != "192.0.2.1" {
			t.Errorf("expected 192.0.2.1, got %s", ip)
		}
	})
}

func TestRateLimiterBlocking(t *testing.T) {
	rl := NewRateLimiter()
	email := "test@example.com"
	req := httptest.NewRequest("POST", "/login", nil)
	req.RemoteAddr = "1.2.3.4:5678"

	// First 5 attempts should be allowed
	for i := 1; i <= 5; i++ {
		if !rl.Allow(req, email) {
			t.Errorf("attempt %d should be allowed", i)
		}
	}

	// 6th attempt should block
	if rl.Allow(req, email) {
		t.Error("6th attempt should have been blocked")
	}

	// Another email from same IP should be allowed
	if !rl.Allow(req, "other@example.com") {
		t.Error("different email from same IP should be allowed")
	}

	// Trigger expiration check by manually resetting blockedAt to the past
	rl.mu.Lock()
	key := "1.2.3.4:" + email
	entry := rl.records[key]
	entry.blockedAt = time.Now().Add(-16 * time.Minute)
	rl.mu.Unlock()

	// Should be allowed again after 15 minute block duration
	if !rl.Allow(req, email) {
		t.Error("should be allowed again after block expiration")
	}
}
