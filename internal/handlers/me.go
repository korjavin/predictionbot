package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"predictionbot/internal/auth"
	"predictionbot/internal/storage"
)

// UserResponse is the response for the /api/me endpoint
type UserResponse struct {
	ID             int64  `json:"id"`
	TelegramID     int64  `json:"telegram_id"`
	Username       string `json:"username"`
	FirstName      string `json:"first_name"`
	Balance        int64  `json:"balance"`
	BalanceDisplay string `json:"balance_display"`
}

// HandleMe handles the GET /api/me endpoint
func HandleMe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get user ID from context
	ctx := r.Context()
	userID, ok := auth.GetUserIDFromContext(ctx)
	if !ok {
		http.Error(w, "Unauthorized: user not in context", http.StatusUnauthorized)
		return
	}

	// Query user by internal ID
	user, err := storage.GetUserByID(userID)
	if err != nil {
		http.Error(w, "Failed to get user", http.StatusInternalServerError)
		return
	}
	if user == nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Calculate balance display (convert cents to formatted string)
	balanceDisplay := fmt.Sprintf("%.2f", float64(user.Balance)/100.0)

	response := UserResponse{
		ID:             user.ID,
		TelegramID:     user.TelegramID,
		Username:       user.Username,
		FirstName:      user.FirstName,
		Balance:        user.Balance,
		BalanceDisplay: balanceDisplay,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}
