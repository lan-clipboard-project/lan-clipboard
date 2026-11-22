package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"lan-clipboard/config"
	"lan-clipboard/model"
	"lan-clipboard/store"
	"lan-clipboard/ws"
)

func main() {
	// Load configuration from environment variables
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	address := net.JoinHostPort(cfg.ListenHost, strconv.Itoa(cfg.Port))
	log.Printf("Starting LAN Clipboard server on %s", address)
	log.Printf("Data directory: %s", cfg.DataDir)
	log.Printf("Static directory: %s", cfg.StaticDir)
	log.Printf("Max file size: %d MB, Max text size: %d bytes", cfg.MaxFileSizeMB, cfg.MaxTextSize)
	log.Printf("Retention: %d days, Thumbnail threshold: %d MB", cfg.RetentionDays, cfg.ThumbnailThreshold)

	// Initialize SQLite store
	db, err := store.NewSQLiteStore(cfg.DataDir)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Initialize and start WebSocket hub
	hub := ws.NewHub()
	go hub.Run()

	// Start the cleanup scheduler (runs daily at 3 AM). Removed clips are
	// broadcast to connected clients so their lists stay in sync.
	store.StartCleanupScheduler(db, cfg, func(clips []model.Clip) {
		for _, clip := range clips {
			hub.BroadcastMessage(ws.Message{Type: "clip_deleted", Data: map[string]string{"id": clip.ID}})
		}
	})

	// Build the HTTP router (API + WebSocket + optional static frontend)
	router := newRouter(cfg, db, hub)

	srv := &http.Server{
		Addr:              address,
		Handler:           router,
		ReadHeaderTimeout: 15 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("Server listening on http://%s", address)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Graceful shutdown on SIGINT/SIGTERM
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited gracefully")
}
