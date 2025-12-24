# APPEND-ONLY LOG - Never remove or overwrite existing entries

# Continuity Ledger

## Goal (incl. success criteria)
Task 8: Add Server-Side Debug Logs
- Add debug logging to track user interactions across the application
- Log format: [DEBUG] timestamp=... user_id=... action=... details=...
- Logs added to: bot.go (bot commands), handlers/* (API endpoints), auth.go (auth events)

## Constraints/Assumptions
- Debug logs should include: user_id, action, timestamp, relevant details
- Logs must not expose sensitive information (credentials, full user data)
- Log level: DEBUG (can be filtered in production)

## Key decisions
- Fixed newline escaping issue in `internal/bot/bot.go` for Markdown formatting
- Consistent log format across all modules for easy parsing
- Using standard `log` package with structured fields

## State
- Task 2: COMPLETED
- Task 3: COMPLETED
- Task 4: COMPLETED
- Task 8: COMPLETED

## Done
- Task 1: User auto-registration with auth
- Task 2: SQLite persistence & user economy
- Task 3: Market Creation & Listing
- Task 4: Betting Engine & Parimutuel Logic
- Task 8: Add Server-Side Debug Logs

## Now
- Task 5 (Market Resolution)
- Task 6 (Admin Controls)
- Task 7 (Bot Market Commands)

## Next
- Testing & Polishing

## Open questions
- None

## Working set (files/ids/commands)
- internal/logger/logger.go (New shared logging package)
- internal/bot/bot.go
- internal/auth/auth.go
- internal/handlers/me.go
- internal/handlers/markets.go
- internal/handlers/bets.go

## 2024-12-24 - Debug Logging Refactor (Task 8 - COMPLETED)
- Created `internal/logger` package for centralized debug logging
- Refactored `auth`, `bot`, and `handlers` packages to use `logger.Debug`
- Ensured consistent log format: `[DEBUG] timestamp=... user_id=... action=... details=...`
- Removed direct `log.Printf` calls and repetitive timestamp formatting


## 2024-12-24 - Debug Logging Implementation (Task 8)
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

## 2024-12-24 - Bot Commands Implementation
- Implemented Telegram bot commands in internal/bot/bot.go:
  - /start: Welcome message with short description and user's current balance (1000 WSC welcome bonus for new users), includes web app button
  - /help: Lists all available commands with descriptions
  - /balance: Shows user's current balance in WSC format
  - /me: User profile info (name, username, balance, join date)
- Added formatBalance() helper to convert cents to WSC format
- Updated README.md with Bot Commands documentation section
