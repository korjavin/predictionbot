package service

import (
	"context"
	"os"
	"testing"
	"time"

	"predictionbot/internal/storage"
)

func setupTestDB(t *testing.T) {
	// Use in-memory database for tests
	if err := storage.InitDB(":memory:"); err != nil {
		t.Fatalf("Failed to initialize test database: %v", err)
	}
}

func cleanupTestDB(t *testing.T) {
	storage.CloseDB()
}

func TestResolveMarket(t *testing.T) {
	setupTestDB(t)
	defer cleanupTestDB(t)

	ctx := context.Background()
	payoutService := NewPayoutService()

	// Create a test user
	user, err := storage.CreateUser(12345, "testuser", "Test User")
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Create a test market (it will be ACTIVE)
	expiresAt := time.Now().Add(1 * time.Hour)
	market, err := storage.CreateMarket(user.ID, "Test market question?", expiresAt)
	if err != nil {
		t.Fatalf("Failed to create market: %v", err)
	}

	// Manually lock the market (simulate expiration)
	err = storage.UpdateMarketStatus(market.ID, storage.MarketStatusLocked, "")
	if err != nil {
		t.Fatalf("Failed to lock market: %v", err)
	}

	// Test: Resolve market as creator
	err = payoutService.ResolveMarket(ctx, market.ID, user.ID, "YES")
	if err != nil {
		t.Fatalf("ResolveMarket failed: %v", err)
	}

	// Verify market is now RESOLVED
	updatedMarket, err := storage.GetMarketByID(market.ID)
	if err != nil {
		t.Fatalf("Failed to get market: %v", err)
	}
	if updatedMarket.Status != storage.MarketStatusResolved {
		t.Errorf("Expected market status RESOLVED, got %s", updatedMarket.Status)
	}
	if updatedMarket.Outcome != "YES" {
		t.Errorf("Expected market outcome YES, got %s", updatedMarket.Outcome)
	}
}

func TestResolveMarketNotCreator(t *testing.T) {
	setupTestDB(t)
	defer cleanupTestDB(t)

	ctx := context.Background()
	payoutService := NewPayoutService()

	// Create two test users
	creator, _ := storage.CreateUser(11111, "creator", "Creator")
	otherUser, _ := storage.CreateUser(22222, "other", "Other User")

	// Create a test market
	expiresAt := time.Now().Add(1 * time.Hour)
	market, _ := storage.CreateMarket(creator.ID, "Test market question?", expiresAt)

	// Lock the market
	storage.UpdateMarketStatus(market.ID, storage.MarketStatusLocked, "")

	// Test: Try to resolve market as non-creator
	err := payoutService.ResolveMarket(ctx, market.ID, otherUser.ID, "YES")
	if err == nil {
		t.Error("Expected error when non-creator tries to resolve market")
	}
}

func TestResolveMarketNotLocked(t *testing.T) {
	setupTestDB(t)
	defer cleanupTestDB(t)

	ctx := context.Background()
	payoutService := NewPayoutService()

	// Create a test user
	user, _ := storage.CreateUser(33333, "test", "Test")

	// Create a test market (still ACTIVE)
	expiresAt := time.Now().Add(1 * time.Hour)
	market, _ := storage.CreateMarket(user.ID, "Test market question?", expiresAt)

	// Test: Try to resolve market that's still ACTIVE
	err := payoutService.ResolveMarket(ctx, market.ID, user.ID, "YES")
	if err == nil {
		t.Error("Expected error when trying to resolve non-LOCKED market")
	}
}

func TestRaiseDispute(t *testing.T) {
	setupTestDB(t)
	defer cleanupTestDB(t)

	ctx := context.Background()
	payoutService := NewPayoutService()

	// Create a test user
	user, _ := storage.CreateUser(44444, "testuser", "Test User")

	// Create a test market
	expiresAt := time.Now().Add(1 * time.Hour)
	market, _ := storage.CreateMarket(user.ID, "Test market question?", expiresAt)

	// Lock and resolve the market
	storage.UpdateMarketStatus(market.ID, storage.MarketStatusLocked, "")
	storage.UpdateMarketStatus(market.ID, storage.MarketStatusResolved, "YES")

	// Test: Raise dispute on resolved market
	err := payoutService.RaiseDispute(ctx, market.ID, user.ID)
	if err != nil {
		t.Fatalf("RaiseDispute failed: %v", err)
	}

	// Verify market is now DISPUTED
	updatedMarket, _ := storage.GetMarketByID(market.ID)
	if updatedMarket.Status != storage.MarketStatusDisputed {
		t.Errorf("Expected market status DISPUTED, got %s", updatedMarket.Status)
	}
}

func TestRaiseDisputeNotResolved(t *testing.T) {
	setupTestDB(t)
	defer cleanupTestDB(t)

	ctx := context.Background()
	payoutService := NewPayoutService()

	// Create a test user
	user, _ := storage.CreateUser(55555, "testuser", "Test User")

	// Create a test market (still ACTIVE)
	expiresAt := time.Now().Add(1 * time.Hour)
	market, _ := storage.CreateMarket(user.ID, "Test market question?", expiresAt)

	// Test: Try to dispute market that's still ACTIVE
	err := payoutService.RaiseDispute(ctx, market.ID, user.ID)
	if err == nil {
		t.Error("Expected error when trying to dispute non-RESOLVED market")
	}
}

func TestFinalizeMarket(t *testing.T) {
	setupTestDB(t)
	defer cleanupTestDB(t)

	ctx := context.Background()
	payoutService := NewPayoutService()

	// Create test users
	creator, _ := storage.CreateUser(66666, "creator", "Creator")
	winner, _ := storage.CreateUser(77777, "winner", "Winner")
	loser, _ := storage.CreateUser(88888, "loser", "Loser")

	// Create a test market
	expiresAt := time.Now().Add(1 * time.Hour)
	market, _ := storage.CreateMarket(creator.ID, "Will it rain tomorrow?", expiresAt)

	// Place bets WHILE market is ACTIVE (before locking)
	_ = storage.PlaceBet(ctx, winner.ID, market.ID, "YES", 10000) // 100.00 on YES
	_ = storage.PlaceBet(ctx, loser.ID, market.ID, "NO", 10000)   // 100.00 on NO

	// Now lock and resolve the market
	storage.UpdateMarketStatus(market.ID, storage.MarketStatusLocked, "")
	storage.UpdateMarketStatus(market.ID, storage.MarketStatusResolved, "YES")

	// Get winner balance before payout
	winnerBefore, _ := storage.GetUserByID(winner.ID)

	// Test: Finalize market
	payouts, err := payoutService.FinalizeMarket(ctx, market.ID, "")
	if err != nil {
		t.Fatalf("FinalizeMarket failed: %v", err)
	}
	if payouts != 1 {
		t.Errorf("Expected 1 payout, got %d", payouts)
	}

	// Verify market is now FINALIZED
	updatedMarket, _ := storage.GetMarketByID(market.ID)
	if updatedMarket.Status != storage.MarketStatusFinalized {
		t.Errorf("Expected market status FINALIZED, got %s", updatedMarket.Status)
	}

	// Verify winner received payout (parimutuel: bet 100, total pool 200, winning pool 100, payout = 100 * 200 / 100 = 200)
	winnerAfter, _ := storage.GetUserByID(winner.ID)
	expectedPayout := int64(20000) // 200.00 in cents
	if winnerAfter.Balance-winnerBefore.Balance != expectedPayout {
		t.Errorf("Expected winner payout of %d, got %d", expectedPayout, winnerAfter.Balance-winnerBefore.Balance)
	}
}

func TestFinalizeMarketWithForceOutcome(t *testing.T) {
	setupTestDB(t)
	defer cleanupTestDB(t)

	ctx := context.Background()
	payoutService := NewPayoutService()

	// Create test users
	creator, _ := storage.CreateUser(99999, "creator", "Creator")
	winner, _ := storage.CreateUser(100000, "winner", "Winner")

	// Create a test market
	expiresAt := time.Now().Add(1 * time.Hour)
	market, _ := storage.CreateMarket(creator.ID, "Test market question?", expiresAt)

	// Place bets WHILE market is ACTIVE
	_ = storage.PlaceBet(ctx, winner.ID, market.ID, "YES", 10000) // Bet on YES

	// Now lock and resolve the market (creator said NO, but admin will override)
	storage.UpdateMarketStatus(market.ID, storage.MarketStatusLocked, "")
	storage.UpdateMarketStatus(market.ID, storage.MarketStatusResolved, "NO")

	// Test: Finalize market with force outcome YES (admin override)
	payouts, err := payoutService.FinalizeMarket(ctx, market.ID, "YES")
	if err != nil {
		t.Fatalf("FinalizeMarket failed: %v", err)
	}
	if payouts != 1 {
		t.Errorf("Expected 1 payout, got %d", payouts)
	}

	// Verify market is FINALIZED with forced outcome
	updatedMarket, _ := storage.GetMarketByID(market.ID)
	if updatedMarket.Status != storage.MarketStatusFinalized {
		t.Errorf("Expected market status FINALIZED, got %s", updatedMarket.Status)
	}
	if updatedMarket.Outcome != "YES" {
		t.Errorf("Expected market outcome YES (forced), got %s", updatedMarket.Outcome)
	}
}

func TestFinalizeMarketNoWinnersRefund(t *testing.T) {
	setupTestDB(t)
	defer cleanupTestDB(t)

	ctx := context.Background()
	payoutService := NewPayoutService()

	// Create test users
	creator, _ := storage.CreateUser(111111, "creator", "Creator")
	bettor, _ := storage.CreateUser(222222, "bettor", "Bettor")

	// Create a test market
	expiresAt := time.Now().Add(1 * time.Hour)
	market, _ := storage.CreateMarket(creator.ID, "Test market question?", expiresAt)

	// Place bet on NO WHILE market is ACTIVE
	_ = storage.PlaceBet(ctx, bettor.ID, market.ID, "NO", 10000)

	// Now lock and resolve the market (outcome YES)
	storage.UpdateMarketStatus(market.ID, storage.MarketStatusLocked, "")
	storage.UpdateMarketStatus(market.ID, storage.MarketStatusResolved, "YES")

	// Test: Finalize market - should refund since no one bet on YES
	payouts, err := payoutService.FinalizeMarket(ctx, market.ID, "")
	if err != nil {
		t.Fatalf("FinalizeMarket failed: %v", err)
	}
	if payouts != 1 {
		t.Errorf("Expected 1 refund, got %d", payouts)
	}

	// Get bettor balance after finalization
	bettorAfter, _ := storage.GetUserByID(bettor.ID)
	// Bettor started with 100000, bet 10000, has 90000 left
	// After refund should be back to 100000
	if bettorAfter.Balance != 100000 {
		t.Errorf("Expected bettor balance to be 100000 after refund, got %d", bettorAfter.Balance)
	}
}

func TestAutoFinalizationConfig(t *testing.T) {
	// Test that DISPUTE_DELAY_MINUTES environment variable is respected
	os.Setenv("DISPUTE_DELAY_MINUTES", "5")
	defer os.Unsetenv("DISPUTE_DELAY_MINUTES")

	worker := NewMarketWorker()
	if worker.disputeDelay != 5*time.Minute {
		t.Errorf("Expected dispute delay of 5 minutes, got %v", worker.disputeDelay)
	}
}
