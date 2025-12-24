Task 02: SQLite Persistence & User Economy
Context: In Task 01, we set up the web server and authentication. Now we need to add the Persistence Layer. The goal of this task is to integrate SQLite, implement automatic user registration upon first login, and handle the "Welcome Bonus" transaction logic.

Tech Stack: Go (database/sql), SQLite. Driver Recommendation: Use modernc.org/sqlite (Pure Go, easier for Alpine Docker images) OR mattn/go-sqlite3 (Standard, requires CGO).

1. Database Infrastructure (internal/storage)
A. Connection & Config
Create a package storage/sqlite.

Implement a function New(storagePath string) that opens (or creates) the database file.

Critical: Enable Write-Ahead Logging for concurrency:

Go

_, err := db.Exec("PRAGMA journal_mode=WAL;")
_, err := db.Exec("PRAGMA foreign_keys = ON;")
B. Migrations (Bootstrap)
On application startup, check for the existence of required tables. Create them if missing.

SQL Schema:

SQL

CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    telegram_id INTEGER UNIQUE NOT NULL,
    username TEXT,
    first_name TEXT,
    balance INTEGER NOT NULL DEFAULT 0, -- Stored in cents (e.g. 1000 = 10.00 WSC)
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS transactions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    amount INTEGER NOT NULL, -- Positive for deposit, negative for withdrawal
    source_type TEXT NOT NULL, -- e.g., 'WELCOME_BONUS', 'BET', 'WIN'
    description TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY(user_id) REFERENCES users(id)
);
2. Business Logic (internal/service)
A. Auth Service Expansion
Modify the authentication middleware/handler logic. After validating initData signature:

Check if the user exists in the DB by telegram_id.

If NO (New User):

Start a DB Transaction (tx).

Insert the new user into users table with 1000 balance (10.00 coins).

Insert a record into transactions (amount: 1000, type: 'WELCOME_BONUS').

Commit the transaction.

If YES (Existing User):

Update first_name and username if they have changed since the last login.

B. User Service
Implement a method GetUser(telegramID int64) (*User, error) that returns the user profile and current balance.

3. API Updates
A. GET /api/me
Create or update this endpoint.

It must return real data from the database.

Response Format:

JSON

{
  "id": 1,
  "telegram_id": 123456789,
  "username": "alice",
  "balance": 1000,
  "balance_display": "10.00"
}
4. Frontend Integration
Update app.js:

After successful authentication, fetch user data from /api/me.

Update the DOM to show:

User's Name.

Balance: Formatted value (divide the integer by 100). E.g., if API returns 1000, display "10.00 WSC".

5. Definition of Done
Persistence: Restarting the Docker container does not lose user data (the market.db file persists via volume).

Onboarding: A new user opening the Web App is automatically created in the DB with a balance of 1000.

Idempotency: Refreshing the page (F5) does not award the Welcome Bonus again (balance stays at 1000).

Audit: The transactions table contains a WELCOME_BONUS record for the new user.

Concurrency: The application runs without locking errors (WAL mode active).