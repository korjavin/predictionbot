# APPEND-ONLY LOG - Never remove or overwrite existing entries

# Continuity Ledger

## Goal (incl. success criteria)
Complete Task 3: Market Creation & Listing
- DB: markets table exists with correct schema
- POST /api/markets creates a market with validation
- Validation rejects past dates and invalid questions
- UI displays list of markets
- User can create and see new markets

## Constraints/Assumptions
- Use pure Go SQLite driver (modernc.org/sqlite)
- Market status: ACTIVE, LOCKED, RESOLVING, FINALIZED
- Question validation: 10-140 chars
- ExpiresAt: Must be at least 1 hour in the future
- Initial status set to ACTIVE

## Key decisions
- Market list includes creator_name, pool_yes, pool_no placeholders
- Frontend uses Vanilla JS with Telegram WebView styling

## State
- Task 2: COMPLETED
- Task 3: COMPLETED

## Done
- Task 1: User auto-registration with auth
- Task 2: SQLite persistence & user economy
- Task 3: Market Creation & Listing

## Now
- Awaiting user feedback on Task 3 completion

## Next
- Task 4 (Betting System)

## Open questions
- None

## Working set (files/ids/commands)
- internal/storage/models.go (Market struct)
- internal/storage/sqlite.go (markets table + methods)
- internal/handlers/markets.go (handlers)
- cmd/main.go (route registration)
- web/index.html (Create Market form + market feed)
- web/app.js (renderMarkets + createMarket functions)

## 2024-12-24 - Bot Commands Implementation
- Implemented Telegram bot commands in internal/bot/bot.go:
  - /start: Welcome message with short description and user's current balance (1000 WSC welcome bonus for new users), includes web app button
  - /help: Lists all available commands with descriptions
  - /balance: Shows user's current balance in WSC format
  - /me: User profile info (name, username, balance, join date)
- Added formatBalance() helper to convert cents to WSC format
- Updated README.md with Bot Commands documentation section
