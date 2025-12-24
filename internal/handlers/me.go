package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"predictionbot/internal/auth"
	"predictionbot/internal/logger"
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
		logger.Debug(0, "me_invalid_method", "method="+r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get user ID from context
	ctx := r.Context()
	telegramID, ok := auth.GetUserIDFromContext(ctx)
	if !ok {
		logger.Debug(0, "me_unauthorized", "path="+r.URL.Path)
		http.Error(w, "Unauthorized: user not in context", http.StatusUnauthorized)
		return
	}

	// Query user by Telegram ID
	user, err := storage.GetUserByTelegramID(telegramID)
	if err != nil {
		logger.Debug(telegramID, "me_error", "error="+err.Error())
		http.Error(w, "Failed to get user", http.StatusInternalServerError)
		return
	}
	if user == nil {
		logger.Debug(telegramID, "me_not_found", "")
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Format balance as integer
	balanceDisplay := fmt.Sprintf("%d", user.Balance)

	response := UserResponse{
		ID:             user.ID,
		TelegramID:     user.TelegramID,
		Username:       user.Username,
		FirstName:      user.FirstName,
		Balance:        user.Balance,
		BalanceDisplay: balanceDisplay,
	}

	logger.Debug(telegramID, "me_success", fmt.Sprintf("telegram_id=%d balance=%d", user.TelegramID, user.Balance))
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// HandleBailout handles the POST /api/me/bailout endpoint
// Users with balance < 1 can request a free bailout
// Bailout resets balance to 500
// 24-hour cooldown between bailouts
func HandleBailout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		logger.Debug(0, "bailout_invalid_method", "method="+r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get user ID from context
	ctx := r.Context()
	telegramID, ok := auth.GetUserIDFromContext(ctx)
	if !ok {
		logger.Debug(0, "bailout_unauthorized", "path="+r.URL.Path)
		http.Error(w, "Unauthorized: user not in context", http.StatusUnauthorized)
		return
	}

	// Get user to check balance
	user, err := storage.GetUserByTelegramID(telegramID)
	if err != nil {
		logger.Debug(telegramID, "bailout_error", "error="+err.Error())
		http.Error(w, "Failed to get user", http.StatusInternalServerError)
		return
	}
	if user == nil {
		logger.Debug(telegramID, "bailout_not_found", "")
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Check if user is eligible (balance < 1)
	if user.Balance >= storage.BailoutBalanceThreshold {
		logger.Debug(telegramID, "bailout_balance_too_high", fmt.Sprintf("balance=%d", user.Balance))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(storage.BailoutError{
			Error: "balance_too_high",
		})
		return
	}

	// Check cooldown (24 hours since last bailout)
	lastBailout, hasBailout, err := storage.GetLastBailout(user.ID)
	if err != nil {
		logger.Debug(telegramID, "bailout_check_error", "error="+err.Error())
		http.Error(w, "Failed to check bailout eligibility", http.StatusInternalServerError)
		return
	}
	if hasBailout {
		nextAvailable := lastBailout.Add(storage.BailoutCooldown)
		if time.Now().Before(nextAvailable) {
			remainingTime := nextAvailable.Sub(time.Now())
			hours := int(remainingTime.Hours())
			minutes := int(remainingTime.Minutes()) % 60
			logger.Debug(telegramID, "bailout_cooldown_active", fmt.Sprintf("next_available=%s", nextAvailable.Format(time.RFC3339)))
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			json.NewEncoder(w).Encode(storage.BailoutError{
				Error:         "cooldown_active",
				NextAvailable: fmt.Sprintf("Come back in %d hours %d minutes", hours, minutes),
			})
			return
		}
	}

	// Execute bailout
	newBalance, err := storage.ExecuteBailout(user.ID)
	if err != nil {
		// Check for specific errors
		if err.Error() == "balance_too_high: user has sufficient funds" {
			logger.Debug(telegramID, "bailout_balance_too_high", "")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(storage.BailoutError{
				Error: "balance_too_high",
			})
			return
		}
		if err.Error() == "cooldown_active: last bailout was at " {
			logger.Debug(telegramID, "bailout_cooldown_active", "")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			json.NewEncoder(w).Encode(storage.BailoutError{
				Error: "cooldown_active",
			})
			return
		}
		logger.Debug(telegramID, "bailout_execute_error", "error="+err.Error())
		http.Error(w, "Failed to execute bailout", http.StatusInternalServerError)
		return
	}

	logger.Debug(telegramID, "bailout_success", fmt.Sprintf("new_balance=%d", newBalance))
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(storage.BailoutResult{
		Message:    "Funds added",
		NewBalance: newBalance,
	})
}
