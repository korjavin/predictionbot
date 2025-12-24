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
)

func main() {
	// Get port from environment or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Start bot in a goroutine
	go bot.StartBot()

	// Set up HTTP server with auth middleware
	mux := http.NewServeMux()

	// API routes with auth middleware
	apiMux := http.NewServeMux()
	apiMux.HandleFunc("/api/ping", handlers.PingHandler)

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
