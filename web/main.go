// snmpEndLog Web Server — Dashboard and API
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"snmpendlog-web/internal/auth"
	"snmpendlog-web/internal/db"
	"snmpendlog-web/internal/handlers"
	"snmpendlog-web/internal/middleware"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	log.Println("============================================================")
	log.Println("snmpEndLog Web Server starting...")
	log.Println("============================================================")

	// Initialize database
	if err := db.Init(); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()
	log.Println("Database connection pool initialized")

	// Initialize auth
	if err := auth.Init(); err != nil {
		log.Fatalf("Failed to initialize auth: %v", err)
	}
	log.Println("Auth module initialized")

	// Initialize templates
	if err := handlers.InitTemplates("templates"); err != nil {
		log.Fatalf("Failed to load templates: %v", err)
	}
	log.Println("Templates loaded")

	// Initialize rate limiter (5 attempts per minute for login)
	handlers.LoginLimiter = middleware.NewRateLimiter(5, time.Minute)

	// Setup routes
	mux := http.NewServeMux()

	// Static files
	fs := http.FileServer(http.Dir("static"))
	mux.Handle("/static/", http.StripPrefix("/static/", fs))

	// Public routes
	mux.HandleFunc("/login", handlers.HandleLogin)
	mux.HandleFunc("/logout", handlers.HandleLogout)

	// Protected page routes
	authMux := http.NewServeMux()
	authMux.HandleFunc("/", handlers.HandleDashboard)
	authMux.HandleFunc("/devices", handlers.HandleDevices)
	authMux.HandleFunc("/devices/add", handlers.HandleDeviceAdd)
	authMux.HandleFunc("/devices/edit", handlers.HandleDeviceEdit)
	authMux.HandleFunc("/devices/delete", handlers.HandleDeviceDelete)
	authMux.HandleFunc("/devices/", handlers.HandleDeviceDetail)
	authMux.HandleFunc("/logs", handlers.HandleLogs)

	mux.Handle("/", middleware.RequireAuth(authMux))

	// Protected API routes
	apiMux := http.NewServeMux()
	apiMux.HandleFunc("/api/metrics/traffic", handlers.HandleAPITraffic)
	apiMux.HandleFunc("/api/metrics/system", handlers.HandleAPISystem)
	apiMux.HandleFunc("/api/metrics/ping", handlers.HandleAPIPing)
	apiMux.HandleFunc("/api/logs", handlers.HandleAPILogs)
	apiMux.HandleFunc("/api/logs/stats", handlers.HandleAPILogStats)
	apiMux.HandleFunc("/api/logs/hosts", handlers.HandleAPILogHosts)

	mux.Handle("/api/", middleware.RequireAPI(apiMux))

	// Wrap with logging middleware
	handler := middleware.Logging(mux)

	// Start server
	port := os.Getenv("WEB_PORT")
	if port == "" {
		port = "8080"
	}
	addr := fmt.Sprintf(":%s", port)

	srv := &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh

		log.Println("Shutting down...")
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			log.Printf("Shutdown error: %v", err)
		}
	}()

	log.Printf("Web server listening on %s", addr)
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("Server error: %v", err)
	}

	log.Println("Server stopped")
}
