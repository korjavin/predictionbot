# Dispute & Payout Flow

## Overview
This document describes the complete flow for market resolution, dispute period, and payout distribution.

## Timeline

```
Market Created (ACTIVE)
    ↓
Deadline Reached → Market Locked (LOCKED)
    ↓
Creator Resolves → Market Resolved (RESOLVED)
    ↓
24h Dispute Period
    ↓
┌───────────────────┬───────────────────┐
│  No Dispute       │   Dispute Raised  │
│  (automatic)      │   (user action)   │
└───────────────────┴───────────────────┘
    ↓                      ↓
Auto-Finalize         Market Disputed (DISPUTED)
(FINALIZED)               ↓
    ↓                 Admin Reviews
Payouts Distributed       ↓
    ↓                 Admin Resolves
Notifications Sent    (FINALIZED)
                          ↓
                     Payouts Distributed
                          ↓
                     Notifications Sent
```

## Detailed Flow

### 1. Market Creation (ACTIVE)
- Creator creates market via web app
- Market broadcast to public channel
- Users can place bets

### 2. Deadline Reached (LOCKED)
- **Automatic:** MarketWorker locks expired markets every minute
- **Notification:** DM sent to market creator
  - Message: "Your market has reached its deadline. Please resolve it."
  - Commands: `/resolve` (interactive button UI)
- **Status:** ACTIVE → LOCKED

### 3. Market Resolution (RESOLVED)
- **Action:** Creator uses `/resolve` command or web app
- **Outcome:** YES or NO
- **Channel Broadcast:**
  ```
  🏁 Market Resolved

  #X Question

  ✅/❌ Outcome: YES/NO
  💰 Total Pool: XXX WSC

  ⏰ Dispute Period: 24 hours

  If you disagree with this outcome, use /dispute to raise a dispute.
  Winners will receive payouts after the dispute period ends.
  ```
- **Status:** LOCKED → RESOLVED
- **Next:** 24-hour dispute period begins

### 4a. Dispute Period - No Disputes (Auto-Finalization)
- **Automatic:** After 24 hours, MarketWorker auto-finalizes
- **Action:** `FinalizeMarket()` distributes payouts
- **Channel Broadcast:**
  ```
  💰 Payouts Distributed

  #X Question

  ✅/❌ Final Outcome: YES/NO
  💸 XX winners received payouts
  🏆 Total distributed: XXX WSC

  Congratulations to all winners!
  ```
- **User DMs:** Each winner receives:
  ```
  🏆 You won XXX WSC on market #X "Question"

  Your bet: YYY WSC on OUTCOME
  Payout: XXX WSC
  New Balance: ZZZ WSC
  ```
- **User DMs:** Each loser receives:
  ```
  📉 Market finalized: Your bet of XXX WSC on market #X did not win.
  ```
- **Status:** RESOLVED → FINALIZED

### 4b. Dispute Period - Dispute Raised
- **Action:** Any user uses `/dispute` command (interactive UI)
- **Requirements:**
  - Market must be in RESOLVED status
  - User must have placed a bet on the market
- **Channel Broadcast:**
  ```
  ⚠️ Dispute Raised

  #X Question

  A user has disputed the resolution of this market.

  Payouts are frozen pending admin review.
  The admin will review and make a final decision.
  ```
- **Admin DM:**
  ```
  ⚠️ Dispute Alert

  Market ID: #X
  Question: ...
  Current Outcome: YES/NO
  Disputed by: User ID XXX

  Please review and resolve using /resolve_disputes
  ```
- **Creator DM:**
  ```
  ⚠️ Your market #X has been disputed

  Question: ...
  Your resolution: YES/NO

  An admin will review and make the final decision.
  ```
- **Status:** RESOLVED → DISPUTED

### 5. Admin Resolution
- **Action:** Admin uses `/resolve_disputes` command (interactive UI)
- **Options:**
  - Confirm original outcome (YES/NO)
  - Override with opposite outcome
- **Effect:** Same as auto-finalization (4a) but with admin-chosen outcome
- **Channel Broadcast:**
  ```
  🔨 Admin Decision

  #X Question

  ✅/❌ Final Outcome: YES/NO
  (Confirmed/Changed by admin)

  💸 XX winners received payouts
  🏆 Total distributed: XXX WSC
  ```
- **Status:** DISPUTED → FINALIZED

## Commands

### User Commands
- `/resolve` - Resolve your locked market (creator only)
- `/dispute` - Raise dispute on resolved market (interactive selection)
- `/mymarkets` - View markets you created
- `/mybets` - View your active bets

### Admin Commands
- `/resolve_disputes` - Review and resolve disputed markets (interactive)

## Environment Variables
- `DISPUTE_DELAY_MINUTES` - Dispute period in minutes (default: 1440 = 24 hours)
- `ADMIN_TELEGRAM_ID` - Telegram ID of admin user
- `CHANNEL_ID` - Public channel for broadcasts

## Database Status Flow
```
ACTIVE → LOCKED → RESOLVED → FINALIZED
                     ↓
                 DISPUTED → FINALIZED
```

## Notifications Summary

| Event | Channel | Creator DM | Admin DM | Winners DM | Losers DM |
|-------|---------|------------|----------|------------|-----------|
| Market Created | ✅ | ❌ | ❌ | ❌ | ❌ |
| Deadline Reached | ❌ | ✅ | ❌ | ❌ | ❌ |
| Resolution | ✅ | ❌ | ❌ | ❌ | ❌ |
| Dispute Raised | ✅ | ✅ | ✅ | ❌ | ❌ |
| Auto-Finalization | ✅ | ❌ | ❌ | ✅ | ✅ |
| Admin Finalization | ✅ | ❌ | ❌ | ✅ | ✅ |
