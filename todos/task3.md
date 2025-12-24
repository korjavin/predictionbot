Task 03: Market Creation & Listing
Context: Users are now registered and have a balance. The next core feature is the ability to create prediction markets and view a list of active markets. This task focuses on the markets table structure, the API for creating markets, and the frontend UI to display them.

Tech Stack: Go, SQLite, Vanilla JS.

1. Database Infrastructure (internal/storage)
A. Schema Update
Create a new migration/table markets.

Schema:

SQL

CREATE TABLE IF NOT EXISTS markets (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    creator_id INTEGER NOT NULL,
    question TEXT NOT NULL,
    image_url TEXT,         -- Optional: URL to an image
    status TEXT NOT NULL DEFAULT 'ACTIVE', -- Enum: ACTIVE, LOCKED, RESOLVING, FINALIZED
    expires_at DATETIME NOT NULL, -- Betting deadline
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY(creator_id) REFERENCES users(id)
);
Note: For now, we only need basic fields. Outcome resolution fields will be added later.

B. Storage Methods
Implement CreateMarket(market *Market) error.

Implement ListActiveMarkets() ([]Market, error).

Query should filter by status = 'ACTIVE' and order by created_at DESC.

2. Backend Logic (internal/service)
A. Market Service
Validation:

Question: Min 10 chars, Max 140 chars.

ExpiresAt: Must be in the future (at least 1 hour from now).

Logic:

Associate the market with the creator_id (from the Context/Auth).

Set initial status to ACTIVE.

3. API Endpoints
A. POST /api/markets
Purpose: Create a new market.

Headers: X-Telegram-Init-Data (Required).

Request JSON:

JSON

{
  "question": "Will Bitcoin hit $100k by Jan 1st?",
  "expires_at": "2024-01-01T00:00:00Z"
}
Response (201 Created):

JSON

{ "id": 12, "status": "created" }
B. GET /api/markets
Purpose: Get list of active markets to display in the feed.

Response (200 OK):

JSON

[
  {
    "id": 12,
    "question": "Will Bitcoin hit $100k by Jan 1st?",
    "creator_name": "Alice",
    "expires_at": "2024-01-01T00:00:00Z",
    "pool_yes": 0,  // Placeholder for next task
    "pool_no": 0    // Placeholder for next task
  }
]
4. Frontend Integration
A. UI Components
"Create Market" Button: Opens a modal or a simple form.

Form Inputs:

Text Input (Question).

Date/Time Picker (Deadline).

Market Feed:

Render a list of cards under the "Create" button.

Each card displays the question and a countdown or date for expires_at.

B. JS Logic (app.js)
Create a function renderMarkets() that fetches /api/markets and updates the DOM.

Handle form submission: POST to /api/markets, then reload the list upon success.

5. Definition of Done
DB: New table markets exists.

API: Sending a POST request creates a row in the database linked to the correct user.

Validation: Trying to create a market with a past date returns an error (400 Bad Request).

UI: The main page displays a list of created markets.

UI: User can fill out the form, submit, and see their new market appear in the list immediately.