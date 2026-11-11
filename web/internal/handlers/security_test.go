package handlers

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"nms-web/internal/auth"
	"nms-web/internal/middleware"
)

func requestWithRole(method, target string, body []byte, role string) *http.Request {
	var req *http.Request
	if body != nil {
		req = httptest.NewRequest(method, target, bytes.NewReader(body))
	} else {
		req = httptest.NewRequest(method, target, nil)
	}
	if role != "" {
		ctx := middleware.WithClaims(req.Context(), &auth.Claims{
			UserID:   10,
			Username: "test-" + role,
			Role:     role,
		})
		req = req.WithContext(ctx)
	}
	return req
}

func TestDeviceEditRBAC_BlocksViewer(t *testing.T) {
	// C1: Viewer trying to access /devices/1/edit via HandleDeviceDetail must get 403 Forbidden
	req := requestWithRole(http.MethodGet, "/devices/1/edit", nil, "viewer")
	rec := httptest.NewRecorder()
	HandleDeviceDetail(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Errorf("HandleDeviceDetail(/devices/1/edit) for viewer = %d, want 403", rec.Code)
	}

	// Direct call to HandleDeviceEditPage as viewer must also get 403 Forbidden
	reqDirect := requestWithRole(http.MethodGet, "/devices/1/edit", nil, "viewer")
	recDirect := httptest.NewRecorder()
	HandleDeviceEditPage(recDirect, reqDirect)
	if recDirect.Code != http.StatusForbidden {
		t.Errorf("HandleDeviceEditPage for viewer = %d, want 403", recDirect.Code)
	}

	// State-changing device endpoints must reject viewer with 403 Forbidden
	for _, tc := range []struct {
		name    string
		handler http.HandlerFunc
		path    string
	}{
		{"HandleDeviceAdd", HandleDeviceAdd, "/devices/add"},
		{"HandleDeviceEdit", HandleDeviceEdit, "/devices/edit"},
		{"HandleDeviceDelete", HandleDeviceDelete, "/devices/delete"},
		{"HandleDeviceGenerateToken", HandleDeviceGenerateToken, "/devices/generate-token"},
	} {
		r := requestWithRole(http.MethodPost, tc.path, []byte("device_id=1"), "viewer")
		w := httptest.NewRecorder()
		tc.handler(w, r)
		if w.Code != http.StatusForbidden {
			t.Errorf("%s for viewer = %d, want 403", tc.name, w.Code)
		}
	}
}

func TestSanitizeDeviceForRole(t *testing.T) {
	comm := "secret-snmp-ro"
	tok := "0195f000-1111-7222-8333-444455556666"

	viewerDev := Device{ID: 1, Hostname: "BGP-ATM", Community: &comm, WebhookToken: &tok}
	sanitizeDeviceForRole(&viewerDev, false)
	if viewerDev.Community != nil || viewerDev.WebhookToken != nil {
		t.Error("Expected Community and WebhookToken to be nil for non-admin viewer")
	}

	adminDev := Device{ID: 1, Hostname: "BGP-ATM", Community: &comm, WebhookToken: &tok}
	sanitizeDeviceForRole(&adminDev, true)
	if adminDev.Community == nil || *adminDev.Community != comm {
		t.Error("Expected Community to remain populated for admin")
	}
	if adminDev.WebhookToken == nil || *adminDev.WebhookToken != tok {
		t.Error("Expected WebhookToken to remain populated for admin")
	}
}

func TestHandleAPIAlarmResolve_RBACAndMethod(t *testing.T) {
	// 1. GET method must be rejected with 405 even for admin
	reqGet := requestWithRole(http.MethodGet, "/api/alarms/12/resolve", nil, "admin")
	recGet := httptest.NewRecorder()
	HandleAPIAlarmResolve(recGet, reqGet)
	if recGet.Code != http.StatusMethodNotAllowed {
		t.Errorf("GET /api/alarms/12/resolve = %d, want 405", recGet.Code)
	}

	// 2. Viewer POST must be rejected with 403 Forbidden (User policy decision A1)
	reqViewer := requestWithRole(http.MethodPost, "/api/alarms/12/resolve", nil, "viewer")
	recViewer := httptest.NewRecorder()
	HandleAPIAlarmResolve(recViewer, reqViewer)
	if recViewer.Code != http.StatusForbidden {
		t.Errorf("Viewer POST /api/alarms/12/resolve = %d, want 403", recViewer.Code)
	}

	// 3. Invalid non-numeric alarm ID rejected with 400 Bad Request before DB query
	reqBadID := requestWithRole(http.MethodPost, "/api/alarms/not-a-number/resolve", nil, "admin")
	recBadID := httptest.NewRecorder()
	HandleAPIAlarmResolve(recBadID, reqBadID)
	if recBadID.Code != http.StatusBadRequest {
		t.Errorf("Admin POST /api/alarms/not-a-number/resolve = %d, want 400", recBadID.Code)
	}
}

func TestAIAndStatusEndpoints_BlockViewer(t *testing.T) {
	// User policy decision A2: /ai and /api/ai/* restricted exclusively to admin
	reqPage := requestWithRole(http.MethodGet, "/ai", nil, "viewer")
	recPage := httptest.NewRecorder()
	HandleAI(recPage, reqPage)
	if recPage.Code != http.StatusForbidden {
		t.Errorf("Viewer GET /ai = %d, want 403", recPage.Code)
	}

	reqSessions := requestWithRole(http.MethodGet, "/api/ai/sessions", nil, "viewer")
	recSessions := httptest.NewRecorder()
	HandleAISessions(recSessions, reqSessions)
	if recSessions.Code != http.StatusForbidden {
		t.Errorf("Viewer GET /api/ai/sessions = %d, want 403", recSessions.Code)
	}

	reqAsk := requestWithRole(http.MethodPost, "/api/ai/sessions/1/ask", []byte(`{"question":"hi"}`), "viewer")
	recAsk := httptest.NewRecorder()
	HandleAISessionAction(recAsk, reqAsk)
	if recAsk.Code != http.StatusForbidden {
		t.Errorf("Viewer POST /api/ai/sessions/1/ask = %d, want 403", recAsk.Code)
	}

	reqStatus := requestWithRole(http.MethodGet, "/api/status", nil, "viewer")
	recStatus := httptest.NewRecorder()
	HandleAPIStatus(recStatus, reqStatus)
	if recStatus.Code != http.StatusForbidden {
		t.Errorf("Viewer GET /api/status = %d, want 403", recStatus.Code)
	}
}

func TestHandleLogout_PostOnlyAndSecureCookie(t *testing.T) {
	reqGet := httptest.NewRequest(http.MethodGet, "/logout", nil)
	recGet := httptest.NewRecorder()
	HandleLogout(recGet, reqGet)
	if recGet.Code != http.StatusMethodNotAllowed {
		t.Errorf("GET /logout = %d, want 405 Method Not Allowed", recGet.Code)
	}

	reqPost := httptest.NewRequest(http.MethodPost, "/logout", nil)
	reqPost.RemoteAddr = "172.18.0.2:34567"
	reqPost.Header.Set("X-Forwarded-Proto", "https")
	recPost := httptest.NewRecorder()
	HandleLogout(recPost, reqPost)
	if recPost.Code != http.StatusSeeOther {
		t.Fatalf("POST /logout = %d, want 303 SeeOther", recPost.Code)
	}

	cookies := recPost.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("Expected cleared token cookie on POST /logout")
	}
	if !cookies[0].Secure || !cookies[0].HttpOnly || cookies[0].MaxAge != -1 {
		t.Errorf("Unexpected logout cookie flags: Secure=%v HttpOnly=%v MaxAge=%d",
			cookies[0].Secure, cookies[0].HttpOnly, cookies[0].MaxAge)
	}
}

func TestWebhookLinkQuality_TargetIPAndPayloadValidation(t *testing.T) {
	validTargets := []string{"8.8.8.8", "177.53.139.255", "2001:4860:4860::8888", "dns.google", "gw-01.proserv.net.br"}
	for _, target := range validTargets {
		if !IsValidWebhookTarget(target) {
			t.Errorf("Expected IsValidWebhookTarget(%q) = true", target)
		}
	}

	invalidTargets := []string{
		"",
		"<img src=x onerror=alert(1)>",
		"8.8.8.8; reboot",
		"-invalid.host",
		"spaces not allowed",
		strings.Repeat("a", 254),
	}
	for _, target := range invalidTargets {
		if IsValidWebhookTarget(target) {
			t.Errorf("Expected IsValidWebhookTarget(%q) = false", target)
		}
	}

	// XSS payload in target_ip must be rejected with 400 Bad Request before DB query
	xssBody := []byte(`{"target_ip":"<script>alert(1)</script>","packet_loss":10}`)
	reqXSS := httptest.NewRequest(http.MethodPost, "/api/webhooks/link-quality", bytes.NewReader(xssBody))
	reqXSS.Header.Set("Authorization", "Bearer 0195f000-1111-7222-8333-444455556666")
	recXSS := httptest.NewRecorder()
	HandleWebhookLinkQuality(recXSS, reqXSS)
	if recXSS.Code != http.StatusBadRequest {
		t.Errorf("XSS target_ip status = %d, want 400", recXSS.Code)
	}

	// Out-of-range packet_loss rejected with 400
	badLossBody := []byte(`{"target_ip":"8.8.8.8","packet_loss":150}`)
	reqLoss := httptest.NewRequest(http.MethodPost, "/api/webhooks/link-quality", bytes.NewReader(badLossBody))
	reqLoss.Header.Set("Authorization", "Bearer 0195f000-1111-7222-8333-444455556666")
	recLoss := httptest.NewRecorder()
	HandleWebhookLinkQuality(recLoss, reqLoss)
	if recLoss.Code != http.StatusBadRequest {
		t.Errorf("Invalid packet_loss status = %d, want 400", recLoss.Code)
	}

	// Oversized >64KB body rejected with 400
	hugeBody := append([]byte(`{"target_ip":"8.8.8.8","packet_loss":0,"pad":"`), bytes.Repeat([]byte("A"), 70000)...)
	hugeBody = append(hugeBody, []byte(`"}`)...)
	reqHuge := httptest.NewRequest(http.MethodPost, "/api/webhooks/link-quality", bytes.NewReader(hugeBody))
	reqHuge.Header.Set("Authorization", "Bearer 0195f000-1111-7222-8333-444455556666")
	recHuge := httptest.NewRecorder()
	HandleWebhookLinkQuality(recHuge, reqHuge)
	if recHuge.Code != http.StatusBadRequest {
		t.Errorf("Oversized webhook body status = %d, want 400", recHuge.Code)
	}
}

func TestSanitizeCSVField(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"Normal syslog message", "Normal syslog message"},
		{"=CMD|' /C calc'!A0", "'=CMD|' /C calc'!A0"},
		{"  +SUM(A1:A2)", "'  +SUM(A1:A2)"},
		{"-10+20,with,commas", "\"'-10+20,with,commas\""},
		{"@import(\"evil\")", "\"'@import(\"\"evil\"\")\""},
	}
	for _, tt := range tests {
		if got := SanitizeCSVField(tt.in); got != tt.want {
			t.Errorf("SanitizeCSVField(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestInitTemplates_AllTemplatesParseCleanly(t *testing.T) {
	if err := InitTemplates("../../templates"); err != nil {
		t.Fatalf("InitTemplates failed: %v", err)
	}
}
