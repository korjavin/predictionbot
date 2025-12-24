# APPEND-ONLY LOG - Never remove or overwrite existing entries

# Continuity Ledger

## Goal (incl. success criteria)
Task 10: Public News Channel (Broadcasting) - IN PROGRESS
- Broadcast new markets to a Telegram channel
- Broadcast market resolutions to the channel
- CHANNEL_ID environment variable configuration

## Constraints/Assumptions
- Extends existing NotificationService to avoid duplication
- Uses goroutines for non-blocking broadcasts
- Gracefully handles missing CHANNEL_ID (no-op)

## Key decisions
- Added channelID field to NotificationService struct
- Created global notification service getter (SetNotificationService/GetNotificationService)
- Added PublishNewMarket and PublishResolution methods
- Message formats use Telegram Markdown for bold/emoji support

## State
- Task 2: COMPLETED
- Task 3: COMPLETED
- Task 4: COMPLETED
- Task 5: COMPLETED
- Task 6: COMPLETED
- Task 7: COMPLETED
- Task 8: COMPLETED
- Task 9: COMPLETED
- Task 10: IN PROGRESS

## Done
- Task 1: User auto-registration with auth
- Task 2: SQLite persistence & user economy
- Task 3: Market Creation & Listing
- Task 4: Betting Engine & Parimutuel Logic
- Task 5: Market Resolution & Payout Engine
- Task 6: Dispute Mechanism & Admin Control
- Task 7: User Profile, History & Push Notifications
- Task 8: Leaderboard & Competition
- Task 9: Bankruptcy Recovery ("The Mortgage")

## Now
- Task 10 implementation in progress
- Broadcasting new markets and resolutions to public channel

## Next
- Test broadcasting functionality with real Telegram channel

## Open questions
- None

## Working set (files/ids/commands)
- internal/service/notification.go (broadcaster methods)
- internal/handlers/markets.go (broadcast on market creation)
- internal/service/payout.go (broadcast on resolution)
- cmd/main.go (global notification service setup)
- docker-compose.yml (CHANNEL_ID env var)

## 2024-12-24 - Public News Channel / Broadcasting (Task 10 - IN PROGRESS)
- Added channelID field to NotificationService struct
- Added SetNotificationService/GetNotificationService global functions
- Added PublishNewMarket() method for broadcasting new markets:
  - Message format: 🆕 *New Market Created* with market ID, question, creator, expiry
- Added PublishResolution() method for broadcasting resolutions:
  - Message format: 🏁 *Market Resolved* with outcome and total pool
- Added parseChannelID() helper (supports @username and -1001234567890 formats)
- Added escapeMarkdown() helper for Telegram Markdown formatting
- Integrated broadcasting into handleCreateMarket() via goroutine
- Integrated broadcasting into payoutService.ResolveMarket() via goroutine
- Added CHANNEL_ID environment variable to docker-compose.yml
- Broadcasting is optional (skips if CHANNEL_ID not set)

## 2024-12-24 - Bug Fix: Blank Screen & Authentication Issues
### Problem
- Web app showing blank blue screen in Telegram browser
- No visible error messages (console inaccessible)
- CRITICAL: HMAC validation using wrong algorithm

### Root Causes Found
1. **Critical Auth Bug**: Hash validation was using bot token directly instead of derived secret key
2. **Missing Key Sorting**: Data check string keys weren't sorted alphabetically (required by Telegram)
3. **Silent Failures**: Frontend errors only logged to console (invisible in Telegram)
4. **Poor Error Handling**: Backend returned generic HTTP errors instead of JSON

### Fixes Applied

**Frontend (web/app.js):**
- Added global `telegramWebApp` and `initData` variables for safer access
- Added `showGlobalError()` function to display errors visibly on screen
- Added validation checks for Telegram WebApp availability
- Added validation checks for empty/missing initData
- Replaced all `window.Telegram.WebApp.initData` references with safe global variable
- Improved error display with user-friendly messages and details
- Enhanced fetchUserProfile() to parse and display backend error messages

**Backend (internal/auth/auth.go):**
- **FIXED CRITICAL BUG**: Implemented correct Telegram Web App HMAC validation:
  - Step 1: Compute secret key = HMAC-SHA256(bot_token, "WebAppData")
  - Step 2: Compute hash = HMAC-SHA256(secret_key, data_check_string)
- Added alphabetical sorting of data check string keys (required by spec)
- Added `writeJSONError()` helper for consistent JSON error responses
- Enhanced logging with [AUTH] prefix for easier debugging
- Improved error messages in all auth failure cases
- Added detailed server logs for auth success/failure paths
