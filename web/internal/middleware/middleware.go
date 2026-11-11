// Package middleware provides HTTP middleware for authentication, CSRF, security headers, and rate limiting.
package middleware

import (
	"context"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"nms-web/internal/auth"
)

// contextKey is a custom type for context keys to avoid collisions.
type contextKey string

const claimsKey contextKey = "claims"

// MaxRateLimiterKeys caps tracked IP entries to prevent unbounded memory growth under DDoS.
const MaxRateLimiterKeys = 10000

// GetClaims extracts JWT claims from the request context.
func GetClaims(r *http.Request) *auth.Claims {
	claims, _ := r.Context().Value(claimsKey).(*auth.Claims)
	return claims
}

// WithClaims attaches JWT claims to a context (useful for middleware and unit tests).
func WithClaims(ctx context.Context, claims *auth.Claims) context.Context {
	return context.WithValue(ctx, claimsKey, claims)
}

// isTrustedProxy returns true when the immediate TCP peer is loopback or private (e.g. Caddy on Docker network).
func isTrustedProxy(ip net.IP) bool {
	if ip == nil {
		return false
	}
	return ip.IsLoopback() || ip.IsPrivate()
}

// remoteHost extracts the raw IP address (without port) from r.RemoteAddr.
func remoteHost(remoteAddr string) string {
	host, _, err := net.SplitHostPort(strings.TrimSpace(remoteAddr))
	if err == nil {
		return host
	}
	return strings.TrimSpace(remoteAddr)
}

// IsSecureRequest reports whether the request arrived over TLS directly or via a trusted HTTPS reverse proxy.
func IsSecureRequest(r *http.Request) bool {
	if r.TLS != nil {
		return true
	}
	peerIP := net.ParseIP(remoteHost(r.RemoteAddr))
	if isTrustedProxy(peerIP) {
		proto := strings.TrimSpace(strings.Split(r.Header.Get("X-Forwarded-Proto"), ",")[0])
		return strings.EqualFold(proto, "https")
	}
	return false
}

// ClientIP extracts the real client IP address without ephemeral port, trusting X-Forwarded-For
// only when the immediate TCP peer is a trusted loopback/private reverse proxy.
func ClientIP(r *http.Request) string {
	peerHost := remoteHost(r.RemoteAddr)
	peerIP := net.ParseIP(peerHost)

	if isTrustedProxy(peerIP) {
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			parts := strings.Split(xff, ",")
			var firstValid string
			// Walk right-to-left so client-spoofed leftmost XFF entries cannot bypass rate limiting
			for i := len(parts) - 1; i >= 0; i-- {
				candidate := strings.TrimSpace(parts[i])
				ip := net.ParseIP(candidate)
				if ip == nil {
					continue
				}
				if firstValid == "" {
					firstValid = ip.String()
				}
				if !isTrustedProxy(ip) {
					return ip.String()
				}
			}
			if firstValid != "" {
				return firstValid
			}
		}
	}

	if peerIP != nil {
		return peerIP.String()
	}
	return peerHost
}

func clearAuthCookie(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   IsSecureRequest(r),
		SameSite: http.SameSiteStrictMode,
	})
}

func refreshActiveClaims(ctx context.Context, claims *auth.Claims) error {
	lookupCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	username, role, err := auth.UserLookupFunc(lookupCtx, claims.UserID)
	if err != nil {
		return err
	}
	claims.Username = username
	claims.Role = role
	return nil
}

// RequireAuth is middleware that requires a valid JWT token in cookies and an active DB user.
func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("token")
		if err != nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		claims, err := auth.ValidateToken(cookie.Value)
		if err != nil {
			clearAuthCookie(w, r)
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		if err := refreshActiveClaims(r.Context(), claims); err != nil {
			clearAuthCookie(w, r)
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		ctx := WithClaims(r.Context(), claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireAPI is middleware for API endpoints — returns 401 JSON instead of redirect.
func RequireAPI(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("token")
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}

		claims, err := auth.ValidateToken(cookie.Value)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}

		if err := refreshActiveClaims(r.Context(), claims); err != nil {
			clearAuthCookie(w, r)
			w.Header().Set("Content-Type", "application/json")
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}

		ctx := WithClaims(r.Context(), claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RateLimiter provides bounded rate limiting per IP.
type RateLimiter struct {
	mu       sync.Mutex
	attempts map[string][]time.Time
	window   time.Duration
	max      int
	maxKeys  int
}

// NewRateLimiter creates a rate limiter that allows max attempts per window.
func NewRateLimiter(max int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		attempts: make(map[string][]time.Time),
		window:   window,
		max:      max,
		maxKeys:  MaxRateLimiterKeys,
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
	rl.cleanupLocked(time.Now())
}

func (rl *RateLimiter) cleanupLocked(now time.Time) {
	cutoff := now.Add(-rl.window)
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

	existing, exists := rl.attempts[ip]
	if !exists && rl.maxKeys > 0 && len(rl.attempts) >= rl.maxKeys {
		rl.cleanupLocked(now)
		if len(rl.attempts) >= rl.maxKeys {
			return false
		}
	}

	var valid []time.Time
	for _, t := range existing {
		if t.After(cutoff) {
			valid = append(valid, t)
		}
	}

	if len(valid) >= rl.max {
		rl.attempts[ip] = valid
		return false
	}

	rl.attempts[ip] = append(valid, now)
	return true
}

// SecurityHeaders adds standard defensive HTTP headers to every response.
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
		w.Header().Set("Content-Security-Policy", "frame-ancestors 'none'; base-uri 'self'; form-action 'self'")
		if IsSecureRequest(r) {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}
		next.ServeHTTP(w, r)
	})
}

// CSRFOriginCheck blocks cross-origin state-changing browser requests on cookie-authenticated routes.
func CSRFOriginCheck(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
			next.ServeHTTP(w, r)
			return
		}

		// Bearer-authenticated router webhooks do not use browser cookies
		if strings.HasPrefix(r.URL.Path, "/api/webhooks/") {
			next.ServeHTTP(w, r)
			return
		}

		if secFetch := r.Header.Get("Sec-Fetch-Site"); secFetch != "" {
			if secFetch != "same-origin" && secFetch != "none" {
				http.Error(w, "Forbidden: cross-site request blocked", http.StatusForbidden)
				return
			}
		}

		if origin := r.Header.Get("Origin"); origin != "" {
			u, err := url.Parse(origin)
			if err != nil || u.Host == "" || !strings.EqualFold(u.Host, r.Host) {
				http.Error(w, "Forbidden: invalid origin", http.StatusForbidden)
				return
			}
		}

		next.ServeHTTP(w, r)
	})
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

// RequireAdmin is middleware that blocks non-admin users from accessing admin-only routes.
// It expects RequireAuth to have already run (claims in context).
func RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims := GetClaims(r)
		if claims == nil || claims.Role != "admin" {
			http.Error(w, "Acesso negado: permissão de administrador necessária.", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// RequireAdminAPI is middleware for API endpoints that blocks non-admin users with a JSON 403.
func RequireAdminAPI(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims := GetClaims(r)
		if claims == nil || claims.Role != "admin" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte(`{"error":"admin access required"}`))
			return
		}
		next.ServeHTTP(w, r)
	})
}
