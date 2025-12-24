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
