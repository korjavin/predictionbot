Task 05: Market Resolution & Payout Engine
Context: Currently, markets stay open indefinitely. We need a lifecycle mechanism to:

Automatically LOCK markets when time expires (stop accepting bets).

Allow the Creator to RESOLVE the market (declare the winner).

Calculate and DISTRIBUTE winnings based on the Parimutuel formula.

Tech Stack: Go (Goroutines/Ticker), SQL Transactions.

1. Background Worker (internal/service/market_worker.go)
A. Auto-Locking Mechanism
Create a Goroutine (Ticker) that runs every 1 minute.

Logic:

Query markets where status = 'ACTIVE' AND expires_at < NOW.

Update their status to LOCKED.

This prevents users from betting on expired events even if they try to bypass frontend checks.

2. Payout Logic (internal/service/payout.go)
A. The Math (Integer Arithmetic)
We use integer math to avoid floating-point errors. Formula for a user who bet on the WINNING outcome:

Go

// Payout = (UserBet * TotalPool) / WinningPool
// Example:
// Total Pool = 1500 (1000 on YES, 500 on NO). YES wins.
// User bet 100 on YES.
// Payout = (100 * 1500) / 1000 = 150 coins.
B. Resolution Service Method
Implement ResolveMarket(marketID int64, creatorID int64, outcome string) error.

Validation:

Only the Creator (or Admin) can call this.

Market must be LOCKED or ACTIVE.

Execution (Transaction):

Update Market: Set status = 'FINALIZED', outcome = ?, resolved_at = NOW.

Get all bets for this market.

Calculate totals: TotalPool, WinningPool.

Edge Case: If WinningPool == 0 (Nobody guessed right), Refund everyone (or House keeps it - let's do Refund for this project).

Loop through Winners:

Calculate Payout.

UPDATE users SET balance = balance + Payout.

INSERT INTO transactions (Type: 'WIN_PAYOUT').

Commit Transaction.

3. API Endpoints
A. POST /api/markets/{id}/resolve
Purpose: Allows the creator to finalize the market.

Request JSON:

JSON

{ "outcome": "YES" } // or "NO"
Response: 200 OK { "status": "finalized", "payouts_processed": 5 }

4. Frontend Integration
A. My Markets / Creator View
If the logged-in user is the Creator of a market, and the market is expired (or active), show a "Resolve" button on the card.

Clicking "Resolve" opens a dialog: "Did this happen? [YES] [NO]".

B. User Notifications (Simple)
When the market is resolved, the user balance in the header should update automatically on the next refresh/action.

5. Definition of Done
Auto-Lock: Waiting past the expiration time changes the market status in the DB to LOCKED.

Security: Only the Creator can resolve their market.

Math Accuracy:

Create a market.

User A bets 100 on YES.

User B bets 100 on NO.

Resolve as YES.

User A receives 200 (Net profit +100). User B gets 0.

Refund Logic: If User A bets 100 on YES, and nobody bets on NO, and YES wins -> User A gets 100 back (Payout logic handles WinningPool == TotalPool).

Transactions: The database history shows WIN_PAYOUT records linked to the correct market.