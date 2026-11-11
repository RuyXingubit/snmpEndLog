package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"nms-web/internal/auth"
)

func TestClientIP(t *testing.T) {
	tests := []struct {
		name       string
		remoteAddr string
		xff        string
		want       string
	}{
		{
			name:       "Direct public IP strips ephemeral port",
			remoteAddr: "79.72.59.54:49632",
			xff:        "",
			want:       "79.72.59.54",
		},
		{
			name:       "Direct public IP ignores spoofed X-Forwarded-For",
			remoteAddr: "79.72.59.54:49632",
			xff:        "8.8.8.8",
			want:       "79.72.59.54",
		},
		{
			name:       "Trusted Docker proxy extracts real public IP",
			remoteAddr: "172.18.0.2:55112",
			xff:        "177.53.136.10",
			want:       "177.53.136.10",
		},
		{
			name:       "Trusted proxy defeats leftmost X-Forwarded-For spoofing",
			remoteAddr: "172.18.0.2:55112",
			xff:        "1.1.1.1, 79.72.59.54",
			want:       "79.72.59.54",
		},
		{
			name:       "Trusted proxy preserves private LAN client IP when all hops are private",
			remoteAddr: "172.18.0.2:55112",
			xff:        "10.254.255.10",
			want:       "10.254.255.10",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/login", nil)
			req.RemoteAddr = tt.remoteAddr
			if tt.xff != "" {
				req.Header.Set("X-Forwarded-For", tt.xff)
			}
			got := ClientIP(req)
			if got != tt.want {
				t.Errorf("ClientIP() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestIsSecureRequest(t *testing.T) {
	reqProxyHTTPS := httptest.NewRequest(http.MethodGet, "/", nil)
	reqProxyHTTPS.RemoteAddr = "172.18.0.2:41234"
	reqProxyHTTPS.Header.Set("X-Forwarded-Proto", "https")
	if !IsSecureRequest(reqProxyHTTPS) {
		t.Error("Expected IsSecureRequest=true behind trusted proxy with X-Forwarded-Proto=https")
	}

	reqSpoofed := httptest.NewRequest(http.MethodGet, "/", nil)
	reqSpoofed.RemoteAddr = "79.72.59.54:41234"
	reqSpoofed.Header.Set("X-Forwarded-Proto", "https")
	if IsSecureRequest(reqSpoofed) {
		t.Error("Expected IsSecureRequest=false when untrusted public peer spoofs X-Forwarded-Proto")
	}
}

func TestRateLimiter_BlocksSameIPAcrossEphemeralPortsAndCapsKeys(t *testing.T) {
	rl := NewRateLimiter(3, time.Minute)
	rl.maxKeys = 5

	for port := 50001; port <= 50003; port++ {
		req := httptest.NewRequest(http.MethodPost, "/login", nil)
		req.RemoteAddr = "172.18.0.2:40000"
		req.Header.Set("X-Forwarded-For", "79.72.59.54")
		ip := ClientIP(req)
		if !rl.Allow(ip) {
			t.Fatalf("Attempt %d should have been allowed", port-50000)
		}
	}

	// 4th attempt from same client IP must be blocked
	req := httptest.NewRequest(http.MethodPost, "/login", nil)
	req.RemoteAddr = "172.18.0.2:40999"
	req.Header.Set("X-Forwarded-For", "79.72.59.54")
	if rl.Allow(ClientIP(req)) {
		t.Fatal("4th attempt from same IP should have been rate-limited")
	}

	// Fill remaining key slots (already has 1 key, maxKeys=5 -> 4 more)
	for _, ip := range []string{"1.1.1.1", "1.1.1.2", "1.1.1.3", "1.1.1.4"} {
		if !rl.Allow(ip) {
			t.Fatalf("Expected %s to be allowed within maxKeys", ip)
		}
	}
	if rl.Allow("1.1.1.5") {
		t.Fatal("Expected key beyond maxKeys cap to be rejected")
	}
}

func TestSecurityHeaders(t *testing.T) {
	handler := SecurityHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "172.18.0.2:12345"
	req.Header.Set("X-Forwarded-Proto", "https")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if got := rec.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Errorf("X-Content-Type-Options = %q, want nosniff", got)
	}
	if got := rec.Header().Get("X-Frame-Options"); got != "DENY" {
		t.Errorf("X-Frame-Options = %q, want DENY", got)
	}
	if got := rec.Header().Get("Strict-Transport-Security"); got == "" {
		t.Error("Expected Strict-Transport-Security header on HTTPS request")
	}
	if got := rec.Header().Get("Content-Security-Policy"); got == "" {
		t.Error("Expected Content-Security-Policy header")
	}
}

func TestCSRFOriginCheck(t *testing.T) {
	nextOK := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	h := CSRFOriginCheck(nextOK)

	// 1. Cross-origin POST blocked
	crossReq := httptest.NewRequest(http.MethodPost, "https://nms.proserv.net.br/devices/delete", nil)
	crossReq.Host = "nms.proserv.net.br"
	crossReq.Header.Set("Origin", "https://evil.example.com")
	crossRec := httptest.NewRecorder()
	h.ServeHTTP(crossRec, crossReq)
	if crossRec.Code != http.StatusForbidden {
		t.Errorf("Cross-origin POST status = %d, want 403", crossRec.Code)
	}

	// 2. Cross-site Sec-Fetch-Site blocked
	secReq := httptest.NewRequest(http.MethodPost, "https://nms.proserv.net.br/logout", nil)
	secReq.Host = "nms.proserv.net.br"
	secReq.Header.Set("Sec-Fetch-Site", "cross-site")
	secRec := httptest.NewRecorder()
	h.ServeHTTP(secRec, secReq)
	if secRec.Code != http.StatusForbidden {
		t.Errorf("Cross-site Sec-Fetch-Site status = %d, want 403", secRec.Code)
	}

	// 3. Same-origin POST allowed
	sameReq := httptest.NewRequest(http.MethodPost, "https://nms.proserv.net.br/devices/delete", nil)
	sameReq.Host = "nms.proserv.net.br"
	sameReq.Header.Set("Origin", "https://nms.proserv.net.br")
	sameReq.Header.Set("Sec-Fetch-Site", "same-origin")
	sameRec := httptest.NewRecorder()
	h.ServeHTTP(sameRec, sameReq)
	if sameRec.Code != http.StatusOK {
		t.Errorf("Same-origin POST status = %d, want 200", sameRec.Code)
	}

	// 4. Bearer webhook exempt from browser Origin check
	whReq := httptest.NewRequest(http.MethodPost, "https://nms.proserv.net.br/api/webhooks/link-quality", nil)
	whReq.Header.Set("Origin", "https://router.local")
	whRec := httptest.NewRecorder()
	h.ServeHTTP(whRec, whReq)
	if whRec.Code != http.StatusOK {
		t.Errorf("Webhook POST status = %d, want 200", whRec.Code)
	}
}

func TestRequireAuthAndAPI_RevokesDeletedUserAndSyncsDowngradedRole(t *testing.T) {
	if err := auth.InitSecret("0123456789abcdef0123456789abcdef"); err != nil {
		t.Fatalf("InitSecret: %v", err)
	}

	origLookup := auth.UserLookupFunc
	defer func() { auth.UserLookupFunc = origLookup }()

	// Issue a token originally claiming role="admin"
	token, err := auth.GenerateToken(auth.User{
		ID:       42,
		Username: "operator",
		Role:     "admin",
	})
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}

	// Case 1: User demoted to "viewer" in DB -> RequireAdminAPI must immediately block with 403
	auth.UserLookupFunc = func(ctx context.Context, userID int) (string, string, error) {
		return "operator", "viewer", nil
	}

	adminEndpoint := RequireAPI(RequireAdminAPI(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})))

	reqDemoted := httptest.NewRequest(http.MethodPost, "/api/alarms/1/resolve", nil)
	reqDemoted.AddCookie(&http.Cookie{Name: "token", Value: token})
	recDemoted := httptest.NewRecorder()
	adminEndpoint.ServeHTTP(recDemoted, reqDemoted)
	if recDemoted.Code != http.StatusForbidden {
		t.Errorf("Demoted user status = %d, want 403 Forbidden", recDemoted.Code)
	}

	// Case 2: User deleted from DB -> RequireAPI returns 401 and RequireAuth redirects to /login
	auth.UserLookupFunc = func(ctx context.Context, userID int) (string, string, error) {
		return "", "", errors.New("no rows in result set")
	}

	reqDeletedAPI := httptest.NewRequest(http.MethodGet, "/api/alarms", nil)
	reqDeletedAPI.AddCookie(&http.Cookie{Name: "token", Value: token})
	recDeletedAPI := httptest.NewRecorder()
	RequireAPI(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(recDeletedAPI, reqDeletedAPI)
	if recDeletedAPI.Code != http.StatusUnauthorized {
		t.Errorf("Deleted user API status = %d, want 401 Unauthorized", recDeletedAPI.Code)
	}

	reqDeletedPage := httptest.NewRequest(http.MethodGet, "/devices", nil)
	reqDeletedPage.AddCookie(&http.Cookie{Name: "token", Value: token})
	recDeletedPage := httptest.NewRecorder()
	RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(recDeletedPage, reqDeletedPage)
	if recDeletedPage.Code != http.StatusSeeOther {
		t.Errorf("Deleted user page status = %d, want 303 SeeOther", recDeletedPage.Code)
	}
}
