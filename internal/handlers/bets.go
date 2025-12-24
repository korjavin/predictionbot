package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"predictionbot/internal/auth"
	"predictionbot/internal/logger"
	"predictionbot/internal/storage"
)

// PlaceBetRequest is the request body for placing a bet
type PlaceBetRequest struct {
	MarketID int64  `json:"market_id"`
	Outcome  string `json:"outcome"`
	Amount   int64  `json:"amount"`
}

// PlaceBetResponse is the response after placing a bet
type PlaceBetResponse struct {
	NewBalance int64 `json:"new_balance"`
	PoolYes    int64 `json:"pool_yes"`
	PoolNo     int64 `json:"pool_no"`
}

// HandleBets handles the POST /api/bets endpoint
func HandleBets(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		logger.Debug(0, "bets_invalid_method", "method="+r.Method+" path="+r.URL.Path)
		respondWithError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get user ID from context
	ctx := r.Context()
	userID, ok := auth.GetUserIDFromContext(ctx)
	if !ok {
		logger.Debug(0, "bets_unauthorized", "path="+r.URL.Path)
		respondWithError(w, "Unauthorized: user not in context", http.StatusUnauthorized)
		return
	}

	// Parse request body
	var req PlaceBetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Debug(userID, "bets_invalid_body", "error="+err.Error())
		respondWithError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Log bet attempt
	logger.Debug(userID, "bet_attempt", fmt.Sprintf("market_id=%d outcome=%s amount=%d", req.MarketID, req.Outcome, req.Amount))

	// Validate outcome
	if req.Outcome != "YES" && req.Outcome != "NO" {
		logger.Debug(userID, "bet_invalid_outcome", "outcome="+req.Outcome)
		respondWithError(w, "Invalid outcome: must be 'YES' or 'NO'", http.StatusBadRequest)
		return
	}

	// Validate amount
	if req.Amount <= 0 {
		logger.Debug(userID, "bet_invalid_amount", fmt.Sprintf("amount=%d", req.Amount))
		respondWithError(w, "Invalid amount: must be greater than 0", http.StatusBadRequest)
		return
	}

	// Place the bet
	if err := storage.PlaceBet(ctx, userID, req.MarketID, req.Outcome, req.Amount); err != nil {
		// Determine appropriate error code
		errMsg := err.Error()
		logger.Debug(userID, "bet_failed", "error="+errMsg)
		if strings.Contains(errMsg, "insufficient funds") {
			respondWithError(w, errMsg, http.StatusPaymentRequired)
		} else if strings.Contains(errMsg, "not active") || strings.Contains(errMsg, "expired") || strings.Contains(errMsg, "not found") {
			respondWithError(w, errMsg, http.StatusForbidden)
		} else if strings.Contains(errMsg, "invalid") {
			respondWithError(w, errMsg, http.StatusBadRequest)
		} else {
			respondWithError(w, "Failed to place bet", http.StatusInternalServerError)
		}
		return
	}

	// Get updated pool totals
	poolYes, poolNo, err := storage.GetPoolTotals(req.MarketID)
	if err != nil {
		logger.Debug(userID, "bet_pool_totals_error", "error="+err.Error())
		respondWithError(w, "Failed to get pool totals", http.StatusInternalServerError)
		return
	}

	// Get user's new balance
	user, err := storage.GetUserByID(userID)
	if err != nil || user == nil {
		logger.Debug(userID, "bet_balance_error", "error="+err.Error())
		respondWithError(w, "Failed to get user balance", http.StatusInternalServerError)
		return
	}

	response := PlaceBetResponse{
		NewBalance: user.Balance,
		PoolYes:    poolYes,
		PoolNo:     poolNo,
	}

	logger.Debug(userID, "bet_success", fmt.Sprintf("market_id=%d outcome=%s amount=%d new_balance=%d pool_yes=%d pool_no=%d", req.MarketID, req.Outcome, req.Amount, user.Balance, poolYes, poolNo))
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// respondWithError responds with an error in JSON format
func respondWithError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(ErrorResponse{Message: message})
}