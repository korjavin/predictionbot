# APPEND-ONLY LOG - Never remove or overwrite existing entries

# Continuity Ledger

## Goal (incl. success criteria)
Complete Task 4: Betting Engine & Parimutuel Logic
- Bets table schema with user_id, market_id, outcome, amount
- PlaceBet() atomic transaction with ACID compliance
- Pool statistics (pool_yes, pool_no) calculation
- POST /api/bets endpoint with validation
- Frontend betting UI with YES/NO buttons and odds display

## Constraints/Assumptions
- Use pure Go SQLite driver (modernc.org/sqlite)
- Bet amount validation: positive integer, cannot exceed balance
- Outcome restricted to "YES" or "NO" (CHECK constraint)
- Transaction isolation: SERIALIZABLE for financial integrity
- Parimutuel pools: implied odds calculated from pool totals
- Market must be ACTIVE and not expired to accept bets

## Key decisions
- Pool totals stored as separate columns (pool_yes, pool_no) for efficiency
- Bet placement returns updated pool totals for immediate UI refresh
- Atomic transaction prevents double-spending on concurrent bets

## State
- Task 2: COMPLETED
- Task 3: COMPLETED
- Task 4: COMPLETED

## Done
- Task 1: User auto-registration with auth
- Task 2: SQLite persistence & user economy
- Task 3: Market Creation & Listing
- Task 4: Betting Engine & Parimutuel Logic

## Now
- Task 4 implementation complete, awaiting user feedback

## Next
- Task 5 (Market Resolution)

## Open questions
- None

## Working set (files/ids/commands)
- internal/storage/models.go (Bet struct, Outcome type)
- internal/storage/sqlite.go (bets table, PlaceBet(), GetPoolTotals(), GetMarketWithPools())
- internal/handlers/bets.go (POST /api/bets handler)
- cmd/main.go (route registration)
- web/index.html (betting UI controls)
- web/app.js (placeBet function, renderMarkets with odds, handleBetClick)

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

## 2024-12-24 - Bot Commands Implementation
- Implemented Telegram bot commands in internal/bot/bot.go:
  - /start: Welcome message with short description and user's current balance (1000 WSC welcome bonus for new users), includes web app button
  - /help: Lists all available commands with descriptions
  - /balance: Shows user's current balance in WSC format
  - /me: User profile info (name, username, balance, join date)
- Added formatBalance() helper to convert cents to WSC format
- Updated README.md with Bot Commands documentation section
