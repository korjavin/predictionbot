package handlers

import (
	"encoding/json"
	"net/http"
)

// PingResponse is the response for the ping endpoint
type PingResponse struct {
	Status string `json:"status"`
}

// PingHandler handles the /api/ping endpoint
func PingHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	response := PingResponse{
		Status: "ok",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}
