// nms Web Server — Dashboard and API
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

	"nms-web/internal/auth"
	"nms-web/internal/db"
	"nms-web/internal/handlers"
	"nms-web/internal/middleware"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	log.Println("============================================================")
	log.Println("nms Web Server starting...")
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

	// Protected page routes (all authenticated users)
	authMux := http.NewServeMux()
	authMux.HandleFunc("/", handlers.HandleDashboard)
	authMux.HandleFunc("/devices", handlers.HandleDevices)
	authMux.HandleFunc("/devices/", handlers.HandleDeviceDetail)
	authMux.HandleFunc("/logs", handlers.HandleLogs)
	authMux.HandleFunc("/ai", handlers.HandleAI)

	// Admin-only page routes (wrapped with RequireAdmin)
	adminMux := http.NewServeMux()
	adminMux.HandleFunc("/devices/add", handlers.HandleDeviceAdd)
	adminMux.HandleFunc("/devices/edit", handlers.HandleDeviceEdit)
	adminMux.HandleFunc("/devices/delete", handlers.HandleDeviceDelete)
	adminMux.HandleFunc("/users", handlers.HandleUsers)
	adminMux.HandleFunc("/users/create", handlers.HandleUserCreate)
	adminMux.HandleFunc("/users/delete", handlers.HandleUserDelete)
	adminMux.HandleFunc("/status", handlers.HandleStatus)

	authMux.Handle("/devices/add", middleware.RequireAdmin(adminMux))
	authMux.Handle("/devices/edit", middleware.RequireAdmin(adminMux))
	authMux.Handle("/devices/delete", middleware.RequireAdmin(adminMux))
	authMux.Handle("/users", middleware.RequireAdmin(adminMux))
	authMux.Handle("/users/", middleware.RequireAdmin(adminMux))
	authMux.Handle("/users/create", middleware.RequireAdmin(adminMux))
	authMux.Handle("/users/delete", middleware.RequireAdmin(adminMux))
	authMux.Handle("/status", middleware.RequireAdmin(adminMux))

	mux.Handle("/", middleware.RequireAuth(authMux))

	// Protected API routes
	apiMux := http.NewServeMux()
	apiMux.HandleFunc("/api/metrics/traffic", handlers.HandleAPITraffic)
	apiMux.HandleFunc("/api/metrics/system", handlers.HandleAPISystem)
	apiMux.HandleFunc("/api/metrics/ping", handlers.HandleAPIPing)
	apiMux.HandleFunc("/api/metrics/bgp", handlers.HandleAPIBGP)
	apiMux.HandleFunc("/api/logs", handlers.HandleAPILogs)
	apiMux.HandleFunc("/api/logs/stats", handlers.HandleAPILogStats)
	apiMux.HandleFunc("/api/logs/hosts", handlers.HandleAPILogHosts)
	apiMux.HandleFunc("/api/logs/export", handlers.HandleAPILogExport)
	apiMux.HandleFunc("/api/logs/export/txt", handlers.HandleAPILogExportTXT)
	apiMux.HandleFunc("/api/alarms", handlers.HandleAPIAlarms)
	apiMux.HandleFunc("/api/alarms/", handlers.HandleAPIAlarmResolve)
	apiMux.HandleFunc("/api/ai/sessions", handlers.HandleAISessions)
	apiMux.HandleFunc("/api/ai/sessions/", handlers.HandleAISessionAction)
	apiMux.HandleFunc("/api/status", handlers.HandleAPIStatus)

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
		WriteTimeout: 90 * time.Second,
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
