Task 06: Dispute Mechanism & Admin Control
Context: Currently, payouts happen immediately when the Creator resolves the market. This is risky if the Creator lies. We need to refactor the workflow to introduce a Dispute Period. New Flow:

Creator resolves -> Status RESOLVED (No money sent yet).

Window (e.g., 24h): Users can verify the result.

If Disputed: Status becomes DISPUTED. Admin intervention required.

If No Dispute: After time expires, system auto-finalizes and pays out.

Tech Stack: Go (Refactoring), SQL, Admin Logic.

1. Database & Config (internal/storage)
A. Schema Update
Add a disputes table (optional, or just track via status). For simplicity, we will trust the status field in markets.

Add outcome field to markets if not added in Task 05 (to store the Creator's proposed result).

Config: Add ADMIN_USER_IDS to .env (list of Telegram IDs who are admins).

B. State Machine Refactor
Old Flow: ACTIVE -> LOCKED -> FINALIZED (Payout).

New Flow: ACTIVE -> LOCKED -> RESOLVED (Wait) -> DISPUTED (Optional) -> FINALIZED (Payout).

2. Business Logic Refactor (internal/service)
A. Refactor ResolveMarket (Creator Action)
Change: This method NO LONGER calculates payouts or updates user balances.

Logic:

Check if user is Creator.

Set outcome = 'YES' (or 'NO').

Set status = 'RESOLVED'.

Set resolved_at = NOW.

Note: The money remains locked in the pool.

B. New Method: RaiseDispute (User Action)
Input: marketID, userID.

Logic:

Market must be in RESOLVED status.

Set status = 'DISPUTED'.

(Optional) Log who disputed it.

C. New Method: FinalizeMarket (System/Admin Action)
Input: marketID, forceOutcome (optional string).

Logic:

This contains the Payout Logic removed from Task 05.

If forceOutcome is provided (Admin case), use it. Otherwise, use the stored market.outcome.

Calculate Parimutuel Payouts.

Update User Balances.

Set status = 'FINALIZED'.

D. Background Worker Update
Update the Ticker to handle Auto-Finalization.

Logic:

Query markets where status = 'RESOLVED' AND resolved_at < (NOW - 24 HOURS).

Call FinalizeMarket for them (accepting the Creator's result).

Hint: For testing purposes, make the delay configurable (e.g., 1 minute instead of 24h).

3. API Endpoints
A. POST /api/markets/{id}/dispute
Auth: Any authenticated user.

Logic: Calls RaiseDispute.

Response: 200 OK { "status": "disputed" }

B. POST /api/admin/resolve (Admin Only)
Auth: Check if user.telegram_id is in ADMIN_USER_IDS list.

Request: { "market_id": 12, "outcome": "NO" } (Admin overrides the result).

Logic: Calls FinalizeMarket with the forced outcome.

4. Frontend Integration
A. Market Card Updates
Status RESOLVED:

Show "Result: YES (Pending)".

Show a "⚠ DISPUTE" button for all users.

Status DISPUTED:

Show "⛔ UNDER REVIEW".

Hide Dispute button.

Admin View:

If the user is Admin, show a generic "Force Resolve" interface (or hidden commands) to settle disputed markets.

5. Definition of Done
Safety: Creator resolving a market does not increase anyone's balance immediately.

Dispute: Clicking "Dispute" changes status to DISPUTED and stops the auto-finalization timer (since the worker only picks up RESOLVED markets).

Admin: Admin can force-resolve a DISPUTED market, triggering the correct payouts.

Auto-Run: If nobody disputes, the market automatically finalizes after the configured time delay.