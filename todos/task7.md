Task 07: User Profile, History & Push Notifications
Context: The core mechanics work, but the User Experience (UX) is incomplete. Users cannot see their betting history and are not notified when they win. This task focuses on Retention: giving users a "Profile" view and sending them Telegram messages when important events occur (especially payouts).

Tech Stack: Go (Telegram Bot API), SQL Queries, Frontend UI.

1. Backend Logic (internal/service)
A. Notification Service
Create a simple internal service or helper to send messages via the Telegram Bot.

Events to Notify:

Payout (Win): When FinalizeMarket runs and a user receives money.

Message: "🏆 Congratulations! You won 500 WSC on market 'Bitcoin $100k'. New Balance: 1500 WSC."

Dispute Alert (Admin only): When RaiseDispute is called.

Message: "⚠ Dispute Raised! Market ID #12 needs review."

Implementation Note: Be mindful of Telegram Rate Limits (approx 30 msgs/sec). For this educational project, sending messages sequentially in a Goroutine is acceptable.

B. Betting History Service
Implement GetUserBets(userID int64) ([]BetHistoryItem, error).

Query:

Select from bets.

Join with markets to get the question title and current status.

Order by placed_at DESC.

Computed Fields:

Determine if the bet was Won/Lost/Pending based on Market Status and Outcome.

2. API Endpoints
A. GET /api/me/bets
Response:

JSON

[
  {
    "market_id": 12,
    "question": "Will it rain?",
    "outcome_chosen": "YES",
    "amount": 100,
    "status": "WON", // or PENDING, LOST, REFUNDED
    "payout": 150,   // Optional: calculated if won
    "placed_at": "2023-10-10T10:00:00Z"
  }
]
B. GET /api/me/stats (Optional)
Return simple stats: "Total Bets", "Wins", "Win Rate %".

3. Frontend Integration
A. Profile Tab/View
Update index.html to have a navigation bar (e.g., "Markets" | "Profile").

Profile View:

Show User Avatar (from Telegram initData if available, or placeholder).

Show Big Balance.

"My History" List: Render the data from /api/me/bets.

Style rows differently based on result (Green for Win, Red for Loss, Gray for Pending).

4. Definition of Done
History: User can open the "Profile" tab and see a list of their past bets.

Accuracy: The history correctly reflects the status (e.g., if a market was finalized as NO, a YES bet shows as "LOST").

Notification:

Create a market -> Bet on it -> Resolve it as a winner.

Result: The User receives a real message in their Telegram Chat from the bot: "You won X coins!".

Admin Alert: Raising a dispute triggers a message to the Admin's Telegram ID.