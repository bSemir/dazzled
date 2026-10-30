package core

import (
	"crypto/sha256"
	"math"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"dazzled/backend/api/rest/v1/handlers/response"
)

const (
	writeRateLimit       = 60
	writeRateWindow      = time.Minute
	maxTrackedRateLimits = 4096
)

type writeRateLimiter struct {
	mu                sync.Mutex
	requests          map[string][]time.Time
	trustProxyHeaders bool
}

func newWriteRateLimiter(trustProxyHeaders bool) *writeRateLimiter {
	return &writeRateLimiter{
		requests:          make(map[string][]time.Time),
		trustProxyHeaders: trustProxyHeaders,
	}
}

func (limiter *writeRateLimiter) handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost || (request.URL.Path != "/api/incidents" && request.URL.Path != "/api/leaderboard" && request.URL.Path != "/api/leaderboard/start") {
			next.ServeHTTP(writer, request)
			return
		}

		clientIP := limiter.clientIP(request)
		rateLimitPath := request.URL.Path
		if strings.HasPrefix(rateLimitPath, "/api/leaderboard") {
			rateLimitPath = "/api/leaderboard"
		}
		digest := sha256.Sum256([]byte(clientIP + "\x00" + rateLimitPath))
		allowed, retryAfter := limiter.allow(digest, time.Now())
		if !allowed {
			writer.Header().Set("Retry-After", strconvDurationSeconds(retryAfter))
			response.WriteError(writer, http.StatusTooManyRequests, "rate_limited", "Too many submissions. Please wait and try again.")
			return
		}
		next.ServeHTTP(writer, request)
	})
}

func (limiter *writeRateLimiter) clientIP(request *http.Request) string {
	if limiter.trustProxyHeaders {
		forwardedFor := request.Header.Get("X-Forwarded-For")
		if forwardedFor != "" {
			candidate := strings.TrimSpace(strings.Split(forwardedFor, ",")[0])
			if ip := net.ParseIP(candidate); ip != nil {
				return normalizeRateLimitIP(ip)
			}
		}
	}
	address := request.RemoteAddr
	if host, _, err := net.SplitHostPort(address); err == nil {
		address = host
	}
	if ip := net.ParseIP(address); ip != nil {
		return normalizeRateLimitIP(ip)
	}
	return address
}

func normalizeRateLimitIP(ip net.IP) string {
	if ipv4 := ip.To4(); ipv4 != nil {
		return ipv4.String()
	}
	mask := net.CIDRMask(64, 128)
	return (&net.IPNet{IP: ip.Mask(mask), Mask: mask}).String()
}

func (limiter *writeRateLimiter) allow(key [32]byte, now time.Time) (bool, time.Duration) {
	limiter.mu.Lock()
	defer limiter.mu.Unlock()

	for trackedKey, timestamps := range limiter.requests {
		if len(timestamps) == 0 || now.Sub(timestamps[len(timestamps)-1]) >= writeRateWindow {
			delete(limiter.requests, trackedKey)
		}
	}

	keyString := string(key[:])
	timestamps := limiter.requests[keyString]
	active := timestamps[:0]
	for _, timestamp := range timestamps {
		if now.Sub(timestamp) < writeRateWindow {
			active = append(active, timestamp)
		}
	}
	if len(active) >= writeRateLimit {
		limiter.requests[keyString] = active
		return false, writeRateWindow - now.Sub(active[0])
	}
	if len(timestamps) == 0 && len(limiter.requests) >= maxTrackedRateLimits {
		var oldestKey string
		var oldestActivity time.Time
		for trackedKey, trackedRequests := range limiter.requests {
			lastActivity := trackedRequests[len(trackedRequests)-1]
			if oldestKey == "" || lastActivity.Before(oldestActivity) {
				oldestKey = trackedKey
				oldestActivity = lastActivity
			}
		}
		delete(limiter.requests, oldestKey)
	}
	limiter.requests[keyString] = append(active, now)
	return true, 0
}

func strconvDurationSeconds(duration time.Duration) string {
	seconds := int(math.Ceil(duration.Seconds()))
	if seconds < 1 {
		seconds = 1
	}
	return strconv.Itoa(seconds)
}
