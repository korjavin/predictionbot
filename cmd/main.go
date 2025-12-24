package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"predictionbot/internal/auth"
	"predictionbot/internal/bot"
	"predictionbot/internal/handlers"
	"predictionbot/internal/service"
	"predictionbot/internal/storage"
)

func main() {
	// Get port from environment or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Initialize SQLite database
	dbPath := os.Getenv("DATABASE_PATH")
	if dbPath == "" {
		dbPath = "/app/data/market.db"
	}
	log.Printf("Initializing database at: %s", dbPath)
	if err := storage.InitDB(dbPath); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer storage.CloseDB()

	// Start bot in a goroutine
	go bot.StartBot()

	// Initialize notification service for Telegram messages
	notificationService, err := service.NewNotificationService()
	if err != nil {
		log.Printf("Warning: Failed to initialize notification service: %v", err)
		// Continue without notifications - the bot can still function
	} else {
		log.Println("Notification service initialized")
		// Set global notification service for use in handlers
		service.SetNotificationService(notificationService)
	}

	// Start market worker for auto-locking expired markets
	marketWorker := service.NewMarketWorker()
	marketWorker.Start()
	defer marketWorker.Stop()

	// Wire up notification service to market worker (for auto-finalization)
	if notificationService != nil {
		marketWorker.SetNotificationService(notificationService)
	}

	// Set up HTTP server with auth middleware
	mux := http.NewServeMux()

	// API routes with auth middleware
	apiMux := http.NewServeMux()
	apiMux.HandleFunc("/ping", handlers.PingHandler)
	apiMux.HandleFunc("/me", handlers.HandleMe)
	apiMux.HandleFunc("/me/bets", handlers.HandleUserBets)
	apiMux.HandleFunc("/me/stats", handlers.HandleUserStats)
	apiMux.HandleFunc("/me/bailout", handlers.HandleBailout)
	apiMux.HandleFunc("/leaderboard", handlers.HandleLeaderboard)
	apiMux.HandleFunc("/markets", handlers.HandleMarkets)
	// Use a single handler for /markets/{id}/resolve and /markets/{id}/dispute
	apiMux.HandleFunc("/markets/", handlers.HandleMarketSubpath)
	apiMux.HandleFunc("/admin/resolve", handlers.HandleAdminResolve) // Handles /api/admin/resolve
	apiMux.HandleFunc("/bets", handlers.HandleBets)

	// Apply auth middleware to API routes (except ping for testing)
	mux.Handle("/api/", auth.Middleware(http.StripPrefix("/api", apiMux)))

	// Static file serving (web directory)
	mux.Handle("/", http.FileServer(http.Dir("./web")))

	// Start server
	addr := fmt.Sprintf(":%s", port)
	log.Printf("Server starting on %s", addr)

	// Graceful shutdown
	go func() {
		if err := http.ListenAndServe(addr, mux); err != nil {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
}
