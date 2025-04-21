package server

import (
	"net/http"
	"sync"
	"time"
)

// RateLimiter tracks and limits requests from clients
type RateLimiter struct {
	requests map[string]*clientRequests // tracks requests per IP(key: IP address per client)
	mu sync.RWMutex
	limit int
	window time.Duration  				// time window for counting requests
}

// clientRequest stores info about each client's requests
type clientRequests struct {
	count int				// number of requests made
	lastSeen time.Time
}

// NewRateLimiter creates a new rate limiter instance
func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		requests: make(map[string]*clientRequests),
		limit: limit,
		window: window,
	}
}

// cleanUp removes old entries to prevent memory leaks/ growth
// uses mutex to unsure thread-safe operation
func (rl *RateLimiter) cleanUp() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Remove entries older than the window
	for ip, req := range rl.requests {
		if time.Since(req.lastSeen) > rl.window {
			delete(rl.requests, ip)
		}
	}
}

// handles HTTP to HTTPS redirection:
func redirectToHTTPS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Forwarded-Proto") != "https" {
			sslUrl := "https://" + r.Host + r.RequestURI
			http.Redirect(w, r, sslUrl, http.StatusTemporaryRedirect)
			return
		}
		next.ServeHTTP(w, r)
	})
}
