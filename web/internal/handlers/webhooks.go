package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net"
	"net/http"
	"strings"
	"time"

	"nms-web/internal/db"
)

// MaxWebhookBodyBytes caps incoming router webhook JSON bodies at 64 KB.
const MaxWebhookBodyBytes = 64 << 10

// WebhookLinkQualityPayload represents the JSON payload sent by the router.
type WebhookLinkQualityPayload struct {
	TargetIP   string   `json:"target_ip"`
	PacketLoss float64  `json:"packet_loss"`
	RttMin     *float64 `json:"rtt_min,omitempty"`
	RttAvg     *float64 `json:"rtt_avg,omitempty"`
	RttMax     *float64 `json:"rtt_max,omitempty"`
}

// IsValidWebhookTarget returns true if target is a valid IPv4/IPv6 address or RFC-1123 hostname.
func IsValidWebhookTarget(target string) bool {
	if target == "" || len(target) > 253 {
		return false
	}
	if net.ParseIP(target) != nil {
		return true
	}
	labels := strings.Split(target, ".")
	for _, label := range labels {
		if len(label) == 0 || len(label) > 63 {
			return false
		}
		if label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}
		for i := 0; i < len(label); i++ {
			c := label[i]
			isAlphaNum := (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')
			if !isAlphaNum && c != '-' {
				return false
			}
		}
	}
	return true
}

// HandleWebhookLinkQuality handles incoming webhooks for link quality checks.
// POST /api/webhooks/link-quality
func HandleWebhookLinkQuality(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, MaxWebhookBodyBytes)

	// 1. Extract Bearer Token
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		http.Error(w, "Missing Authorization header", http.StatusUnauthorized)
		return
	}

	token := strings.TrimPrefix(authHeader, "Bearer ")
	if token == authHeader {
		http.Error(w, "Invalid Authorization format", http.StatusUnauthorized)
		return
	}
	token = strings.TrimSpace(token)
	if token == "" || len(token) > 128 {
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	// 2. Parse and validate payload before hitting the database
	var payload WebhookLinkQualityPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	payload.TargetIP = strings.TrimSpace(payload.TargetIP)
	if !IsValidWebhookTarget(payload.TargetIP) {
		http.Error(w, "Invalid target_ip", http.StatusBadRequest)
		return
	}

	if math.IsNaN(payload.PacketLoss) || math.IsInf(payload.PacketLoss, 0) || payload.PacketLoss < 0 || payload.PacketLoss > 100 {
		http.Error(w, "Invalid packet_loss value", http.StatusBadRequest)
		return
	}

	// 3. Authenticate token against database
	if db.Pool == nil {
		http.Error(w, "Service unavailable", http.StatusServiceUnavailable)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	var deviceID int
	err := db.Pool.QueryRow(ctx, "SELECT id FROM devices WHERE webhook_token = $1 AND enabled = true", token).Scan(&deviceID)
	if err != nil {
		http.Error(w, "Invalid token or device not found", http.StatusUnauthorized)
		return
	}

	// 4. Insert metric
	_, err = db.Pool.Exec(ctx, `
		INSERT INTO metric_link_quality (time, device_id, target_ip, rtt_min, rtt_avg, rtt_max, packet_loss)
		VALUES (NOW(), $1, $2, $3, $4, $5, $6)
	`, deviceID, payload.TargetIP, payload.RttMin, payload.RttAvg, payload.RttMax, payload.PacketLoss)
	if err != nil {
		log.Printf("HandleWebhookLinkQuality DB error: %v", err)
		http.Error(w, "Failed to store metric", http.StatusInternalServerError)
		return
	}

	// 4. Update Alarms
	handleLinkQualityAlarm(ctx, deviceID, payload)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status": "ok"}`))
}

func handleLinkQualityAlarm(ctx context.Context, deviceID int, payload WebhookLinkQualityPayload) {
	entityType := "link_quality"
	entityID := payload.TargetIP
	name := "Link Quality Ping to " + payload.TargetIP

	// Check for active alarm
	var activeAlarmID int
	var currentSeverity string
	err := db.Pool.QueryRow(ctx, `
		SELECT id, severity FROM alarms 
		WHERE device_id = $1 AND entity_type = $2 AND entity_id = $3 AND status = 'active'
	`, deviceID, entityType, entityID).Scan(&activeAlarmID, &currentSeverity)

	if payload.PacketLoss == 0 {
		// Resolve alarm if exists
		if err == nil { // Found active alarm
			db.Pool.Exec(ctx, "UPDATE alarms SET status = 'resolved', resolved_at = NOW() WHERE id = $1", activeAlarmID)
		}
		return
	}

	// Packet loss > 0
	newSeverity := "warning"
	if payload.PacketLoss > 5 {
		newSeverity = "critical"
	}
	
	message := "Packet loss is at " + formatFloat(payload.PacketLoss) + "%"

	if err == nil { // Found active alarm
		// Update severity/message if changed
		if currentSeverity != newSeverity {
			db.Pool.Exec(ctx, "UPDATE alarms SET severity = $1, message = $2 WHERE id = $3", newSeverity, message, activeAlarmID)
		} else {
			db.Pool.Exec(ctx, "UPDATE alarms SET message = $1 WHERE id = $2", message, activeAlarmID)
		}
	} else {
		// Insert new alarm
		db.Pool.Exec(ctx, `
			INSERT INTO alarms (device_id, entity_type, entity_id, name, severity, status, message)
			VALUES ($1, $2, $3, $4, $5, 'active', $6)
		`, deviceID, entityType, entityID, name, newSeverity, message)
	}
}

// Helper to format float to string avoiding verbose printing
func formatFloat(f float64) string {
	return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%f", f), "0"), ".")
}
