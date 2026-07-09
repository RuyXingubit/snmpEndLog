package handlers

import (
	"context"
	"net/http"
	"time"

	"nms-web/internal/db"
)

// Alarm represents a system alarm.
type Alarm struct {
	ID         int        `json:"id"`
	DeviceID   int        `json:"device_id"`
	EntityType string     `json:"entity_type"`
	EntityID   string     `json:"entity_id"`
	Name       string     `json:"name"`
	Severity   string     `json:"severity"`
	Status     string     `json:"status"`
	Message    string     `json:"message"`
	CreatedAt  time.Time  `json:"created_at"`
	ResolvedAt *time.Time `json:"resolved_at,omitempty"`
}

// HandleAPIAlarms returns a list of active alarms.
// GET /api/alarms
func HandleAPIAlarms(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	rows, err := db.Pool.Query(ctx, `
		SELECT id, device_id, entity_type, entity_id, name, severity, status, message, created_at, resolved_at
		FROM alarms
		WHERE status = 'active'
		ORDER BY created_at DESC
	`)
	if err != nil {
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	defer rows.Close()

	var alarms []Alarm
	for rows.Next() {
		var a Alarm
		if err := rows.Scan(
			&a.ID, &a.DeviceID, &a.EntityType, &a.EntityID,
			&a.Name, &a.Severity, &a.Status, &a.Message,
			&a.CreatedAt, &a.ResolvedAt,
		); err != nil {
			continue
		}
		alarms = append(alarms, a)
	}

	if alarms == nil {
		alarms = []Alarm{} // return empty array instead of null
	}

	jsonResponse(w, http.StatusOK, alarms)
}

// HandleAPIAlarmResolve resolves an active alarm manually.
// POST /api/alarms/{id}/resolve
func HandleAPIAlarmResolve(w http.ResponseWriter, r *http.Request) {
	// Extract ID from path manually since we don't have a strict router
	// e.g. /api/alarms/123/resolve
	pathParts := r.URL.Path[len("/api/alarms/"):] // "123/resolve"
	idStr := ""
	for i := 0; i < len(pathParts); i++ {
		if pathParts[i] == '/' {
			break
		}
		idStr += string(pathParts[i])
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	_, err := db.Pool.Exec(ctx, `
		UPDATE alarms 
		SET status = 'resolved', resolved_at = NOW()
		WHERE id = $1 AND status = 'active'
	`, idStr)

	if err != nil {
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	jsonResponse(w, http.StatusOK, map[string]string{"status": "ok"})
}
