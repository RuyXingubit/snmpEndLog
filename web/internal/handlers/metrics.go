package handlers

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"snmpendlog-web/internal/db"
)

// MetricPoint represents a single data point for charts.
type MetricPoint struct {
	Time  time.Time `json:"time"`
	Value float64   `json:"value"`
}

// TrafficData represents traffic metrics for a chart.
type TrafficData struct {
	InBps  []MetricPoint `json:"in_bps"`
	OutBps []MetricPoint `json:"out_bps"`
}

// SystemData represents system metrics for a chart.
type SystemData struct {
	CPU    []MetricPoint `json:"cpu"`
	Memory []MetricPoint `json:"memory"`
	PPPoE  []MetricPoint `json:"pppoe,omitempty"`
}

// PingData represents ping metrics for a chart.
type PingData struct {
	RTT        []MetricPoint `json:"rtt"`
	PacketLoss []MetricPoint `json:"packet_loss"`
}

// HandleAPITraffic returns traffic metrics as JSON for chart rendering.
// GET /api/metrics/traffic?device_id=1&if_index=1&period=1h
func HandleAPITraffic(w http.ResponseWriter, r *http.Request) {
	deviceID, _ := strconv.Atoi(r.URL.Query().Get("device_id"))
	ifIndex, _ := strconv.Atoi(r.URL.Query().Get("if_index"))
	period := parsePeriod(r.URL.Query().Get("period"))

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	rows, err := db.Pool.Query(ctx, `
		SELECT time, COALESCE(in_bps, 0), COALESCE(out_bps, 0)
		FROM metric_traffic
		WHERE device_id = $1 AND if_index = $2 AND time > NOW() - $3::interval
		ORDER BY time ASC
	`, deviceID, ifIndex, period)
	if err != nil {
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	defer rows.Close()

	data := TrafficData{}
	for rows.Next() {
		var t time.Time
		var inBps, outBps float64
		if err := rows.Scan(&t, &inBps, &outBps); err != nil {
			continue
		}
		data.InBps = append(data.InBps, MetricPoint{Time: t, Value: inBps})
		data.OutBps = append(data.OutBps, MetricPoint{Time: t, Value: outBps})
	}

	jsonResponse(w, http.StatusOK, data)
}

// HandleAPISystem returns system metrics (CPU/memory) as JSON.
// GET /api/metrics/system?device_id=1&period=1h
func HandleAPISystem(w http.ResponseWriter, r *http.Request) {
	deviceID, _ := strconv.Atoi(r.URL.Query().Get("device_id"))
	period := parsePeriod(r.URL.Query().Get("period"))

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	rows, err := db.Pool.Query(ctx, `
		SELECT time, COALESCE(cpu_percent, 0), COALESCE(memory_percent, 0), pppoe_online
		FROM metric_system
		WHERE device_id = $1 AND time > NOW() - $2::interval
		ORDER BY time ASC
	`, deviceID, period)
	if err != nil {
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	defer rows.Close()

	data := SystemData{}
	for rows.Next() {
		var t time.Time
		var cpu, mem float64
		var pppoe *int
		if err := rows.Scan(&t, &cpu, &mem, &pppoe); err != nil {
			continue
		}
		data.CPU = append(data.CPU, MetricPoint{Time: t, Value: cpu})
		data.Memory = append(data.Memory, MetricPoint{Time: t, Value: mem})
		if pppoe != nil {
			data.PPPoE = append(data.PPPoE, MetricPoint{Time: t, Value: float64(*pppoe)})
		}
	}

	jsonResponse(w, http.StatusOK, data)
}

// HandleAPIPing returns ping metrics as JSON.
// GET /api/metrics/ping?device_id=1&period=1h
func HandleAPIPing(w http.ResponseWriter, r *http.Request) {
	deviceID, _ := strconv.Atoi(r.URL.Query().Get("device_id"))
	period := parsePeriod(r.URL.Query().Get("period"))

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	rows, err := db.Pool.Query(ctx, `
		SELECT time, COALESCE(rtt_avg, 0), COALESCE(packet_loss, 0)
		FROM metric_ping
		WHERE device_id = $1 AND time > NOW() - $2::interval
		ORDER BY time ASC
	`, deviceID, period)
	if err != nil {
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	defer rows.Close()

	data := PingData{}
	for rows.Next() {
		var t time.Time
		var rtt, loss float64
		if err := rows.Scan(&t, &rtt, &loss); err != nil {
			continue
		}
		data.RTT = append(data.RTT, MetricPoint{Time: t, Value: rtt})
		data.PacketLoss = append(data.PacketLoss, MetricPoint{Time: t, Value: loss})
	}

	jsonResponse(w, http.StatusOK, data)
}

// parsePeriod converts a period string to a PostgreSQL interval string.
func parsePeriod(p string) string {
	switch p {
	case "15m":
		return "15 minutes"
	case "1h":
		return "1 hour"
	case "6h":
		return "6 hours"
	case "24h", "1d":
		return "24 hours"
	case "7d":
		return "7 days"
	case "30d":
		return "30 days"
	default:
		return "1 hour"
	}
}
