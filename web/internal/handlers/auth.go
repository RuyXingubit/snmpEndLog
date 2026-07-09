// Package handlers provides HTTP handlers for the web dashboard.
package handlers

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"time"

	"nms-web/internal/auth"
	"nms-web/internal/middleware"
)

// PageTemplates holds parsed HTML template sets, keyed by page name.
var PageTemplates map[string]*template.Template

// LoginLimiter is the rate limiter for login attempts.
var LoginLimiter *middleware.RateLimiter

// templateFuncs returns the shared template function map.
func templateFuncs() template.FuncMap {
	return template.FuncMap{
		"formatTime": func(t time.Time) string {
			return t.Format("02/01/2006 15:04:05")
		},
		"formatDuration": func(ticks int64) string {
			seconds := ticks / 100
			days := seconds / 86400
			hours := (seconds % 86400) / 3600
			minutes := (seconds % 3600) / 60
			if days > 0 {
				return fmt.Sprintf("%dd %dh %dm", days, hours, minutes)
			}
			if hours > 0 {
				return fmt.Sprintf("%dh %dm", hours, minutes)
			}
			return fmt.Sprintf("%dm", minutes)
		},
		"formatDurationSeconds": func(seconds int64) string {
			days := seconds / 86400
			hours := (seconds % 86400) / 3600
			minutes := (seconds % 3600) / 60
			if days > 0 {
				return fmt.Sprintf("%dd %dh %dm", days, hours, minutes)
			}
			if hours > 0 {
				return fmt.Sprintf("%dh %dm", hours, minutes)
			}
			return fmt.Sprintf("%dm", minutes)
		},
		"formatBps": func(bps float64) string {
			if bps >= 1_000_000_000 {
				return fmt.Sprintf("%.2f Gbps", bps/1_000_000_000)
			}
			if bps >= 1_000_000 {
				return fmt.Sprintf("%.2f Mbps", bps/1_000_000)
			}
			if bps >= 1_000 {
				return fmt.Sprintf("%.2f Kbps", bps/1_000)
			}
			return fmt.Sprintf("%.0f bps", bps)
		},
		"formatPercent": func(p *float64) string {
			if p == nil {
				return "N/A"
			}
			return fmt.Sprintf("%.1f%%", *p)
		},
		"severityClass": func(s string) string {
			switch s {
			case "emergency", "alert", "critical":
				return "severity-critical"
			case "error":
				return "severity-error"
			case "warning":
				return "severity-warning"
			case "notice", "info":
				return "severity-info"
			default:
				return "severity-debug"
			}
		},
		"deref": func(s *string) string {
			if s == nil {
				return ""
			}
			return *s
		},
		"derefTime": func(t *time.Time) time.Time {
			if t == nil {
				return time.Time{}
			}
			return *t
		},
		"derefInt64": func(i *int64) int64 {
			if i == nil {
				return 0
			}
			return *i
		},
		"derefInt": func(i *int) int {
			if i == nil {
				return 0
			}
			return *i
		},
		"derefFloat64": func(f *float64) float64 {
			if f == nil {
				return 0
			}
			return *f
		},
		"toFloat64": func(i int64) float64 {
			return float64(i)
		},
	}
}

// InitTemplates parses each page template individually with the layout.
func InitTemplates(dir string) error {
	PageTemplates = make(map[string]*template.Template)
	funcs := templateFuncs()
	layoutFile := filepath.Join(dir, "layout.html")

	// Pages that use the layout
	pages := []string{
		"dashboard.html",
		"devices.html",
		"device.html",
		"device_edit.html",
		"logs.html",
		"users.html",
		"ai.html",
		"status.html",
	}

	for _, page := range pages {
		t, err := template.New("").Funcs(funcs).ParseFiles(layoutFile, filepath.Join(dir, page))
		if err != nil {
			return fmt.Errorf("parse template %s: %w", page, err)
		}
		PageTemplates[page] = t
	}

	// Login page (no layout)
	loginTmpl, err := template.New("").Funcs(funcs).ParseFiles(filepath.Join(dir, "login.html"))
	if err != nil {
		return fmt.Errorf("parse login template: %w", err)
	}
	PageTemplates["login.html"] = loginTmpl

	return nil
}

// renderTemplate renders an HTML template with common data.
func renderTemplate(w http.ResponseWriter, name string, data map[string]interface{}, r *http.Request) {
	if data == nil {
		data = make(map[string]interface{})
	}

	// Add user info from context
	claims := middleware.GetClaims(r)
	if claims != nil {
		data["Username"] = claims.Username
		data["Role"] = claims.Role
	}

	tmpl, ok := PageTemplates[name]
	if !ok {
		log.Printf("Template %q not found", name)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	// For pages with layout, execute "layout.html"; for login, execute "login.html"
	execName := "layout.html"
	if name == "login.html" {
		execName = "login.html"
	}

	if err := tmpl.ExecuteTemplate(w, execName, data); err != nil {
		log.Printf("Template error rendering %s: %v", name, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// jsonResponse writes a JSON response.
func jsonResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("JSON encode error: %v", err)
	}
}

// HandleLogin renders the login page and processes login form.
func HandleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		renderTemplate(w, "login.html", nil, r)
		return
	}

	// Rate limit check
	ip := r.RemoteAddr
	if !LoginLimiter.Allow(ip) {
		renderTemplate(w, "login.html", map[string]interface{}{
			"Error": "Muitas tentativas. Aguarde alguns minutos.",
		}, r)
		return
	}

	username := r.FormValue("username")
	password := r.FormValue("password")

	token, err := auth.Authenticate(username, password)
	if err != nil {
		log.Printf("Login failed for user %q from %s", username, ip)
		renderTemplate(w, "login.html", map[string]interface{}{
			"Error": "Usuário ou senha inválidos.",
		}, r)
		return
	}

	// Set JWT cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    token,
		Path:     "/",
		MaxAge:   int(auth.TokenExpiry.Seconds()),
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteStrictMode,
	})

	log.Printf("Login successful for user %q from %s", username, ip)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// HandleLogout clears the auth cookie and redirects to login.
func HandleLogout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}
