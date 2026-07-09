// Package auth provides JWT-based authentication with bcrypt password hashing.
package auth

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"

	"nms-web/internal/db"
)

// BcryptCost is the bcrypt hashing cost (≥12 for security).
const BcryptCost = 12

// TokenExpiry is how long a JWT token is valid.
const TokenExpiry = 24 * time.Hour

// Claims represents JWT token claims.
type Claims struct {
	UserID   int    `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

// User represents a user record from the database.
type User struct {
	ID           int
	Username     string
	PasswordHash string
	Role         string
}

// jwtSecret is loaded from environment.
var jwtSecret []byte

// Init initializes the auth module and creates the admin user if needed.
func Init() error {
	secret := os.Getenv("JWT_SECRET")
	if len(secret) < 32 {
		return errors.New("JWT_SECRET must be at least 32 characters")
	}
	jwtSecret = []byte(secret)

	// Create admin user if no users exist
	return ensureAdminUser()
}

func ensureAdminUser() error {
	ctx := context.Background()

	var count int
	err := db.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil {
		return fmt.Errorf("count users: %w", err)
	}

	if count > 0 {
		return nil // Users already exist
	}

	username := os.Getenv("ADMIN_USER")
	password := os.Getenv("ADMIN_PASSWORD")
	if username == "" {
		username = "admin"
	}
	if password == "" {
		return errors.New("ADMIN_PASSWORD is required for initial setup")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), BcryptCost)
	if err != nil {
		return fmt.Errorf("hash admin password: %w", err)
	}

	_, err = db.Pool.Exec(ctx,
		"INSERT INTO users (username, password_hash, role) VALUES ($1, $2, 'admin')",
		username, string(hash),
	)
	if err != nil {
		return fmt.Errorf("create admin user: %w", err)
	}

	log.Printf("Created initial admin user: %s", username)
	return nil
}

// Authenticate verifies credentials and returns a JWT token.
func Authenticate(username, password string) (string, error) {
	ctx := context.Background()

	var user User
	err := db.Pool.QueryRow(ctx,
		"SELECT id, username, password_hash, role FROM users WHERE username = $1",
		username,
	).Scan(&user.ID, &user.Username, &user.PasswordHash, &user.Role)

	if errors.Is(err, pgx.ErrNoRows) {
		// Constant-time comparison even on missing user to prevent timing attacks
		bcrypt.CompareHashAndPassword(
			[]byte("$2a$12$000000000000000000000000000000000000000000000000000000"),
			[]byte(password),
		)
		return "", errors.New("invalid credentials")
	}
	if err != nil {
		return "", fmt.Errorf("query user: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "", errors.New("invalid credentials")
	}

	// Generate JWT
	now := time.Now()
	claims := &Claims{
		UserID:   user.ID,
		Username: user.Username,
		Role:     user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(TokenExpiry)),
			IssuedAt:  jwt.NewNumericDate(now),
			Subject:   fmt.Sprintf("%d", user.ID),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}

	return tokenString, nil
}

// ValidateToken validates a JWT token and returns the claims.
func ValidateToken(tokenString string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}
