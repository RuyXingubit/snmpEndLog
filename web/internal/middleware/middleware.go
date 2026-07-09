// Package middleware provides HTTP middleware for authentication, CSRF, and rate limiting.
package middleware

import (
	"context"
	"log"
	"net/http"
	"sync"
	"time"

	"nms-web/internal/auth"
)

// contextKey is a custom type for context keys to avoid collisions.
type contextKey string

const claimsKey contextKey = "claims"

// GetClaims extracts JWT claims from the request context.
func GetClaims(r *http.Request) *auth.Claims {
	claims, _ := r.Context().Value(claimsKey).(*auth.Claims)
	return claims
}

// RequireAuth is middleware that requires a valid JWT token in cookies.
func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("token")
		if err != nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		claims, err := auth.ValidateToken(cookie.Value)
		if err != nil {
			// Clear invalid cookie
			http.SetCookie(w, &http.Cookie{
				Name:     "token",
				Value:    "",
				Path:     "/",
				MaxAge:   -1,
				HttpOnly: true,
				SameSite: http.SameSiteStrictMode,
			})
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		// Add claims to context
		ctx := context.WithValue(r.Context(), claimsKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireAPI is middleware for API endpoints — returns 401 JSON instead of redirect.
func RequireAPI(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("token")
		if err != nil {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}

		claims, err := auth.ValidateToken(cookie.Value)
		if err != nil {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), claimsKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RateLimiter provides basic rate limiting per IP.
type RateLimiter struct {
	mu       sync.Mutex
	attempts map[string][]time.Time
	window   time.Duration
	max      int
}

// NewRateLimiter creates a rate limiter that allows max attempts per window.
func NewRateLimiter(max int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		attempts: make(map[string][]time.Time),
		window:   window,
		max:      max,
	}
	// Cleanup goroutine
	go func() {
		ticker := time.NewTicker(window)
		defer ticker.Stop()
		for range ticker.C {
			rl.cleanup()
		}
	}()
	return rl
}

func (rl *RateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	cutoff := time.Now().Add(-rl.window)
	for ip, times := range rl.attempts {
		var valid []time.Time
		for _, t := range times {
			if t.After(cutoff) {
				valid = append(valid, t)
			}
		}
		if len(valid) == 0 {
			delete(rl.attempts, ip)
		} else {
			rl.attempts[ip] = valid
		}
	}
}

// Allow checks if the IP is within rate limits and records the attempt.
func (rl *RateLimiter) Allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-rl.window)

	var valid []time.Time
	for _, t := range rl.attempts[ip] {
		if t.After(cutoff) {
			valid = append(valid, t)
		}
	}

	if len(valid) >= rl.max {
		return false
	}

	rl.attempts[ip] = append(valid, now)
	return true
}

// Logging middleware logs all HTTP requests.
func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap response writer to capture status code
		lw := &loggingResponseWriter{ResponseWriter: w, statusCode: 200}
		next.ServeHTTP(lw, r)

		log.Printf("%s %s %d %s",
			r.Method, r.URL.Path, lw.statusCode, time.Since(start),
		)
	})
}

type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (lw *loggingResponseWriter) WriteHeader(code int) {
	lw.statusCode = code
	lw.ResponseWriter.WriteHeader(code)
}
