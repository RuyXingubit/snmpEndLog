// Package handlers — User management handlers for admin users.
package handlers

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"

	"nms-web/internal/db"
	"nms-web/internal/middleware"
)

// UserEntry represents a user row for the listing template.
type UserEntry struct {
	ID        int
	Username  string
	Role      string
	CreatedAt time.Time
}

// HandleUsers renders the user management page (admin-only).
func HandleUsers(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r)
	if claims == nil || claims.Role != "admin" {
		http.Error(w, "Acesso negado", http.StatusForbidden)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	rows, err := db.Pool.Query(ctx, `SELECT id, username, role, created_at FROM users ORDER BY id`)
	if err != nil {
		log.Printf("Error listing users: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var users []UserEntry
	for rows.Next() {
		var u UserEntry
		if err := rows.Scan(&u.ID, &u.Username, &u.Role, &u.CreatedAt); err != nil {
			log.Printf("Error scanning user row: %v", err)
			continue
		}
		users = append(users, u)
	}

	data := map[string]interface{}{
		"Title": "Usuários",
		"Users": users,
	}

	if msg := r.URL.Query().Get("msg"); msg != "" {
		data["SuccessMsg"] = msg
	}
	if msg := r.URL.Query().Get("err"); msg != "" {
		data["ErrorMsg"] = msg
	}

	renderTemplate(w, "users.html", data, r)
}

// HandleUserCreate creates a new user (admin-only).
func HandleUserCreate(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r)
	if claims == nil || claims.Role != "admin" {
		http.Error(w, "Acesso negado", http.StatusForbidden)
		return
	}

	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/users", http.StatusSeeOther)
		return
	}

	username := strings.TrimSpace(r.FormValue("username"))
	password := r.FormValue("password")
	role := r.FormValue("role")

	// Validate username
	if len(username) < 3 || len(username) > 64 {
		http.Redirect(w, r, "/users?err=Nome+de+usuário+deve+ter+entre+3+e+64+caracteres.", http.StatusSeeOther)
		return
	}
	for _, ch := range username {
		if !unicode.IsLetter(ch) && !unicode.IsDigit(ch) && ch != '_' && ch != '-' && ch != '.' {
			http.Redirect(w, r, "/users?err=Nome+de+usuário+contém+caracteres+inválidos.+Use+letras,+números,+_,+-+ou+ponto.", http.StatusSeeOther)
			return
		}
	}

	// Validate password (minimum 8 chars)
	if len(password) < 8 {
		http.Redirect(w, r, "/users?err=A+senha+deve+ter+no+mínimo+8+caracteres.", http.StatusSeeOther)
		return
	}

	// Validate role
	if role != "admin" && role != "viewer" {
		http.Redirect(w, r, "/users?err=Role+inválida.", http.StatusSeeOther)
		return
	}

	// Hash password
	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		log.Printf("Error hashing password: %v", err)
		http.Redirect(w, r, "/users?err=Erro+interno+ao+criar+usuário.", http.StatusSeeOther)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// Check if username already exists
	var exists bool
	err = db.Pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE username = $1)`, username).Scan(&exists)
	if err != nil {
		log.Printf("Error checking user existence: %v", err)
		http.Redirect(w, r, "/users?err=Erro+interno+ao+criar+usuário.", http.StatusSeeOther)
		return
	}
	if exists {
		http.Redirect(w, r, "/users?err=Já+existe+um+usuário+com+esse+nome.", http.StatusSeeOther)
		return
	}

	_, err = db.Pool.Exec(ctx,
		`INSERT INTO users (username, password_hash, role) VALUES ($1, $2, $3)`,
		username, string(hash), role,
	)
	if err != nil {
		log.Printf("Error creating user: %v", err)
		http.Redirect(w, r, "/users?err=Erro+ao+criar+usuário.", http.StatusSeeOther)
		return
	}

	log.Printf("Admin %q created user %q with role %q", claims.Username, username, role)
	http.Redirect(w, r, "/users?msg=Usuário+"+username+"+criado+com+sucesso!", http.StatusSeeOther)
}

// HandleUserDelete removes a user (admin-only, cannot delete self).
func HandleUserDelete(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r)
	if claims == nil || claims.Role != "admin" {
		http.Error(w, "Acesso negado", http.StatusForbidden)
		return
	}

	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/users", http.StatusSeeOther)
		return
	}

	userIDStr := r.FormValue("user_id")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		http.Redirect(w, r, "/users?err=ID+de+usuário+inválido.", http.StatusSeeOther)
		return
	}

	// Prevent self-deletion
	if userID == claims.UserID {
		http.Redirect(w, r, "/users?err=Você+não+pode+remover+sua+própria+conta.", http.StatusSeeOther)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// Check the user exists and get their username for logging
	var targetUsername string
	err = db.Pool.QueryRow(ctx, `SELECT username FROM users WHERE id = $1`, userID).Scan(&targetUsername)
	if errors.Is(err, pgx.ErrNoRows) {
		http.Redirect(w, r, "/users?err=Usuário+não+encontrado.", http.StatusSeeOther)
		return
	}
	if err != nil {
		log.Printf("Error querying user for delete: %v", err)
		http.Redirect(w, r, "/users?err=Erro+interno.", http.StatusSeeOther)
		return
	}

	_, err = db.Pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, userID)
	if err != nil {
		log.Printf("Error deleting user: %v", err)
		http.Redirect(w, r, "/users?err=Erro+ao+remover+usuário.", http.StatusSeeOther)
		return
	}

	log.Printf("Admin %q deleted user %q (ID: %d)", claims.Username, targetUsername, userID)
	http.Redirect(w, r, "/users?msg=Usuário+"+targetUsername+"+removido.", http.StatusSeeOther)
}
