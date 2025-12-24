package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"predictionbot/internal/logger"
	"predictionbot/internal/storage"
)

// HandleLeaderboard handles GET /api/leaderboard
func HandleLeaderboard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		logger.Debug(0, "leaderboard_invalid_method", "method="+r.Method+" path="+r.URL.Path)
		respondWithError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get top 20 users by balance
	leaderboard, err := storage.GetTopUsers(20)
	if err != nil {
		logger.Debug(0, "leaderboard_error", "error="+err.Error())
		respondWithError(w, "Failed to fetch leaderboard", http.StatusInternalServerError)
		return
	}

	logger.Debug(0, "leaderboard_success", fmt.Sprintf("count=%d", len(leaderboard)))
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(leaderboard)
}
