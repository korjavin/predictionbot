package storage

import (
	"context"
	"strings"
	"testing"
	"time"
)

func setupTestDB(t *testing.T) {
	// Use in-memory database for tests
	if err := InitDB(":memory:"); err != nil {
		t.Fatalf("Failed to initialize test database: %v", err)
	}
}

func cleanupTestDB(t *testing.T) {
	CloseDB()
}

func TestCreateUser(t *testing.T) {
	setupTestDB(t)
	defer cleanupTestDB(t)

	user, err := CreateUser(12345, "testuser", "Test User")
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}
	if user.ID == 0 {
		t.Error("Expected non-zero user ID")
	}
	if user.TelegramID != 12345 {
		t.Errorf("Expected TelegramID 12345, got %d", user.TelegramID)
	}
	if user.Username != "testuser" {
		t.Errorf("Expected username 'testuser', got %s", user.Username)
	}
	if user.Balance != WelcomeBonusAmount {
		t.Errorf("Expected initial balance %d, got %d", WelcomeBonusAmount, user.Balance)
	}
}

func TestGetUserByTelegramID(t *testing.T) {
	setupTestDB(t)
	defer cleanupTestDB(t)

	// Create a user
	_, err := CreateUser(99999, "uniqueuser", "Unique User")
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	// Retrieve by Telegram ID
	user, err := GetUserByTelegramID(99999)
	if err != nil {
		t.Fatalf("GetUserByTelegramID failed: %v", err)
	}
	if user == nil {
		t.Fatal("Expected user, got nil")
	}
	if user.Username != "uniqueuser" {
		t.Errorf("Expected username 'uniqueuser', got %s", user.Username)
	}
}

func TestGetUserByTelegramIDNotFound(t *testing.T) {
	setupTestDB(t)
	defer cleanupTestDB(t)

	user, err := GetUserByTelegramID(99999999)
	if err != nil {
		t.Fatalf("GetUserByTelegramID should not fail for non-existent user: %v", err)
	}
	if user != nil {
		t.Error("Expected nil user for non-existent Telegram ID")
	}
}

func TestGetUserByID(t *testing.T) {
	setupTestDB(t)
	defer cleanupTestDB(t)

	// Create a user
	created, _ := CreateUser(88888, "idtest", "ID Test")

	// Retrieve by internal ID
	user, err := GetUserByID(created.ID)
	if err != nil {
		t.Fatalf("GetUserByID failed: %v", err)
	}
	if user == nil {
		t.Fatal("Expected user, got nil")
	}
	if user.Username != "idtest" {
		t.Errorf("Expected username 'idtest', got %s", user.Username)
	}
}

func TestCreateMarket(t *testing.T) {
	setupTestDB(t)
	defer cleanupTestDB(t)

	user, _ := CreateUser(66666, "marketcreator", "Market Creator")
	expiresAt := time.Now().Add(24 * time.Hour)

	market, err := CreateMarket(user.ID, "Will it rain tomorrow?", expiresAt)
	if err != nil {
		t.Fatalf("CreateMarket failed: %v", err)
	}
	if market.ID == 0 {
		t.Error("Expected non-zero market ID")
	}
	if market.Question != "Will it rain tomorrow?" {
		t.Errorf("Expected question 'Will it rain tomorrow?', got %s", market.Question)
	}
	if market.CreatorID != user.ID {
		t.Errorf("Expected creator ID %d, got %d", user.ID, market.CreatorID)
	}
	if market.Status != MarketStatusActive {
		t.Errorf("Expected status ACTIVE, got %s", market.Status)
	}
}

func TestGetMarketByID(t *testing.T) {
	setupTestDB(t)
	defer cleanupTestDB(t)

	user, _ := CreateUser(55555, "getmarkettest", "Get Market Test")
	expiresAt := time.Now().Add(24 * time.Hour)
	created, _ := CreateMarket(user.ID, "Test market question?", expiresAt)

	market, err := GetMarketByID(created.ID)
	if err != nil {
		t.Fatalf("GetMarketByID failed: %v", err)
	}
	if market == nil {
		t.Fatal("Expected market, got nil")
	}
	if market.Question != "Test market question?" {
		t.Errorf("Expected question 'Test market question?', got %s", market.Question)
	}
}

func TestUpdateMarketStatus(t *testing.T) {
	setupTestDB(t)
	defer cleanupTestDB(t)

	user, _ := CreateUser(44444, "statustest", "Status Test")
	expiresAt := time.Now().Add(24 * time.Hour)
	market, _ := CreateMarket(user.ID, "Status test market?", expiresAt)

	// Update to LOCKED
	err := UpdateMarketStatus(market.ID, MarketStatusLocked, "")
	if err != nil {
		t.Fatalf("UpdateMarketStatus failed: %v", err)
	}

	// Verify
	updated, _ := GetMarketByID(market.ID)
	if updated.Status != MarketStatusLocked {
		t.Errorf("Expected status LOCKED, got %s", updated.Status)
	}
}

func TestUpdateMarketStatusWithOutcome(t *testing.T) {
	setupTestDB(t)
	defer cleanupTestDB(t)

	user, _ := CreateUser(33333, "outcometest", "Outcome Test")
	expiresAt := time.Now().Add(24 * time.Hour)
	market, _ := CreateMarket(user.ID, "Outcome test market?", expiresAt)

	// Update to RESOLVED with outcome
	err := UpdateMarketStatus(market.ID, MarketStatusResolved, "YES")
	if err != nil {
		t.Fatalf("UpdateMarketStatus failed: %v", err)
	}

	// Verify
	updated, _ := GetMarketByID(market.ID)
	if updated.Status != MarketStatusResolved {
		t.Errorf("Expected status RESOLVED, got %s", updated.Status)
	}
	if updated.Outcome != "YES" {
		t.Errorf("Expected outcome 'YES', got %s", updated.Outcome)
	}
}

func TestPlaceBet(t *testing.T) {
	setupTestDB(t)
	defer cleanupTestDB(t)

	user, _ := CreateUser(22222, "bettest", "Bet Test")
	expiresAt := time.Now().Add(24 * time.Hour)
	market, _ := CreateMarket(user.ID, "Bet test market?", expiresAt)

	ctx := context.Background()
	err := PlaceBet(ctx, user.ID, market.ID, "YES", 10000)
	if err != nil {
		t.Fatalf("PlaceBet failed: %v", err)
	}

	// Verify bet was placed by checking pool totals
	poolYes, _, err := GetPoolTotals(market.ID)
	if err != nil {
		t.Fatalf("GetPoolTotals failed: %v", err)
	}
	if poolYes != 10000 {
		t.Errorf("Expected YES pool 10000, got %d", poolYes)
	}
}

func TestPlaceBetInsufficientFunds(t *testing.T) {
	setupTestDB(t)
	defer cleanupTestDB(t)

	user, _ := CreateUser(22223, "bettestpoor", "Bet Test Poor")
	expiresAt := time.Now().Add(24 * time.Hour)
	market, _ := CreateMarket(user.ID, "Bet test market?", expiresAt)

	ctx := context.Background()
	// Try to bet more than initial balance
	err := PlaceBet(ctx, user.ID, market.ID, "YES", 200000)
	if err == nil {
		t.Error("Expected error for insufficient funds")
	}
}

func TestPlaceBetInvalidOutcome(t *testing.T) {
	setupTestDB(t)
	defer cleanupTestDB(t)

	user, _ := CreateUser(22224, "bettestinvalid", "Bet Test Invalid")
	expiresAt := time.Now().Add(24 * time.Hour)
	market, _ := CreateMarket(user.ID, "Bet test market?", expiresAt)

	ctx := context.Background()
	// Try to bet with invalid outcome
	err := PlaceBet(ctx, user.ID, market.ID, "MAYBE", 1000)
	if err == nil {
		t.Error("Expected error for invalid outcome")
	}
}

func TestGetPoolTotals(t *testing.T) {
	setupTestDB(t)
	defer cleanupTestDB(t)

	user, _ := CreateUser(11111, "pooltest", "Pool Test")
	expiresAt := time.Now().Add(24 * time.Hour)
	market, _ := CreateMarket(user.ID, "Pool test market?", expiresAt)

	ctx := context.Background()
	// Place bets on both outcomes
	_ = PlaceBet(ctx, user.ID, market.ID, "YES", 10000)
	_ = PlaceBet(ctx, user.ID, market.ID, "NO", 15000)

	poolYes, poolNo, err := GetPoolTotals(market.ID)
	if err != nil {
		t.Fatalf("GetPoolTotals failed: %v", err)
	}
	if poolYes != 10000 {
		t.Errorf("Expected YES pool 10000, got %d", poolYes)
	}
	if poolNo != 15000 {
		t.Errorf("Expected NO pool 15000, got %d", poolNo)
	}
}

func TestListActiveMarketsWithCreator(t *testing.T) {
	setupTestDB(t)
	defer cleanupTestDB(t)

	// Use unique Telegram IDs for this test
	user1, _ := CreateUser(111119, "creator1", "Creator 1")
	user2, _ := CreateUser(222229, "creator2", "Creator 2")

	expiresAt := time.Now().Add(24 * time.Hour)
	_, _ = CreateMarket(user1.ID, "Market 1 by creator1", expiresAt)
	_, _ = CreateMarket(user1.ID, "Market 2 by creator1", expiresAt)
	_, _ = CreateMarket(user2.ID, "Market 3 by creator2", expiresAt)

	markets, err := ListActiveMarketsWithCreator()
	if err != nil {
		t.Fatalf("ListActiveMarketsWithCreator failed: %v", err)
	}
	// We might get more than 3 due to other tests, just verify at least 3
	if len(markets) < 3 {
		t.Errorf("Expected at least 3 markets, got %d", len(markets))
	}
}

func TestGetUserBets(t *testing.T) {
	setupTestDB(t)
	defer cleanupTestDB(t)

	user, _ := CreateUser(333333, "bethistory", "Bet History")
	expiresAt := time.Now().Add(24 * time.Hour)
	market1, _ := CreateMarket(user.ID, "Market 1", expiresAt)
	market2, _ := CreateMarket(user.ID, "Market 2", expiresAt)

	ctx := context.Background()
	_ = PlaceBet(ctx, user.ID, market1.ID, "YES", 10000)
	_ = PlaceBet(ctx, user.ID, market2.ID, "NO", 20000)

	bets, err := GetUserBets(user.ID)
	if err != nil {
		t.Fatalf("GetUserBets failed: %v", err)
	}
	if len(bets) != 2 {
		t.Errorf("Expected 2 bets, got %d", len(bets))
	}
}

func TestGetUserStats(t *testing.T) {
	setupTestDB(t)
	defer cleanupTestDB(t)

	user, _ := CreateUser(444444, "stats", "Stats")
	expiresAt := time.Now().Add(24 * time.Hour)
	market, _ := CreateMarket(user.ID, "Stats market", expiresAt)

	ctx := context.Background()
	_ = PlaceBet(ctx, user.ID, market.ID, "YES", 10000)

	stats, err := GetUserStats(user.ID)
	if err != nil {
		t.Fatalf("GetUserStats failed: %v", err)
	}
	if stats.TotalBets != 1 {
		t.Errorf("Expected 1 total bet, got %d", stats.TotalBets)
	}
	if stats.TotalWager != 10000 {
		t.Errorf("Expected 10000 total wager, got %d", stats.TotalWager)
	}
}

func TestGetTopUsers(t *testing.T) {
	setupTestDB(t)
	defer cleanupTestDB(t)

	// Create users with different balances
	_, _ = CreateUser(555551, "richuser", "Rich User")
	_, _ = CreateUser(555552, "pooruser", "Poor User")

	leaderboard, err := GetTopUsers(10)
	if err != nil {
		t.Fatalf("GetTopUsers failed: %v", err)
	}
	if len(leaderboard) < 2 {
		t.Errorf("Expected at least 2 users, got %d", len(leaderboard))
	}
}

func TestGetLastBailoutNoBailout(t *testing.T) {
	setupTestDB(t)
	defer cleanupTestDB(t)

	user, _ := CreateUser(777777, "bailouttest", "Bailout Test")

	// No bailout yet
	lastBailout, hasBailout, err := GetLastBailout(user.ID)
	if err != nil {
		t.Fatalf("GetLastBailout failed: %v", err)
	}
	if hasBailout {
		t.Error("Expected no bailout to exist")
	}
	if !lastBailout.IsZero() {
		t.Error("Expected zero time for no bailout")
	}
}

func TestExecuteBailoutBalanceTooHigh(t *testing.T) {
	setupTestDB(t)
	defer cleanupTestDB(t)

	// User starts with WelcomeBonusAmount (100000) which is > BailoutBalanceThreshold (100)
	user, _ := CreateUser(777779, "bailoutrich", "Bailout Rich")

	// Try to execute bailout with high balance (should fail)
	_, err := ExecuteBailout(user.ID)
	if err == nil {
		t.Error("Expected error for balance too high")
	}
	if err != nil && !strings.Contains(err.Error(), "balance_too_high") {
		t.Errorf("Expected 'balance_too_high' error, got: %v", err)
	}
}

func TestListActiveMarkets(t *testing.T) {
	setupTestDB(t)
	defer cleanupTestDB(t)

	user, _ := CreateUser(888880, "activemarkets", "Active Markets")
	expiresAt := time.Now().Add(24 * time.Hour)
	_, _ = CreateMarket(user.ID, "Active market 1", expiresAt)
	_, _ = CreateMarket(user.ID, "Active market 2", expiresAt)

	markets, err := ListActiveMarkets()
	if err != nil {
		t.Fatalf("ListActiveMarkets failed: %v", err)
	}
	if len(markets) < 2 {
		t.Errorf("Expected at least 2 active markets, got %d", len(markets))
	}
}

func TestGetMarketWithPools(t *testing.T) {
	setupTestDB(t)
	defer cleanupTestDB(t)

	user, _ := CreateUser(888881, "marketpools", "Market Pools")
	expiresAt := time.Now().Add(24 * time.Hour)
	market, _ := CreateMarket(user.ID, "Market with pools", expiresAt)

	ctx := context.Background()
	_ = PlaceBet(ctx, user.ID, market.ID, "YES", 5000)
	_ = PlaceBet(ctx, user.ID, market.ID, "NO", 3000)

	marketWithPools, err := GetMarketWithPools(market.ID)
	if err != nil {
		t.Fatalf("GetMarketWithPools failed: %v", err)
	}
	if marketWithPools == nil {
		t.Fatal("Expected market with pools, got nil")
	}
	if marketWithPools.PoolYes != 5000 {
		t.Errorf("Expected poolYes 5000, got %d", marketWithPools.PoolYes)
	}
	if marketWithPools.PoolNo != 3000 {
		t.Errorf("Expected poolNo 3000, got %d", marketWithPools.PoolNo)
	}
}
