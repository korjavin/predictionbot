package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"predictionbot/internal/auth"
	"predictionbot/internal/logger"
	"predictionbot/internal/storage"
)

// HandleUserBets handles the GET /api/me/bets endpoint
func HandleUserBets(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		logger.Debug(0, "user_bets_invalid_method", "method="+r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get user ID from context
	ctx := r.Context()
	telegramID, ok := auth.GetUserIDFromContext(ctx)
	if !ok {
		logger.Debug(0, "user_bets_unauthorized", "path="+r.URL.Path)
		http.Error(w, "Unauthorized: user not in context", http.StatusUnauthorized)
		return
	}

	// Get user by Telegram ID to retrieve internal user ID
	user, err := storage.GetUserByTelegramID(telegramID)
	if err != nil || user == nil {
		logger.Debug(telegramID, "user_bets_user_not_found", "error=user lookup failed")
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Get user's bets using internal user ID
	bets, err := storage.GetUserBets(user.ID)
	if err != nil {
		logger.Debug(telegramID, "user_bets_error", "error="+err.Error())
		http.Error(w, "Failed to get user bets", http.StatusInternalServerError)
		return
	}

	logger.Debug(telegramID, "user_bets_success", fmt.Sprintf("count=%d", len(bets)))
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(bets)
}

// HandleUserStats handles the GET /api/me/stats endpoint
func HandleUserStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		logger.Debug(0, "user_stats_invalid_method", "method="+r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get user ID from context
	ctx := r.Context()
	telegramID, ok := auth.GetUserIDFromContext(ctx)
	if !ok {
		logger.Debug(0, "user_stats_unauthorized", "path="+r.URL.Path)
		http.Error(w, "Unauthorized: user not in context", http.StatusUnauthorized)
		return
	}

	// Get user by Telegram ID to retrieve internal user ID
	user, err := storage.GetUserByTelegramID(telegramID)
	if err != nil || user == nil {
		logger.Debug(telegramID, "user_stats_user_not_found", "error=user lookup failed")
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Get user's stats using internal user ID
	stats, err := storage.GetUserStats(user.ID)
	if err != nil {
		logger.Debug(telegramID, "user_stats_error", "error="+err.Error())
		http.Error(w, "Failed to get user stats", http.StatusInternalServerError)
		return
	}

	logger.Debug(telegramID, "user_stats_success", fmt.Sprintf("total_bets=%d wins=%d win_rate=%.2f", stats.TotalBets, stats.Wins, stats.WinRate))
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(stats)
}
