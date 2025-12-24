package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"predictionbot/internal/auth"
	"predictionbot/internal/storage"
)

func setupTestDB(t *testing.T) {
	if err := storage.InitDB(":memory:"); err != nil {
		t.Fatalf("Failed to initialize test database: %v", err)
	}
}

func cleanupTestDB(t *testing.T) {
	storage.CloseDB()
}

func TestPingHandler(t *testing.T) {
	req, err := http.NewRequest("GET", "/ping", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(PingHandler)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rr.Code)
	}

	var response map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response["status"] != "ok" {
		t.Errorf("Expected status 'ok', got '%s'", response["status"])
	}
}

func TestHandleMeUnauthorized(t *testing.T) {
	setupTestDB(t)
	defer cleanupTestDB(t)

	req, err := http.NewRequest("GET", "/me", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Don't add auth context - should fail
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(HandleMe)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, rr.Code)
	}
}

func TestHandleMeAuthorized(t *testing.T) {
	setupTestDB(t)
	defer cleanupTestDB(t)

	// Create a user first
	user, err := storage.CreateUser(12345, "testuser", "Test User")
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	req, err := http.NewRequest("GET", "/me", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Add auth context
	ctx := context.WithValue(req.Context(), auth.UserIDKey, user.ID)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(HandleMe)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rr.Code)
	}
}

func TestHandleCreateMarketUnauthorized(t *testing.T) {
	setupTestDB(t)
	defer cleanupTestDB(t)

	body := `{"question":"Will it rain?","expires_at":"2025-12-31T00:00:00Z"}`
	req, err := http.NewRequest("POST", "/markets", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(HandleMarkets)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, rr.Code)
	}
}

func TestHandleCreateMarketInvalidBody(t *testing.T) {
	setupTestDB(t)
	defer cleanupTestDB(t)

	user, _ := storage.CreateUser(12346, "testuser2", "Test User 2")

	body := `{"invalid json"`
	req, err := http.NewRequest("POST", "/markets", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")

	ctx := context.WithValue(req.Context(), auth.UserIDKey, user.ID)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(HandleMarkets)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}
}

func TestHandleCreateMarketShortQuestion(t *testing.T) {
	setupTestDB(t)
	defer cleanupTestDB(t)

	user, _ := storage.CreateUser(12347, "testuser3", "Test User 3")

	body := `{"question":"Short","expires_at":"2025-12-31T00:00:00Z"}`
	req, err := http.NewRequest("POST", "/markets", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")

	ctx := context.WithValue(req.Context(), auth.UserIDKey, user.ID)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(HandleMarkets)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}
}

func TestHandleCreateMarketInvalidExpiry(t *testing.T) {
	setupTestDB(t)
	defer cleanupTestDB(t)

	user, _ := storage.CreateUser(12348, "testuser4", "Test User 4")

	body := `{"question":"Will it rain tomorrow?","expires_at":"invalid-date"}`
	req, err := http.NewRequest("POST", "/markets", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")

	ctx := context.WithValue(req.Context(), auth.UserIDKey, user.ID)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(HandleMarkets)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}
}

func TestHandleCreateMarketPastExpiry(t *testing.T) {
	setupTestDB(t)
	defer cleanupTestDB(t)

	user, _ := storage.CreateUser(12349, "testuser5", "Test User 5")

	// Past date
	body := `{"question":"Will it rain tomorrow?","expires_at":"2020-12-31T00:00:00Z"}`
	req, err := http.NewRequest("POST", "/markets", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")

	ctx := context.WithValue(req.Context(), auth.UserIDKey, user.ID)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(HandleMarkets)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}
}

func TestHandleCreateMarketSuccess(t *testing.T) {
	setupTestDB(t)
	defer cleanupTestDB(t)

	user, _ := storage.CreateUser(12350, "testuser6", "Test User 6")

	// Future date
	futureDate := time.Now().Add(24 * time.Hour).Format(time.RFC3339)
	body := `{"question":"Will it rain tomorrow?","expires_at":"` + futureDate + `"}`
	req, err := http.NewRequest("POST", "/markets", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")

	ctx := context.WithValue(req.Context(), auth.UserIDKey, user.ID)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(HandleMarkets)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("Expected status %d, got %d", http.StatusCreated, rr.Code)
	}

	var response CreateMarketResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response.ID == 0 {
		t.Error("Expected non-zero market ID")
	}
}

func TestHandleListMarkets(t *testing.T) {
	setupTestDB(t)
	defer cleanupTestDB(t)

	// Create a user and market
	user, _ := storage.CreateUser(12351, "testuser7", "Test User 7")
	expiresAt := time.Now().Add(24 * time.Hour)
	_, _ = storage.CreateMarket(user.ID, "Test market?", expiresAt)

	req, err := http.NewRequest("GET", "/markets", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(HandleMarkets)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rr.Code)
	}
}

func TestHandleMarketsInvalidMethod(t *testing.T) {
	setupTestDB(t)
	defer cleanupTestDB(t)

	req, err := http.NewRequest("PUT", "/markets", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(HandleMarkets)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status %d, got %d", http.StatusMethodNotAllowed, rr.Code)
	}
}

func TestHandleBetsUnauthorized(t *testing.T) {
	setupTestDB(t)
	defer cleanupTestDB(t)

	body := `{"market_id":1,"outcome":"YES","amount":1000}`
	req, err := http.NewRequest("POST", "/bets", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(HandleBets)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, rr.Code)
	}
}

func TestHandleBetsInvalidBody(t *testing.T) {
	setupTestDB(t)
	defer cleanupTestDB(t)

	user, _ := storage.CreateUser(12352, "testuser8", "Test User 8")

	body := `{"invalid`
	req, err := http.NewRequest("POST", "/bets", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")

	ctx := context.WithValue(req.Context(), auth.UserIDKey, user.ID)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(HandleBets)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}
}

func TestHandleLeaderboard(t *testing.T) {
	setupTestDB(t)
	defer cleanupTestDB(t)

	// Create users
	_, _ = storage.CreateUser(12353, "user1", "User 1")
	_, _ = storage.CreateUser(12354, "user2", "User 2")

	req, err := http.NewRequest("GET", "/leaderboard", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(HandleLeaderboard)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rr.Code)
	}
}

func TestHandleUserBetsUnauthorized(t *testing.T) {
	setupTestDB(t)
	defer cleanupTestDB(t)

	req, err := http.NewRequest("GET", "/me/bets", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(HandleUserBets)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, rr.Code)
	}
}

func TestHandleUserStatsUnauthorized(t *testing.T) {
	setupTestDB(t)
	defer cleanupTestDB(t)

	req, err := http.NewRequest("GET", "/me/stats", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(HandleUserStats)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, rr.Code)
	}
}
