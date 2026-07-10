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

// LogEntry represents a syslog entry for templates and API.
type LogEntry struct {
	Time     time.Time `json:"time"`
	Host     string    `json:"host"`
	Severity string    `json:"severity"`
	AppName  string    `json:"app_name"`
	Message  string    `json:"message"`
}

// LogSearchResult holds paginated log search results.
type LogSearchResult struct {
	Logs       []LogEntry `json:"logs"`
	Total      int        `json:"total"`
	Page       int        `json:"page"`
	PerPage    int        `json:"per_page"`
	HasMore    bool       `json:"has_more"`
}

// SeverityCount represents log count by severity.
type SeverityCount struct {
	Severity string `json:"severity"`
	Count    int    `json:"count"`
}

// HandleLogs renders the log viewer page.
func HandleLogs(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, "logs.html", map[string]interface{}{
		"Title": "Logs",
	}, r)
}

// HandleAPILogs returns paginated, filtered logs as JSON.
// GET /api/logs?host=X&severity=error&q=text&period=1h&page=1&per_page=50
func HandleAPILogs(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	host := strings.TrimSpace(q.Get("host"))
	severity := strings.TrimSpace(q.Get("severity"))
	search := strings.TrimSpace(q.Get("q"))
	tf := parseTimeFilter(r)
	exact := q.Get("exact") == "true"
	page, _ := strconv.Atoi(q.Get("page"))
	perPage, _ := strconv.Atoi(q.Get("per_page"))

	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 200 {
		perPage = 50
	}
	offset := (page - 1) * perPage

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	// Build dynamic query
	var where []string
	var args []interface{}
	argIdx := 1
	where, args, argIdx = tf.AppendTimeWhere(where, args, argIdx)

	if host != "" {
		where = append(where, "host = $"+strconv.Itoa(argIdx))
		args = append(args, host)
		argIdx++
	}

	if severity != "" {
		where = append(where, "severity_name = $"+strconv.Itoa(argIdx))
		args = append(args, severity)
		argIdx++
	}

	if search != "" {
		if exact {
			// ILIKE for exact substring match — search in both message and app_name
			where = append(where, "(message ILIKE $"+strconv.Itoa(argIdx)+" OR app_name ILIKE $"+strconv.Itoa(argIdx)+")")
			args = append(args, "%"+search+"%")
		} else {
			// Full-text on message + ILIKE fallback on app_name
			where = append(where, "(to_tsvector('simple', message) @@ plainto_tsquery('simple', $"+strconv.Itoa(argIdx)+") OR app_name ILIKE $"+strconv.Itoa(argIdx+1)+")")
			args = append(args, search, "%"+search+"%")
			argIdx++ // extra arg for app_name ILIKE
		}
		argIdx++
	}

	whereClause := strings.Join(where, " AND ")

	// Count total
	var total int
	countQuery := "SELECT COUNT(*) FROM logs WHERE " + whereClause
	_ = db.Pool.QueryRow(ctx, countQuery, args...).Scan(&total)

	// Fetch page
	query := "SELECT time, host, COALESCE(severity_name, 'unknown'), COALESCE(app_name, ''), message FROM logs WHERE " +
		whereClause + " ORDER BY time DESC LIMIT $" + strconv.Itoa(argIdx) + " OFFSET $" + strconv.Itoa(argIdx+1)
	args = append(args, perPage, offset)

	rows, err := db.Pool.Query(ctx, query, args...)
	if err != nil {
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	defer rows.Close()

	var logs []LogEntry
	for rows.Next() {
		var entry LogEntry
		if err := rows.Scan(&entry.Time, &entry.Host, &entry.Severity, &entry.AppName, &entry.Message); err != nil {
			continue
		}
		logs = append(logs, entry)
	}

	jsonResponse(w, http.StatusOK, LogSearchResult{
		Logs:    logs,
		Total:   total,
		Page:    page,
		PerPage: perPage,
		HasMore: offset+perPage < total,
	})
}

// HandleAPILogStats returns log counts by severity for the dashboard.
// GET /api/logs/stats?period=1h
func HandleAPILogStats(w http.ResponseWriter, r *http.Request) {
	tf := parseTimeFilter(r)

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var where []string
	var args []interface{}
	argIdx := 1
	where, args, _ = tf.AppendTimeWhere(where, args, argIdx)

	query := "SELECT COALESCE(severity_name, 'unknown') AS sev, COUNT(*) FROM logs WHERE " +
		strings.Join(where, " AND ") + " GROUP BY sev ORDER BY COUNT(*) DESC"

	rows, err := db.Pool.Query(ctx, query, args...)
	if err != nil {
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	defer rows.Close()

	var stats []SeverityCount
	for rows.Next() {
		var sc SeverityCount
		if err := rows.Scan(&sc.Severity, &sc.Count); err != nil {
			continue
		}
		stats = append(stats, sc)
	}

	jsonResponse(w, http.StatusOK, stats)
}

// HandleAPILogHosts returns distinct hosts that have sent logs.
// GET /api/logs/hosts
func HandleAPILogHosts(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	rows, err := db.Pool.Query(ctx, `
		SELECT DISTINCT host FROM logs
		WHERE time > NOW() - INTERVAL '7 days'
		ORDER BY host
	`)
	if err != nil {
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	defer rows.Close()

	var hosts []string
	for rows.Next() {
		var h string
		if err := rows.Scan(&h); err == nil {
			hosts = append(hosts, h)
		}
	}

	jsonResponse(w, http.StatusOK, hosts)
}

// HandleAPILogExport streams filtered logs as CSV download.
// GET /api/logs/export?host=X&severity=error&q=text&period=1h
func HandleAPILogExport(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	host := strings.TrimSpace(q.Get("host"))
	severity := strings.TrimSpace(q.Get("severity"))
	search := strings.TrimSpace(q.Get("q"))
	tf := parseTimeFilter(r)
	exact := q.Get("exact") == "true"

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	// Build dynamic query
	var where []string
	var args []interface{}
	argIdx := 1
	where, args, argIdx = tf.AppendTimeWhere(where, args, argIdx)

	if host != "" {
		where = append(where, "host = $"+strconv.Itoa(argIdx))
		args = append(args, host)
		argIdx++
	}

	if severity != "" {
		where = append(where, "severity_name = $"+strconv.Itoa(argIdx))
		args = append(args, severity)
		argIdx++
	}

	if search != "" {
		if exact {
			where = append(where, "(message ILIKE $"+strconv.Itoa(argIdx)+" OR app_name ILIKE $"+strconv.Itoa(argIdx)+")")
			args = append(args, "%"+search+"%")
		} else {
			where = append(where, "(to_tsvector('simple', message) @@ plainto_tsquery('simple', $"+strconv.Itoa(argIdx)+") OR app_name ILIKE $"+strconv.Itoa(argIdx+1)+")")
			args = append(args, search, "%"+search+"%")
			argIdx++
		}
		argIdx++
	}

	whereClause := strings.Join(where, " AND ")
	query := "SELECT time, host, COALESCE(severity_name, 'unknown'), COALESCE(app_name, ''), message FROM logs WHERE " +
		whereClause + " ORDER BY time DESC LIMIT 10000"

	rows, err := db.Pool.Query(ctx, query, args...)
	if err != nil {
		http.Error(w, "Query error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=logs_export.csv")

	// BOM for Excel UTF-8 support
	w.Write([]byte("\xEF\xBB\xBF"))
	w.Write([]byte("Timestamp,Host,Severity,App,Message\r\n"))

	for rows.Next() {
		var t time.Time
		var h, sev, app, msg string
		if err := rows.Scan(&t, &h, &sev, &app, &msg); err != nil {
			continue
		}

		// Escape CSV fields (double-quote fields containing commas/quotes/newlines)
		escapeCsv := func(s string) string {
			if strings.ContainsAny(s, ",\"\r\n") {
				return "\"" + strings.ReplaceAll(s, "\"", "\"\"") + "\""
			}
			return s
		}

		line := t.Format("2006-01-02 15:04:05") + "," +
			escapeCsv(h) + "," +
			escapeCsv(sev) + "," +
			escapeCsv(app) + "," +
			escapeCsv(msg) + "\r\n"
		w.Write([]byte(line))
	}
}

// HandleAPILogExportTXT streams filtered logs as plain text download.
// GET /api/logs/export/txt?host=X&severity=error&q=text&period=1h&exact=true
func HandleAPILogExportTXT(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	host := strings.TrimSpace(q.Get("host"))
	severity := strings.TrimSpace(q.Get("severity"))
	search := strings.TrimSpace(q.Get("q"))
	tf := parseTimeFilter(r)
	exact := q.Get("exact") == "true"

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	// Build dynamic query
	var where []string
	var args []interface{}
	argIdx := 1
	where, args, argIdx = tf.AppendTimeWhere(where, args, argIdx)

	if host != "" {
		where = append(where, "host = $"+strconv.Itoa(argIdx))
		args = append(args, host)
		argIdx++
	}

	if severity != "" {
		where = append(where, "severity_name = $"+strconv.Itoa(argIdx))
		args = append(args, severity)
		argIdx++
	}

	if search != "" {
		if exact {
			where = append(where, "(message ILIKE $"+strconv.Itoa(argIdx)+" OR app_name ILIKE $"+strconv.Itoa(argIdx)+")")
			args = append(args, "%"+search+"%")
		} else {
			where = append(where, "(to_tsvector('simple', message) @@ plainto_tsquery('simple', $"+strconv.Itoa(argIdx)+") OR app_name ILIKE $"+strconv.Itoa(argIdx+1)+")")
			args = append(args, search, "%"+search+"%")
			argIdx++
		}
		argIdx++
	}

	whereClause := strings.Join(where, " AND ")
	query := "SELECT time, host, COALESCE(severity_name, 'unknown'), COALESCE(app_name, ''), message FROM logs WHERE " +
		whereClause + " ORDER BY time DESC LIMIT 10000"

	rows, err := db.Pool.Query(ctx, query, args...)
	if err != nil {
		http.Error(w, "Query error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=logs_export.txt")

	for rows.Next() {
		var t time.Time
		var h, sev, app, msg string
		if err := rows.Scan(&t, &h, &sev, &app, &msg); err != nil {
			continue
		}

		line := fmt.Sprintf("[%s] %s %s %s: %s\n",
			t.Format("2006-01-02 15:04:05"), h, sev, app, msg)
		w.Write([]byte(line))
	}
}
