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
