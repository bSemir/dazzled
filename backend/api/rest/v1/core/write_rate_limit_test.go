// Package core tests HTTP middleware behavior and abuse-control limits.
package core

import (
	"crypto/sha256"
	"fmt"
	"net/http/httptest"
	"testing"
	"time"
)

func TestClientIPIgnoresSpoofedForwardedHeaderByDefault(t *testing.T) {
	request := httptest.NewRequest("POST", "/api/incidents", nil)
	request.RemoteAddr = "203.0.113.10:54321"
	request.Header.Set("X-Forwarded-For", "198.51.100.99")
	limiter := newWriteRateLimiter(false)
	if got := limiter.clientIP(request); got != "203.0.113.10" {
		t.Fatalf("client address=%q, want TCP peer address", got)
	}
}

func TestClientIPGroupsIPv6PrivacyAddressesBy64(t *testing.T) {
	limiter := newWriteRateLimiter(false)
	first := httptest.NewRequest("POST", "/api/incidents", nil)
	first.RemoteAddr = "[2001:db8:abcd:12ff:1234::1]:54321"
	second := httptest.NewRequest("POST", "/api/incidents", nil)
	second.RemoteAddr = "[2001:db8:abcd:12ff:ffff::abcd]:12345"
	if gotFirst, gotSecond := limiter.clientIP(first), limiter.clientIP(second); gotFirst != gotSecond {
		t.Fatalf("IPv6 addresses in the same /64 produced different keys: %q and %q", gotFirst, gotSecond)
	}
}

func TestRateLimiterBoundsTrackedAddresses(t *testing.T) {
	limiter := newWriteRateLimiter(false)
	now := time.Now()
	for index := 0; index < maxTrackedRateLimits+1; index++ {
		key := sha256.Sum256([]byte(fmt.Sprintf("address-%d", index)))
		if allowed, _ := limiter.allow(key, now); !allowed {
			t.Fatalf("address %d unexpectedly rejected", index)
		}
	}
	if got := len(limiter.requests); got != maxTrackedRateLimits {
		t.Fatalf("tracked address count=%d, want cap %d", got, maxTrackedRateLimits)
	}
}
