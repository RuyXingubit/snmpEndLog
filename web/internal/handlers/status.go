package handlers

import (
	"context"
	"net/http"
	"time"

	"nms-web/internal/db"
)

// HandleStatus renders the system status page.
func HandleStatus(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, "status.html", map[string]interface{}{
		"Title": "Status",
	}, r)
}

// HandleAPIStatus returns system health metrics.
func HandleAPIStatus(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	status := map[string]interface{}{}

	// --- Database size ---
	var dbSizeBytes int64
	var dbSizeHuman string
	err := db.Pool.QueryRow(ctx,
		`SELECT pg_database_size(current_database()), pg_size_pretty(pg_database_size(current_database()))`,
	).Scan(&dbSizeBytes, &dbSizeHuman)
	if err == nil {
		status["db_size_bytes"] = dbSizeBytes
		status["db_size"] = dbSizeHuman
	}

	// --- Table sizes ---
	type TableInfo struct {
		Name     string `json:"name"`
		Rows     int64  `json:"rows"`
		Size     string `json:"size"`
		SizeByte int64  `json:"size_bytes"`
	}
	var tables []TableInfo

	rows, err := db.Pool.Query(ctx, `
		SELECT
			relname AS name,
			n_live_tup AS rows,
			pg_size_pretty(pg_total_relation_size(relid)) AS size,
			pg_total_relation_size(relid) AS size_bytes
		FROM pg_stat_user_tables
		ORDER BY pg_total_relation_size(relid) DESC
		LIMIT 20
	`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var t TableInfo
			if err := rows.Scan(&t.Name, &t.Rows, &t.Size, &t.SizeByte); err == nil {
				tables = append(tables, t)
			}
		}
	}
	status["tables"] = tables

	// --- Hypertable details (TimescaleDB) ---
	type HypertableInfo struct {
		Name              string `json:"name"`
		TotalSize         string `json:"total_size"`
		NumChunks         int    `json:"num_chunks"`
		CompressionStatus string `json:"compression_status"`
	}
	var hypertables []HypertableInfo

	htRows, err := db.Pool.Query(ctx, `
		SELECT
			ht.table_name,
			pg_size_pretty(hypertable_size(format('%I.%I', ht.schema_name, ht.table_name)::regclass)),
			(SELECT count(*) FROM timescaledb_information.chunks c WHERE c.hypertable_name = ht.table_name),
			COALESCE(ht.compression_state::text, 'off')
		FROM timescaledb_information.hypertables ht
		ORDER BY hypertable_size(format('%I.%I', ht.schema_name, ht.table_name)::regclass) DESC
	`)
	if err == nil {
		defer htRows.Close()
		for htRows.Next() {
			var h HypertableInfo
			if err := htRows.Scan(&h.Name, &h.TotalSize, &h.NumChunks, &h.CompressionStatus); err == nil {
				hypertables = append(hypertables, h)
			}
		}
	}
	status["hypertables"] = hypertables

	// --- Logs rate (last hour) ---
	var logsLastHour int64
	db.Pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM logs WHERE time > NOW() - INTERVAL '1 hour'`,
	).Scan(&logsLastHour)
	status["logs_last_hour"] = logsLastHour

	// --- Metrics rate (last hour) ---
	var metricsLastHour int64
	db.Pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM interface_metrics WHERE time > NOW() - INTERVAL '1 hour'`,
	).Scan(&metricsLastHour)
	status["metrics_last_hour"] = metricsLastHour

	// --- Total devices ---
	var totalDevices int64
	db.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM devices`).Scan(&totalDevices)
	status["total_devices"] = totalDevices

	// --- Total log entries ---
	var totalLogs int64
	db.Pool.QueryRow(ctx,
		`SELECT COALESCE(n_live_tup, 0) FROM pg_stat_user_tables WHERE relname = 'logs'`,
	).Scan(&totalLogs)
	status["total_logs"] = totalLogs

	// --- AI sessions ---
	var totalSessions int64
	var totalAIMessages int64
	db.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM ai_sessions`).Scan(&totalSessions)
	db.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM ai_messages`).Scan(&totalAIMessages)
	status["ai_sessions"] = totalSessions
	status["ai_messages"] = totalAIMessages

	// --- PostgreSQL connections ---
	var activeConns int64
	db.Pool.QueryRow(ctx,
		`SELECT count(*) FROM pg_stat_activity WHERE datname = current_database()`,
	).Scan(&activeConns)
	status["db_connections"] = activeConns

	// --- Server uptime (DB) ---
	var dbUptime string
	db.Pool.QueryRow(ctx,
		`SELECT date_trunc('second', NOW() - pg_postmaster_start_time())::text`,
	).Scan(&dbUptime)
	status["db_uptime"] = dbUptime

	jsonResponse(w, http.StatusOK, status)
}
