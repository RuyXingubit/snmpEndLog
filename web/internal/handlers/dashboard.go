package handlers

import (
	"context"
	"net/http"
	"time"

	"nms-web/internal/db"
)

// DashboardData holds all data for the dashboard overview.
type DashboardData struct {
	TotalDevices   int
	DevicesUp      int
	DevicesDown    int
	DevicesUnknown int
	TotalInterfaces int
	InterfacesUp    int
	InterfacesDown  int
	RecentLogs     []LogEntry
	TopTraffic     []TopTrafficEntry
}

// TopTrafficEntry represents a high-traffic interface.
type TopTrafficEntry struct {
	DeviceName string
	IfDescr    string
	InBps      float64
	OutBps     float64
}

// HandleDashboard renders the main dashboard overview page.
func HandleDashboard(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	data := DashboardData{}

	// Device counts
	err := db.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM devices WHERE enabled = TRUE").Scan(&data.TotalDevices)
	if err != nil {
		data.TotalDevices = 0
	}

	_ = db.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM devices WHERE enabled = TRUE AND status = 'up'").Scan(&data.DevicesUp)
	_ = db.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM devices WHERE enabled = TRUE AND status = 'down'").Scan(&data.DevicesDown)
	data.DevicesUnknown = data.TotalDevices - data.DevicesUp - data.DevicesDown

	// Interface counts
	_ = db.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM interfaces").Scan(&data.TotalInterfaces)
	_ = db.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM interfaces WHERE if_oper_status = 1").Scan(&data.InterfacesUp)
	data.InterfacesDown = data.TotalInterfaces - data.InterfacesUp

	// Recent logs (last 20)
	rows, err := db.Pool.Query(ctx, `
		SELECT time, host, COALESCE(severity_name, 'unknown'), COALESCE(app_name, ''),
		       message
		FROM logs ORDER BY time DESC LIMIT 20
	`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var entry LogEntry
			if err := rows.Scan(&entry.Time, &entry.Host, &entry.Severity, &entry.AppName, &entry.Message); err == nil {
				data.RecentLogs = append(data.RecentLogs, entry)
			}
		}
	}

	// Top traffic interfaces (latest reading)
	trows, err := db.Pool.Query(ctx, `
		WITH latest_traffic AS (
			SELECT DISTINCT ON (device_id, if_index) 
				   device_id, if_index, in_bps, out_bps, time
			FROM metric_traffic
			WHERE time > NOW() - INTERVAL '10 minutes'
			  AND in_bps IS NOT NULL
			ORDER BY device_id, if_index, time DESC
		)
		SELECT d.hostname, i.if_descr, lt.in_bps, lt.out_bps
		FROM latest_traffic lt
		JOIN devices d ON d.id = lt.device_id
		JOIN interfaces i ON i.device_id = lt.device_id AND i.if_index = lt.if_index
		ORDER BY (lt.in_bps + lt.out_bps) DESC
		LIMIT 10
	`)
	if err == nil {
		defer trows.Close()
		for trows.Next() {
			var entry TopTrafficEntry
			if err := trows.Scan(&entry.DeviceName, &entry.IfDescr, &entry.InBps, &entry.OutBps); err == nil {
				data.TopTraffic = append(data.TopTraffic, entry)
			}
		}
	}

	renderTemplate(w, "dashboard.html", map[string]interface{}{
		"Title": "Dashboard",
		"Data":  data,
	}, r)
}
