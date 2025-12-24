package handlers

import (
	"encoding/json"
	"net/http"

	"predictionbot/internal/auth"
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
		respondWithError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get user ID from context
	ctx := r.Context()
	userID, ok := auth.GetUserIDFromContext(ctx)
	if !ok {
		respondWithError(w, "Unauthorized: user not in context", http.StatusUnauthorized)
		return
	}

	// Parse request body
	var req PlaceBetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate outcome
	if req.Outcome != "YES" && req.Outcome != "NO" {
		respondWithError(w, "Invalid outcome: must be 'YES' or 'NO'", http.StatusBadRequest)
		return
	}

	// Validate amount
	if req.Amount <= 0 {
		respondWithError(w, "Invalid amount: must be greater than 0", http.StatusBadRequest)
		return
	}

	// Place the bet
	if err := storage.PlaceBet(ctx, userID, req.MarketID, req.Outcome, req.Amount); err != nil {
		// Determine appropriate error code
		errMsg := err.Error()
		switch {
		case contains(errMsg, "insufficient funds"):
			respondWithError(w, errMsg, http.StatusPaymentRequired)
		case contains(errMsg, "not active") || contains(errMsg, "expired") || contains(errMsg, "not found"):
			respondWithError(w, errMsg, http.StatusForbidden)
		case contains(errMsg, "invalid"):
			respondWithError(w, errMsg, http.StatusBadRequest)
		default:
			respondWithError(w, "Failed to place bet", http.StatusInternalServerError)
		}
		return
	}

	// Get updated pool totals
	poolYes, poolNo, err := storage.GetPoolTotals(req.MarketID)
	if err != nil {
		respondWithError(w, "Failed to get pool totals", http.StatusInternalServerError)
		return
	}

	// Get user's new balance
	user, err := storage.GetUserByID(userID)
	if err != nil || user == nil {
		respondWithError(w, "Failed to get user balance", http.StatusInternalServerError)
		return
	}

	response := PlaceBetResponse{
		NewBalance: user.Balance,
		PoolYes:    poolYes,
		PoolNo:     poolNo,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// respondWithError sends a JSON error response
func respondWithError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
