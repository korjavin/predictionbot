package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"predictionbot/internal/auth"
	"predictionbot/internal/logger"
	"predictionbot/internal/service"
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
		logger.Debug(0, "markets_invalid_method", "path="+r.URL.Path+" method="+r.Method)
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
		logger.Debug(0, "markets_create_unauthorized", "path="+r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(ErrorResponse{Message: "Unauthorized: user not in context"})
		return
	}

	// Decode request body
	var req CreateMarketRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Debug(userID, "markets_create_invalid_body", "error="+err.Error())
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Message: "Invalid request body"})
		return
	}

	// Validate question length (10-140 chars)
	if len(req.Question) < 10 || len(req.Question) > 140 {
		logger.Debug(userID, "markets_create_validation_failed", "question_length_invalid")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Message: "Question must be between 10 and 140 characters"})
		return
	}

	// Parse expires_at
	expiresAt, err := time.Parse(time.RFC3339, req.ExpiresAt)
	if err != nil {
		logger.Debug(userID, "markets_create_invalid_expiry", "expires_at="+req.ExpiresAt+" error="+err.Error())
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Message: "Invalid expires_at format. Use RFC3339 format (e.g., 2024-01-01T00:00:00Z)"})
		return
	}

	// Validate that expires_at is at least 1 hour in the future
	minExpiry := time.Now().Add(1 * time.Hour)
	if expiresAt.Before(minExpiry) {
		logger.Debug(userID, "markets_create_expiry_too_early", "expires_at="+req.ExpiresAt+" min_expiry="+minExpiry.Format(time.RFC3339))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Message: "Expiration must be at least 1 hour from now"})
		return
	}

	// Create the market
	market, err := storage.CreateMarket(userID, req.Question, expiresAt)
	if err != nil {
		questionPreview := req.Question
		if len(questionPreview) > 50 {
			questionPreview = questionPreview[:50]
		}
		logger.Debug(userID, "markets_create_failed", "question="+questionPreview+" error="+err.Error())
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ErrorResponse{Message: "Failed to create market"})
		return
	}

	questionPreview := req.Question
	if len(questionPreview) > 50 {
		questionPreview = questionPreview[:50]
	}
	logger.Debug(userID, "market_created", fmt.Sprintf("market_id=%d question=%s expires_at=%s", market.ID, questionPreview, expiresAt.Format(time.RFC3339)))
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
	// Get user ID from context (optional - markets are public but we log it for tracking)
	ctx := r.Context()
	userID, ok := auth.GetUserIDFromContext(ctx)

	markets, err := storage.ListActiveMarketsWithCreator()
	if err != nil {
		if ok {
			logger.Debug(userID, "markets_list_error", "error="+err.Error())
		} else {
			logger.Debug(0, "markets_list_error", "error="+err.Error())
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ErrorResponse{Message: "Failed to fetch markets"})
		return
	}

	// Get pool totals for each market
	for i := range markets {
		poolYes, poolNo, _ := storage.GetPoolTotals(markets[i].ID)
		markets[i].PoolYes = poolYes
		markets[i].PoolNo = poolNo
	}

	if ok {
		logger.Debug(userID, "markets_list_success", fmt.Sprintf("count=%d", len(markets)))
	} else {
		logger.Debug(0, "markets_list_success", fmt.Sprintf("count=%d", len(markets)))
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(markets)
}

// ResolveMarketRequest is the request body for resolving a market
type ResolveMarketRequest struct {
	Outcome string `json:"outcome"`
}

// ResolveMarketResponse is the response for resolving a market
type ResolveMarketResponse struct {
	Status           string `json:"status"`
	PayoutsProcessed int    `json:"payouts_processed"`
}

// HandleMarketResolve handles POST /api/markets/{id}/resolve
func HandleMarketResolve(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		logger.Debug(0, "resolve_invalid_method", "method="+r.Method+" path="+r.URL.Path)
		respondWithError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get user ID from context
	ctx := r.Context()
	userID, ok := auth.GetUserIDFromContext(ctx)
	if !ok {
		logger.Debug(0, "resolve_unauthorized", "path="+r.URL.Path)
		respondWithError(w, "Unauthorized: user not in context", http.StatusUnauthorized)
		return
	}

	// Parse market ID from URL path
	// Expected path: /api/markets/{id}/resolve (after StripPrefix removes /api)
	// So we get /markets/{id}/resolve
	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(pathParts) < 3 || pathParts[0] != "markets" || pathParts[2] != "resolve" {
		logger.Debug(userID, "resolve_invalid_path", "path="+r.URL.Path)
		respondWithError(w, "Invalid path format", http.StatusBadRequest)
		return
	}

	marketID, err := strconv.ParseInt(pathParts[1], 10, 64)
	if err != nil {
		logger.Debug(userID, "resolve_invalid_id", "id="+pathParts[1])
		respondWithError(w, "Invalid market ID", http.StatusBadRequest)
		return
	}

	// Parse request body
	var req ResolveMarketRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Debug(userID, "resolve_invalid_body", "error="+err.Error())
		respondWithError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate outcome
	if req.Outcome != "YES" && req.Outcome != "NO" {
		logger.Debug(userID, "resolve_invalid_outcome", "outcome="+req.Outcome)
		respondWithError(w, "Invalid outcome: must be 'YES' or 'NO'", http.StatusBadRequest)
		return
	}

	// Resolve the market using the payout service
	payoutService := service.NewPayoutService()
	payoutsProcessed, err := payoutService.ResolveMarket(ctx, marketID, userID, req.Outcome)
	if err != nil {
		errMsg := err.Error()
		logger.Debug(userID, "resolve_failed", fmt.Sprintf("market_id=%d error=%s", marketID, errMsg))
		if strings.Contains(errMsg, "not found") {
			respondWithError(w, errMsg, http.StatusNotFound)
		} else if strings.Contains(errMsg, "only the market creator") {
			respondWithError(w, errMsg, http.StatusForbidden)
		} else if strings.Contains(errMsg, "cannot be resolved") {
			respondWithError(w, errMsg, http.StatusConflict)
		} else if strings.Contains(errMsg, "invalid outcome") {
			respondWithError(w, errMsg, http.StatusBadRequest)
		} else {
			respondWithError(w, "Failed to resolve market", http.StatusInternalServerError)
		}
		return
	}

	logger.Debug(userID, "resolve_success", fmt.Sprintf("market_id=%d outcome=%s payouts=%d", marketID, req.Outcome, payoutsProcessed))
	response := ResolveMarketResponse{
		Status:           "finalized",
		PayoutsProcessed: payoutsProcessed,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}