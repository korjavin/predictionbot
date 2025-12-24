package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"predictionbot/internal/auth"
	"predictionbot/internal/storage"
)

// CreateMarketRequest is the request body for creating a market
type CreateMarketRequest struct {
	Question  string `json:"question"`
	ExpiresAt string `json:"expires_at"`
}

// CreateMarketResponse is the response for creating a market
type CreateMarketResponse struct {
	ID     int64  `json:"id"`
	Status string `json:"status"`
}

// ErrorResponse is the standard error response format
type ErrorResponse struct {
	Message string `json:"message"`
}

// HandleMarkets routes between GET and POST for /api/markets
func HandleMarkets(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		handleCreateMarket(w, r)
	case http.MethodGet:
		handleListMarkets(w, r)
	default:
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(ErrorResponse{Message: "Method not allowed"})
	}
}

// handleCreateMarket handles POST /api/markets
func handleCreateMarket(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	ctx := r.Context()
	userID, ok := auth.GetUserIDFromContext(ctx)
	if !ok {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(ErrorResponse{Message: "Unauthorized: user not in context"})
		return
	}

	// Decode request body
	var req CreateMarketRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Message: "Invalid request body"})
		return
	}

	// Validate question length (10-140 chars)
	if len(req.Question) < 10 || len(req.Question) > 140 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Message: "Question must be between 10 and 140 characters"})
		return
	}

	// Parse expires_at
	expiresAt, err := time.Parse(time.RFC3339, req.ExpiresAt)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Message: "Invalid expires_at format. Use RFC3339 format (e.g., 2024-01-01T00:00:00Z)"})
		return
	}

	// Validate that expires_at is at least 1 hour in the future
	minExpiry := time.Now().Add(1 * time.Hour)
	if expiresAt.Before(minExpiry) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Message: "Expiration must be at least 1 hour from now"})
		return
	}

	// Create the market
	market, err := storage.CreateMarket(userID, req.Question, expiresAt)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ErrorResponse{Message: "Failed to create market"})
		return
	}

	response := CreateMarketResponse{
		ID:     market.ID,
		Status: string(market.Status),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// handleListMarkets handles GET /api/markets
func handleListMarkets(w http.ResponseWriter, r *http.Request) {
	// Get active markets with creator names
	markets, err := storage.ListActiveMarketsWithCreator()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ErrorResponse{Message: "Failed to fetch markets"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(markets)
}
