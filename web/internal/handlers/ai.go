package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"nms-web/internal/ai"
	"nms-web/internal/db"
	"nms-web/internal/middleware"
)

// HandleAI renders the AI analysis page.
func HandleAI(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, "ai.html", map[string]interface{}{
		"Title": "Análise IA",
	}, r)
}

// HandleAISessions handles CRUD for AI sessions.
// GET  /api/ai/sessions        — list sessions
// POST /api/ai/sessions        — create session
func HandleAISessions(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r)
	if claims == nil {
		jsonResponse(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	switch r.Method {
	case http.MethodGet:
		rows, err := db.Pool.Query(ctx, `
			SELECT id, title, created_at, updated_at
			FROM ai_sessions
			ORDER BY updated_at DESC
			LIMIT 50
		`)
		if err != nil {
			jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		defer rows.Close()

		type Session struct {
			ID        int       `json:"id"`
			Title     string    `json:"title"`
			CreatedAt time.Time `json:"created_at"`
			UpdatedAt time.Time `json:"updated_at"`
		}

		var sessions []Session
		for rows.Next() {
			var s Session
			if err := rows.Scan(&s.ID, &s.Title, &s.CreatedAt, &s.UpdatedAt); err == nil {
				sessions = append(sessions, s)
			}
		}

		jsonResponse(w, http.StatusOK, sessions)

	case http.MethodPost:
		var body struct {
			Title string `json:"title"`
		}
		if err := parseJSON(r, &body); err != nil {
			body.Title = "Nova Análise"
		}
		if body.Title == "" {
			body.Title = "Nova Análise"
		}

		var sessionID int
		err := db.Pool.QueryRow(ctx,
			`INSERT INTO ai_sessions (title) VALUES ($1) RETURNING id`,
			body.Title,
		).Scan(&sessionID)
		if err != nil {
			jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}

		jsonResponse(w, http.StatusCreated, map[string]interface{}{
			"id":    sessionID,
			"title": body.Title,
		})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// HandleAISessionAction handles actions on a specific session.
// Routes: /api/ai/sessions/{id}, /api/ai/sessions/{id}/context, /api/ai/sessions/{id}/ask, /api/ai/sessions/{id}/messages
func HandleAISessionAction(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r)
	if claims == nil {
		jsonResponse(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	// Parse path: /api/ai/sessions/{id}[/action]
	path := strings.TrimPrefix(r.URL.Path, "/api/ai/sessions/")
	parts := strings.SplitN(path, "/", 2)

	sessionID, err := strconv.Atoi(parts[0])
	if err != nil {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid session id"})
		return
	}

	action := ""
	if len(parts) > 1 {
		action = parts[1]
	}

	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()

	switch {
	case r.Method == http.MethodDelete && action == "":
		handleDeleteSession(w, ctx, sessionID)
	case r.Method == http.MethodGet && action == "messages":
		handleGetMessages(w, ctx, sessionID)
	case r.Method == http.MethodPost && action == "context":
		handleAddContext(w, r, ctx, sessionID)
	case r.Method == http.MethodDelete && action == "context":
		handleClearContext(w, ctx, sessionID)
	case r.Method == http.MethodPost && action == "ask":
		handleAsk(w, r, ctx, sessionID)
	default:
		http.Error(w, "Not found", http.StatusNotFound)
	}
}

func handleDeleteSession(w http.ResponseWriter, ctx context.Context, sessionID int) {
	_, err := db.Pool.Exec(ctx, `DELETE FROM ai_sessions WHERE id = $1`, sessionID)
	if err != nil {
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	jsonResponse(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func handleClearContext(w http.ResponseWriter, ctx context.Context, sessionID int) {
	_, err := db.Pool.Exec(ctx, `DELETE FROM ai_messages WHERE session_id = $1`, sessionID)
	if err != nil {
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	jsonResponse(w, http.StatusOK, map[string]string{"status": "cleared"})
}

func handleGetMessages(w http.ResponseWriter, ctx context.Context, sessionID int) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, role, content, created_at
		FROM ai_messages
		WHERE session_id = $1
		ORDER BY created_at ASC
	`, sessionID)
	if err != nil {
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	defer rows.Close()

	type Msg struct {
		ID        int       `json:"id"`
		Role      string    `json:"role"`
		Content   string    `json:"content"`
		CreatedAt time.Time `json:"created_at"`
	}

	var messages []Msg
	for rows.Next() {
		var m Msg
		if err := rows.Scan(&m.ID, &m.Role, &m.Content, &m.CreatedAt); err == nil {
			messages = append(messages, m)
		}
	}

	jsonResponse(w, http.StatusOK, messages)
}

func handleAddContext(w http.ResponseWriter, r *http.Request, ctx context.Context, sessionID int) {
	var body struct {
		Host     string `json:"host"`
		Severity string `json:"severity"`
		Period   string `json:"period"`
		Search   string `json:"q"`
	}
	if err := parseJSON(r, &body); err != nil {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}

	period := parsePeriod(body.Period)

	// Build query for logs
	where := []string{"time > NOW() - $1::interval"}
	args := []interface{}{period}
	argIdx := 2

	if body.Host != "" {
		where = append(where, "host = $"+strconv.Itoa(argIdx))
		args = append(args, body.Host)
		argIdx++
	}

	if body.Severity != "" {
		where = append(where, "severity_name = $"+strconv.Itoa(argIdx))
		args = append(args, body.Severity)
		argIdx++
	}

	if body.Search != "" {
		where = append(where, "to_tsvector('simple', message) @@ plainto_tsquery('simple', $"+strconv.Itoa(argIdx)+")")
		args = append(args, body.Search)
		argIdx++
	}

	whereClause := strings.Join(where, " AND ")
	query := "SELECT time, host, COALESCE(severity_name, 'unknown'), COALESCE(app_name, ''), message FROM logs WHERE " +
		whereClause + " ORDER BY time DESC LIMIT 500"

	rows, err := db.Pool.Query(ctx, query, args...)
	if err != nil {
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	defer rows.Close()

	// Build context text from logs
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("=== Logs: host=%s severity=%s period=%s ===\n",
		orDefault(body.Host, "todos"), orDefault(body.Severity, "todas"), body.Period))

	count := 0
	for rows.Next() {
		var t time.Time
		var host, sev, app, msg string
		if err := rows.Scan(&t, &host, &sev, &app, &msg); err != nil {
			continue
		}
		sb.WriteString(fmt.Sprintf("[%s] %s %s %s: %s\n",
			t.Format("2006-01-02 15:04:05"), host, sev, app, msg))
		count++
	}

	if count == 0 {
		jsonResponse(w, http.StatusOK, map[string]interface{}{
			"count":   0,
			"message": "Nenhum log encontrado com os filtros selecionados.",
		})
		return
	}

	// Store context message
	_, err = db.Pool.Exec(ctx,
		`INSERT INTO ai_messages (session_id, role, content) VALUES ($1, 'context', $2)`,
		sessionID, sb.String(),
	)
	if err != nil {
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	// Update session timestamp
	db.Pool.Exec(ctx, `UPDATE ai_sessions SET updated_at = NOW() WHERE id = $1`, sessionID)

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"count":   count,
		"message": fmt.Sprintf("%d logs adicionados ao contexto.", count),
	})
}

func handleAsk(w http.ResponseWriter, r *http.Request, ctx context.Context, sessionID int) {
	var body struct {
		Question string `json:"question"`
	}
	if err := parseJSON(r, &body); err != nil || strings.TrimSpace(body.Question) == "" {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "question is required"})
		return
	}

	// Load all messages from this session
	rows, err := db.Pool.Query(ctx, `
		SELECT role, content FROM ai_messages
		WHERE session_id = $1
		ORDER BY created_at ASC
	`, sessionID)
	if err != nil {
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	defer rows.Close()

	var messages []ai.Message
	for rows.Next() {
		var m ai.Message
		if err := rows.Scan(&m.Role, &m.Content); err == nil {
			// Context messages are sent as user messages to Gemini
			if m.Role == "context" {
				m.Role = "user"
			}
			messages = append(messages, m)
		}
	}

	// Add the new question
	messages = append(messages, ai.Message{Role: "user", Content: body.Question})

	// Call Gemini
	response, err := ai.Analyze(messages)
	if err != nil {
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	// Store user question and AI response
	_, err = db.Pool.Exec(ctx,
		`INSERT INTO ai_messages (session_id, role, content) VALUES ($1, 'user', $2), ($1, 'assistant', $3)`,
		sessionID, body.Question, response,
	)
	if err != nil {
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	// Update session timestamp
	db.Pool.Exec(ctx, `UPDATE ai_sessions SET updated_at = NOW() WHERE id = $1`, sessionID)

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"response": response,
	})
}

func orDefault(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

// parseJSON decodes JSON from request body.
func parseJSON(r *http.Request, v interface{}) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(v)
}
