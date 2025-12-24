Task 04: Betting Engine & Parimutuel Logic
Context: Users have balances (Task 02), and Markets exist (Task 03). Now we implement the core gameplay: Placing Bets. This task requires strict transactional integrity (ACID) to ensure money is never lost during the betting process (e.g., deducted but not bet).

Mechanic: Binary Parimutuel (Pool) Betting. Users bet on "YES" or "NO".

1. Database Infrastructure (internal/storage)
A. Schema Update
Create a new migration/table bets.

Schema:

SQL

CREATE TABLE IF NOT EXISTS bets (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    market_id INTEGER NOT NULL,
    outcome TEXT NOT NULL, -- 'YES' or 'NO'
    amount INTEGER NOT NULL, -- Bet size in cents
    placed_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY(user_id) REFERENCES users(id),
    FOREIGN KEY(market_id) REFERENCES markets(id)
);
B. Transactional Method
Implement PlaceBet(userID int64, marketID int64, outcome string, amount int64) error.

CRITICAL: Atomic Transaction (tx)

Start SQL Transaction.

Check Balance: SELECT balance FROM users WHERE id = ?. If balance < amount -> Rollback & Error.

Check Market: Verify market exists, status = 'ACTIVE', and expires_at > NOW. If not -> Rollback & Error.

Deduct Funds: UPDATE users SET balance = balance - ? WHERE id = ?.

Record Transaction: INSERT INTO transactions ... (Type: 'BET_PLACED').

Record Bet: INSERT INTO bets ....

Commit Transaction.

2. Backend Logic (internal/service)
A. Market Statistics
Update the GetMarkets query/logic to include current pool sizes.

Calculation:

pool_yes = Sum of all bets on this market where outcome = 'YES'.

pool_no = Sum of all bets on this market where outcome = 'NO'.

Return these values in the API so the frontend can calculate "implied odds" (e.g., if Pool Yes is 90% of total, "YES" is the heavy favorite).

3. API Endpoints
A. POST /api/bets
Request JSON:

JSON

{
  "market_id": 12,
  "outcome": "YES", // or "NO"
  "amount": 100     // 1.00 WSC
}
Validation:

Amount > 0.

Outcome is strictly "YES" or "NO".

Response:

Success: 200 OK { "new_balance": 900 }

Failure: 400 Bad Request { "error": "insufficient_funds" }

4. Frontend Integration
A. Betting UI
Update the Market Card in index.html/app.js.

Add:

Amount Input: Number field (e.g., default 10 or 50).

Two Buttons: Green "Vote YES", Red "Vote NO".

Pool Display: Show the total money staked on YES vs NO (e.g., "Pool: 1000 vs 500").

B. Logic
When clicking "Vote YES":

Read amount from input.

Send POST to /api/bets.

On success: Update user balance in the header and refresh the market list to show updated pools.

On error: Show alert() with the error message.

5. Definition of Done
Integrity: Betting 100 coins reduces user balance by exactly 100.

Validation: Cannot bet more than current balance.

Validation: Cannot bet on a market that is expired (past expires_at).

Persistence: The bet appears in the bets table and the deduction appears in transactions.

UI: User sees the pools update immediately after betting.