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
// Limit - middleware that implements rate limiting
func (rl *RateLimiter) Limit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		clientIP := r.RemoteAddr

		rl.mu.Lock()

		// Initialize client's record if not exists
		if _, exists := rl.requests[clientIP]; !exists {
			rl.requests[clientIP] = &clientRequests{}
		}

		// Get existing client's record
		clientReq := rl.requests[clientIP]

		// reset count if time has passed
		if time.Since(clientReq.lastSeen) > rl.window {
			clientReq.count = 0
		}

		// Increament the request counter and update timestamp
		clientReq.count++
		clientReq.lastSeen = time.Now()

		// check limit
		if clientReq.count > rl.limit {
			rl.mu.Unlock()
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		rl.mu.Unlock()
		// if rate limit not exceeded, continue to next handler
		next.ServeHTTP(w, r)
	})
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
