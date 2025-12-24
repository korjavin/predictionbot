Task 08: Leaderboard & Competition
Context: To increase user engagement and retention, we need to introduce social competition. Users should be able to see who the best predictors are. This task implements a Global Leaderboard showing the top users ranked by their current wealth (Balance).

Tech Stack: Go, SQL (Indexing), Frontend UI.

1. Database Optimization (internal/storage)
A. Indexing
Since we will frequently query users sorted by balance, we must optimize the database.

Migration: Execute CREATE INDEX IF NOT EXISTS idx_users_balance ON users(balance DESC);.

This ensures the leaderboard query remains fast even with thousands of users.

B. Storage Method
Implement GetTopUsers(limit int) ([]User, error).

Query:

SQL

SELECT telegram_id, username, first_name, balance
FROM users
ORDER BY balance DESC
LIMIT ?;
Privacy Note: Do not return internal id or sensitive fields, only public info.

2. API Endpoints
A. GET /api/leaderboard
Purpose: Returns the top N (e.g., 20) players.

Response:

JSON

[
  {
    "rank": 1,
    "name": "Pavel D.",
    "username": "durov",
    "balance": 50000,
    "balance_display": "500.00"
  },
  {
    "rank": 2,
    "name": "Elon",
    "username": "elonmusk",
    "balance": 45000,
    "balance_display": "450.00"
  }
  // ...
]
3. Frontend Integration
A. UI Update (Navigation)
Add a third tab to the bottom navigation: "Leaders" (Icon: 🏆).

B. Leaderboard View
Render a list of users.

Visuals:

Gold/Silver/Bronze icons for positions 1, 2, 3.

Distinct background for the current user if they appear in the list (highlight "Me").

Display Name and formatted Balance.

Empty State: If for some reason the list is empty (should not happen), show "No leaders yet."

4. Definition of Done
Performance: The database has an index on the balance column.

API: The endpoint returns a JSON array sorted from highest balance to lowest.

UI: Clicking the "Leaders" tab displays the list.

Formatting: Balances are correctly divided by 100 (e.g., 50000 -> "500.00").

Accuracy: If User A wins a bet and their balance exceeds User B, User A immediately moves up in the leaderboard on refresh.
