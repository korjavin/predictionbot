# APPEND-ONLY LOG - Never remove or overwrite existing entries

# Continuity Ledger

## Goal (incl. success criteria)
Task 9: Bankruptcy Recovery ("The Mortgage") - COMPLETED
- Users with balance < 100 cents can request a free bailout
- 24-hour cooldown between bailouts
- Bailout resets balance to 500 cents (5.00 WSC)
- Transaction recorded as "BAILOUT" or "Mortgage"
- Frontend shows "Take Mortgage" button when balance < 1.00

## Constraints/Assumptions
- Bailout amount is less than welcome bonus (500 vs 1000 WSC) to encourage valuing capital
- No new database table needed - use existing transactions table
- Cooldown checked against last BAILOUT transaction timestamp

## Key decisions
- BAILOUT source type added to Transaction model
- Fixed bailout amount: 50000 cents (500 WSC)
- Cooldown: 24 hours from last bailout
- Eligibility threshold: balance < 100 cents

## State
- Task 2: COMPLETED
- Task 3: COMPLETED
- Task 4: COMPLETED
- Task 5: COMPLETED
- Task 6: COMPLETED
- Task 7: COMPLETED
- Task 8: COMPLETED
- Task 9: COMPLETED

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
- Task 9 implementation complete, ready for testing

## Next
- Testing & Polishing
- Task 10 (if applicable)

## Open questions
- None

## Working set (files/ids/commands)
- internal/handlers/me.go (HandleBailout endpoint)
- internal/storage/sqlite.go (GetLastBailout, ExecuteBailout)
- internal/storage/models.go (BailoutResult, BailoutError models)
- cmd/main.go (route registration)
- web/app.js (takeMortgage, renderMortgageButton functions)
- web/index.html (mortgage button UI)

## 2024-12-24 - Bankruptcy Recovery / Mortgage (Task 9 - COMPLETED)
- Added BailoutAmount constant (50000 cents = 500 WSC)
- Added BailoutCooldown constant (24 hours)
- Added BailoutBalanceThreshold constant (100 cents)
- Created GetLastBailout(userID) to check last bailout timestamp
- Created ExecuteBailout(userID) with ACID transaction:
  - Validates balance < 100 cents
  - Checks 24-hour cooldown
  - Updates user balance to 50000 cents
  - Creates BAILOUT transaction record
- Created HandleBailout POST /api/me/bailout endpoint:
  - Returns 400 if balance_too_high
  - Returns 429 with cooldown message if cooldown_active
  - Returns 200 with new_balance on success
- Added BailoutResult and BailoutError response models
- Registered route in cmd/main.go
- Added mortgage button UI in Profile tab:
  - Styled with orange background (.btn-mortgage)
  - Hidden by default, shown when balance < 1.00
  - Helper text: "Get 5.00 WSC free (once per 24h)"
- Added JavaScript functions:
  - renderMortgageButton() - Shows/hides button based on balance
  - handleMortgageClick() - Handles button click with loading state
  - takeMortgage() - Calls /api/me/bailout endpoint
- Updated displayUserProfile() to call renderMortgageButton()
- Haptic feedback on success/error for better UX
