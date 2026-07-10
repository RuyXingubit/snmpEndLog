package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"nms-web/internal/db"
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
	CPU         []MetricPoint `json:"cpu"`
	Memory      []MetricPoint `json:"memory"`
	Temperature []MetricPoint `json:"temperature,omitempty"`
	PPPoE       []MetricPoint `json:"pppoe,omitempty"`
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
	tf := parseTimeFilter(r)

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	var where []string
	var args []interface{}
	where = append(where, "device_id = $1", "if_index = $2")
	args = append(args, deviceID, ifIndex)
	argIdx := 3
	where, args, _ = tf.AppendTimeWhere(where, args, argIdx)

	query := "SELECT time, COALESCE(in_bps, 0), COALESCE(out_bps, 0) FROM metric_traffic WHERE " +
		joinWhere(where) + " ORDER BY time ASC"

	rows, err := db.Pool.Query(ctx, query, args...)
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
	tf := parseTimeFilter(r)

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	var where []string
	var args []interface{}
	where = append(where, "device_id = $1")
	args = append(args, deviceID)
	argIdx := 2
	where, args, _ = tf.AppendTimeWhere(where, args, argIdx)

	query := "SELECT time, COALESCE(cpu_percent, 0), COALESCE(memory_percent, 0), pppoe_online, temperature FROM metric_system WHERE " +
		joinWhere(where) + " ORDER BY time ASC"

	rows, err := db.Pool.Query(ctx, query, args...)
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
		var temp *float64
		if err := rows.Scan(&t, &cpu, &mem, &pppoe, &temp); err != nil {
			continue
		}
		data.CPU = append(data.CPU, MetricPoint{Time: t, Value: cpu})
		data.Memory = append(data.Memory, MetricPoint{Time: t, Value: mem})
		if pppoe != nil {
			data.PPPoE = append(data.PPPoE, MetricPoint{Time: t, Value: float64(*pppoe)})
		}
		if temp != nil {
			data.Temperature = append(data.Temperature, MetricPoint{Time: t, Value: *temp})
		}
	}

	jsonResponse(w, http.StatusOK, data)
}

// HandleAPIPing returns ping metrics as JSON.
// GET /api/metrics/ping?device_id=1&period=1h
func HandleAPIPing(w http.ResponseWriter, r *http.Request) {
	deviceID, _ := strconv.Atoi(r.URL.Query().Get("device_id"))
	tf := parseTimeFilter(r)

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	var where []string
	var args []interface{}
	where = append(where, "device_id = $1")
	args = append(args, deviceID)
	argIdx := 2
	where, args, _ = tf.AppendTimeWhere(where, args, argIdx)

	query := "SELECT time, COALESCE(rtt_avg, 0), COALESCE(packet_loss, 0) FROM metric_ping WHERE " +
		joinWhere(where) + " ORDER BY time ASC"

	rows, err := db.Pool.Query(ctx, query, args...)
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

// BGPData represents BGP metrics for a chart.
type BGPData struct {
	State  []MetricPoint `json:"state"`
	Uptime []MetricPoint `json:"uptime"`
}

// HandleAPIBGP returns BGP metrics as JSON.
// GET /api/metrics/bgp?device_id=1&peer_addr=1.1.1.1&period=1h
func HandleAPIBGP(w http.ResponseWriter, r *http.Request) {
	deviceID, _ := strconv.Atoi(r.URL.Query().Get("device_id"))
	peerAddr := r.URL.Query().Get("peer_addr")
	tf := parseTimeFilter(r)

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	var where []string
	var args []interface{}
	where = append(where, "device_id = $1", "peer_addr = $2")
	args = append(args, deviceID, peerAddr)
	argIdx := 3
	where, args, _ = tf.AppendTimeWhere(where, args, argIdx)

	query := "SELECT time, COALESCE(state, 0), COALESCE(uptime, 0) FROM metric_bgp WHERE " +
		joinWhere(where) + " ORDER BY time ASC"

	rows, err := db.Pool.Query(ctx, query, args...)
	if err != nil {
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	defer rows.Close()

	data := BGPData{}
	for rows.Next() {
		var t time.Time
		var state, uptime float64
		if err := rows.Scan(&t, &state, &uptime); err != nil {
			continue
		}
		data.State = append(data.State, MetricPoint{Time: t, Value: state})
		data.Uptime = append(data.Uptime, MetricPoint{Time: t, Value: uptime})
	}

	jsonResponse(w, http.StatusOK, data)
}

// joinWhere joins WHERE conditions with AND.
func joinWhere(where []string) string {
	return strings.Join(where, " AND ")
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

// TimeFilter encapsulates period-based or custom date range filtering.
type TimeFilter struct {
	IsCustom bool
	Period   string    // PostgreSQL interval string (e.g. "1 hour")
	Start    time.Time // Only used when IsCustom is true
	End      time.Time // Only used when IsCustom is true
}

// parseTimeFilter builds a TimeFilter from HTTP query parameters.
// Supports both predefined periods (period=1h) and custom ranges (period=custom&start=...&end=...).
// Start/end are expected in ISO 8601 / RFC 3339 format or "2006-01-02T15:04" (datetime-local input).
func parseTimeFilter(r *http.Request) TimeFilter {
	q := r.URL.Query()
	period := q.Get("period")

	if period == "custom" {
		startStr := q.Get("start")
		endStr := q.Get("end")

		start, errS := parseFlexibleTime(startStr)
		end, errE := parseFlexibleTime(endStr)

		if errS == nil && errE == nil && !start.IsZero() && !end.IsZero() {
			return TimeFilter{IsCustom: true, Start: start, End: end}
		}
		// Fallback to 1 hour if dates are invalid
		return TimeFilter{Period: "1 hour"}
	}

	return TimeFilter{Period: parsePeriod(period)}
}

// parseTimeFilterFromBody builds a TimeFilter from JSON body fields.
func parseTimeFilterFromBody(period, startStr, endStr string) TimeFilter {
	if period == "custom" {
		start, errS := parseFlexibleTime(startStr)
		end, errE := parseFlexibleTime(endStr)

		if errS == nil && errE == nil && !start.IsZero() && !end.IsZero() {
			return TimeFilter{IsCustom: true, Start: start, End: end}
		}
		return TimeFilter{Period: "1 hour"}
	}

	return TimeFilter{Period: parsePeriod(period)}
}

// parseFlexibleTime parses time from multiple formats (ISO 8601, datetime-local input).
func parseFlexibleTime(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, fmt.Errorf("empty time string")
	}

	// Try RFC 3339 (e.g. "2026-07-10T17:00:00Z" or "2026-07-10T17:00:00-03:00")
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}

	// Try datetime-local format (e.g. "2026-07-10T17:00")
	if t, err := time.Parse("2006-01-02T15:04", s); err == nil {
		return t, nil
	}

	// Try date only (e.g. "2026-07-10")
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return t, nil
	}

	return time.Time{}, fmt.Errorf("unsupported time format: %s", s)
}

// AppendTimeWhere appends the time filter condition to a WHERE clause builder.
// Returns the updated where slice, args slice, and argIdx.
func (tf TimeFilter) AppendTimeWhere(where []string, args []interface{}, argIdx int) ([]string, []interface{}, int) {
	if tf.IsCustom {
		where = append(where, "time >= $"+strconv.Itoa(argIdx)+" AND time <= $"+strconv.Itoa(argIdx+1))
		args = append(args, tf.Start, tf.End)
		argIdx += 2
	} else {
		where = append(where, "time > NOW() - $"+strconv.Itoa(argIdx)+"::interval")
		args = append(args, tf.Period)
		argIdx++
	}
	return where, args, argIdx
}
