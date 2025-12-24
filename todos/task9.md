Task 09: Bankruptcy Recovery ("The Mortgage")
Context: Since this is an educational project, we want users to keep playing even if they lose all their money. We are introducing a "Bailout" mechanism (fun name: "Mortgage"). If a user runs out of funds, they can request a free refill, but with strict limits to prevent abuse.

Tech Stack: Go (Time logic), SQL.

1. Business Logic Rules
Eligibility Condition: User's current balance must be < 100 (less than 1.00 WSC).

Cooldown: The user cannot receive a bailout if they have already received one in the last 24 hours.

Amount: The bailout resets the balance to a fixed amount (e.g., 500 cents = 5.00 WSC). Note: It's less than the starting bonus to encourage valuing the initial capital.

2. Database & Logic (internal/service)
A. Check Eligibility
We don't need a new table. We can query the transactions table.

Logic:

Check user.balance. If >= 100, return Error ("You are not bankrupt").

Query transactions:

SQL

SELECT created_at FROM transactions
WHERE user_id = ? AND source_type = 'BAILOUT'
ORDER BY created_at DESC LIMIT 1;
If a record exists and created_at is less than 24 hours ago, return Error ("Come back tomorrow").

B. Execute Bailout
Transaction (tx):

UPDATE users SET balance = balance + 500 ...

INSERT INTO transactions (user_id, amount, source_type, description) VALUES (?, 500, 'BAILOUT', 'Emergency mortgage').

Commit.

3. API Endpoints
A. POST /api/me/bailout
Auth: Required.

Response:

200 OK: { "message": "Funds added", "new_balance": 505 }

400 Bad Request: { "error": "balance_too_high" }

429 Too Many Requests: { "error": "cooldown_active", "next_available": "2023-10-11T12:00:00Z" }

4. Frontend Integration
A. Profile UI Update
In the Profile tab, check the user's balance.

Condition: If balance < 1.00:

Show a prominent button: "💸 Take Mortgage" (Взять ипотеку).

Add a helper text: "Get 5.00 WSC free (once per 24h)".

Action:

Clicking sends POST to /api/me/bailout.

On success: Play a sound (optional), update balance, hide button.

On error (429): Show alert "Bank says NO: Come back in [Time]".

5. Definition of Done
Restriction: Users with money cannot use the endpoint.

Cooldown: A user cannot get the bonus twice in a row immediately.

Accounting: The transaction appears in history as BAILOUT (or "Mortgage").

UX: The button is hidden for rich users and visible for poor users.