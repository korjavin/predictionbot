# APPEND-ONLY LOG - Never remove or overwrite existing entries

# Continuity Ledger

## Goal (incl. success criteria)
Task 8: Leaderboard & Competition
- Add Global Leaderboard showing top users ranked by balance
- Database index on balance column for performance
- API endpoint: GET /api/leaderboard returns top 20 users
- Frontend: Leaders tab with medal icons (🥇🥈🥉) and user highlight

## Constraints/Assumptions
- Leaderboard is public (no auth required for viewing)
- Balances formatted as "500.00" (divide by 100)
- Privacy: Only public info returned (name, username, balance)
- Performance: Index ensures fast queries with many users

## Key decisions
- Used ROW_NUMBER() for proper ranking in SQL
- Three navigation tabs: Markets | Leaders | Profile
- Medal icons for top 3 positions
- Distinct background for current user in list

## State
- Task 2: COMPLETED
- Task 3: COMPLETED
- Task 4: COMPLETED
- Task 5: COMPLETED
- Task 6: COMPLETED
- Task 7: COMPLETED
- Task 8: COMPLETED (Leaderboard)

## Done
- Task 1: User auto-registration with auth
- Task 2: SQLite persistence & user economy
- Task 3: Market Creation & Listing
- Task 4: Betting Engine & Parimutuel Logic
- Task 5: Market Resolution & Payout Engine
- Task 6: Dispute Mechanism & Admin Control
- Task 7: User Profile, History & Push Notifications
- Task 8: Leaderboard & Competition

## Now
- Task 9 (Integration Testing)

## Next
- Testing & Polishing

## Open questions
- None

## Working set (files/ids/commands)
- internal/handlers/leaderboard.go (NEW - Leaderboard API endpoint)
- internal/storage/sqlite.go (GetTopUsers, balance index)
- internal/storage/models.go (LeaderboardEntry model)
- web/app.js (renderLeaderboard, fetchLeaderboard)
- web/index.html (Leaders tab UI)

## 2024-12-24 - Leaderboard & Competition (Task 8 - COMPLETED)
- Added database index: `idx_users_balance ON users(balance DESC)`
- Created `LeaderboardEntry` model with rank, name, username, balance, balance_display
- Implemented `GetTopUsers(limit)` with ROW_NUMBER() for proper ranking
- Created GET /api/leaderboard endpoint (HandleLeaderboard)
- Registered route in cmd/main.go
- Added Leaders tab to navigation (Markets | Leaders | Profile)
- Added leaderboard CSS styles:
  - Medal icons for ranks 1-3 (🥇🥈🥉)
  - Gold/silver/bronze rank styling
  - Distinct background for current user ("is-me" class)
- Added JavaScript functions:
  - `fetchLeaderboard()` - calls /api/leaderboard
  - `renderLeaderboard()` - renders leaderboard with medals and user highlight
- Updated navigation to show/hide leaders tab content

## 2024-12-24 - Debug Logging Refactor (Task 8 - PREVIOUS)
- Created `internal/logger` package for centralized debug logging
- Refactored `auth`, `bot`, and `handlers` packages to use `logger.Debug`
- Ensured consistent log format: `[DEBUG] timestamp=... user_id=... action=... details=...`
- Removed direct `log.Printf` calls and repetitive timestamp formatting

## 2024-12-24 - Dispute Mechanism & Admin Control (Task 6 - COMPLETED)
- Added `MarketStatusResolved` and `MarketStatusDisputed` status constants
- Added `Outcome` and `ResolvedAt` fields to Market struct
- Refactored ResolveMarket() to set RESOLVED status (payouts separate)
- Added RaiseDispute() for users to dispute resolved markets
- Added FinalizeMarket() for payout distribution to winners
- Added GetMarketsPendingFinalization() for auto-finalization queries
- Added UpdateMarketStatus() helper function
- Created POST /api/markets/{id}/dispute endpoint (HandleDispute)
- Created POST /api/admin/resolve endpoint (HandleAdminResolve) with ADMIN_USER_IDS check
- Added DISPUTE_DELAY_MINUTES env var (default: 24 hours)
- Updated market_worker.go to auto-finalize after dispute period
- State Machine: ACTIVE → LOCKED → RESOLVED → (DISPUTED) → FINALIZED
- Added 8 comprehensive tests in payout_test.go

## 2024-12-24 - Debug Logging Implementation (Task 8 - PREVIOUS)
- Added server-side debug logs to track user interactions
- Log format: [DEBUG] timestamp=... user_id=... action=... details=...
- Logs added to:
  - internal/bot/bot.go: Bot command interactions (/start, /help, /balance, /me)
  - internal/auth/auth.go: Authentication events (validation, user creation)
  - internal/handlers/me.go: API endpoint calls
  - internal/handlers/markets.go: Market creation and listing
  - internal/handlers/bets.go: Bet placements and pool interactions

## 2024-12-24 - Betting Engine Implementation (Task 4 - POLISHED)
- Added Bet struct with ID, UserID, MarketID, Outcome, Amount, PlacedAt
- Created bets table with foreign keys and CHECK constraint for outcome validation
- Implemented PlaceBet() with SERIALIZABLE isolation and full ACID compliance
- Added GetPoolTotals() and GetMarketWithPools() for real-time pool statistics
- Created POST /api/bets endpoint with JSON error responses
- Updated frontend with betting UI: amount input, YES/NO buttons, odds display
- Added implied odds calculation (%) and pool amounts in market cards
- Fixed outcome case handling (uppercase YES/NO throughout)
- Fixed error handling with proper JSON responses and UI state restoration
- Transaction type: BET_PLACED per specification
- All validation: positive amounts, sufficient balance, active markets only

## 2024-12-24 - Market Resolution & Payout Engine (Task 5 - COMPLETED)
- Created internal/service/market_worker.go with auto-locking background worker
  - Goroutine runs every 1 minute to check for expired markets
  - Automatically updates ACTIVE markets with expires_at < NOW to LOCKED status
  - Prevents late betting on expired markets
- Implemented internal/service/payout.go with resolution logic
  - ResolveMarket() validates only creator can resolve their market
  - Parimutuel calculation: Payout = (UserBet * TotalPool) / WinningPool
  - Edge case: WinningPool == 0 refunds all bets
  - SERIALIZABLE transaction for ACID compliance
  - Creates WIN_PAYOUT or REFUND transaction records
  - Updates market to FINALIZED with outcome and resolved_at timestamp
- Added POST /api/markets/{id}/resolve endpoint in handlers/markets.go
  - Request: {"outcome": "YES"} or {"outcome": "NO"}
  - Response: {"status": "finalized", "payouts_processed": N}
  - Proper error handling: 404 (not found), 403 (not creator), 409 (wrong status)
- Database schema updates in storage/sqlite.go
  - Added outcome TEXT column to markets table
  - Added resolved_at DATETIME column to markets table
  - Migration logic for existing databases
- Integrated worker in cmd/main.go
  - Worker starts on app startup, stops on graceful shutdown
  - Registered /markets/{id}/resolve route

## 2024-12-24 - User Profile, History & Push Notifications (Task 7 - COMPLETED)
- Created notification service (internal/service/notification.go) for Telegram messages:
  - SendWinNotification(): "🏆 Congratulations! You won X WSC on market '#ID Question'. New Balance: Y WSC"
  - SendRefundNotification(): "💰 Refund received: X WSC has been returned for market '#ID Question'"
  - SendLossNotification(): "📉 Market resolved: Your bet of X WSC on market '#ID Question' did not win"
  - SendDisputeAlert(): Sends alert to admin when dispute is raised
- Integrated notifications into payout.go:
  - FinalizeMarket() sends win/loss/refund notifications after payout
  - RaiseDispute() sends dispute alert to admin (ADMIN_TELEGRAM_ID env var)
- Added betting history queries to sqlite.go:
  - GetUserBets(userID): Returns bets with computed status (WON/LOST/PENDING/REFUNDED)
  - GetUserStats(userID): Returns total_bets, wins, losses, win_rate, total_wager, total_wins
  - computeBetStatus(): Determines bet status based on market state
- Created API endpoints in internal/handlers/history.go:
  - GET /api/me/bets: Returns array of BetHistoryItem
  - GET /api/me/stats: Returns UserStats object
- Updated frontend with Profile tab:
  - Navigation tabs: Markets | Profile
  - Profile header with avatar initial, name, username
  - Balance display synced between tabs
  - Stats grid: Total Bets, Wins, Win Rate (%), Total Profit
  - Betting history with color-coded status badges:
    - Green (WON): Shows payout amount
    - Red (LOST): Shows loss amount
    - Gray (PENDING): Shows bet amount
    - Gray (REFUNDED): Shows refund notification
- Environment variables:
  - ADMIN_TELEGRAM_ID: Telegram user ID to receive dispute alerts
  - TELEGRAM_BOT_TOKEN: Required for notification service

## 2024-12-24 - Bot Commands Implementation
- Implemented Telegram bot commands in internal/bot/bot.go:
  - /start: Welcome message with short description and user's current balance (1000 WSC welcome bonus for new users), includes web app button
  - /help: Lists all available commands with descriptions
  - /balance: Shows user's current balance in WSC format
  - /me: User profile info (name, username, balance, join date)
- Added formatBalance() helper to convert cents to WSC format
- Updated README.md with Bot Commands documentation section
