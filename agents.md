## Continuity Ledger (compaction-safe)
Maintain a single Continuity Ledger for this workspace in CONTINUITY.md. The ledger is the canonical session briefing designed to survive context compaction; do not rely on earlier chat text unless it’s reflected in the ledger.

### How it works
- At the start of every assistant turn: read CONTINUITY.md, update it to reflect the latest goal/constraints/decisions/state, then proceed with the work.
- Update CONTINUITY.md again whenever any of these change: goal, constraints/assumptions, key decisions, progress state (Done/Now/Next), or important tool outcomes.
- Keep it short and stable: facts only, no transcripts. Prefer bullets. Mark uncertainty as UNCONFIRMED (never guess).
- If you notice missing recall or a compaction/summary event: refresh/rebuild the ledger from visible context, mark gaps AGENTS.md ask up to 1–3 targeted questions, then continue.

### functions.update_plan vs the Ledger
- functions.update_plan is for short-term execution scaffolding while you work (a small 3–7 step plan with pending/in_progress/completed).
- CONTINUITY.md is for long-running continuity across compaction (the “what/why/current state”), not a step-by-step task list.
- Keep them consistent: when the plan or state changes, update the ledger at the intent/progress level (not every micro-step).

### In replies
- Begin with a brief “Ledger Snapshot” (Goal + Now/Next + Open Questions). Print the full ledger only when it materially changes or when the user asks.

### CONTINUITY.md format (keep headings)
- Goal (incl. success criteria):
- Constraints/Assumptions:
- Key decisions:
- State:
- Done:
- Now:
- Next:
- Open questions (UNCONFIRMED if needed):
- Working set (files/ids/commands):

  
# Architecture & Design Document

This document outlines the service architecture, technology choices, and security protocols.

## 🏗 System Architecture

We utilize a **Monolithic Architecture**. This is a deliberate choice to simplify development, deployment, and debugging for this educational project.

### System Agents (Components)

1.  **Backend (Go Service):**
    * A single compiled binary.
    * Handles business logic (markets, bets, transactions).
    * Acts as the web server (serves static HTML/JS/CSS).
    * Provides a REST API for the frontend.
    * Listens for Telegram Bot API events (Long Polling).

2.  **Database (SQLite):**
    * Serverless relational database.
    * Stores all data in a single file `market.db`.
    * Runs in WAL (Write-Ahead Logging) mode for better concurrency.

3.  **Frontend (Telegram Web App):**
    * SPA (Single Page Application) built with Vanilla JavaScript.
    * Runs inside the Telegram WebView.
    * Does not store secrets; all validation logic resides on the backend.

## 🔐 Authentication & Security

Since the app runs inside Telegram, we bypass traditional login/password methods in favor of native Telegram authentication.

### `initData` Protocol
Telegram passes a query string called `initData` to the Web App, containing user data and a cryptographic signature.

**Interaction Flow:**
1.  **User -> Web App:** User opens the Web App. The Telegram client generates a data packet (`user_id`, `first_name`, `auth_date`, etc.) and signs it with the Bot's secret token (HMAC-SHA256).
2.  **Frontend -> Backend:** JavaScript reads `window.Telegram.WebApp.initData` and sends this "raw" string in the header of every API request:
    `X-Telegram-Init-Data: query_id=...&user=...&hash=...`
3.  **Backend Validation:**
    * The server retrieves the `BOT_TOKEN` (from environment variables).
    * It validates the cryptographic signature (`hash`) from the received string.
    * It checks the data freshness (`auth_date`) to prevent Replay Attacks.
    * If valid, the request is authorized on behalf of the `user_id` contained in the packet.

### Security Justification
The `initData` is signed on Telegram's servers. Forging it without the `BOT_TOKEN` is impossible. This allows us to validate every request "on the fly" (Stateless auth) without implementing complex session management (JWT) initially.

## 💻 Tech Stack & Justification

### 1. Go (Golang)
* **Reasoning:** Strong typing, excellent standard HTTP library, easy compilation into a single binary (Docker-friendly), high performance.
* **Libraries:**
    * `net/http` — Standard web server.
    * `database/sql` + `mattn/go-sqlite3` driver — Database interaction.
    * `gopkg.in/telebot.v3` (or similar) — Wrapper for Telegram Bot API.

### 2. SQLite
* **Reasoning:** "Serverless" database. Ideal for projects that do not require handling millions of requests per second.
* **Storage:** The database file is mounted via a Docker Volume, making backups as simple as copying a file.

### 3. Vanilla JavaScript
* **Reasoning:** Removes the need for a complex build process (Webpack/Vite). Excellent for learning how the DOM and API calls work without framework abstractions. Sufficient for the TWA interface.

### 4. Docker & Portainer
* **Reasoning:** Environment isolation.
* **Hosting:** The project will be deployed on a VPS managed by Portainer. `docker-compose` handles the service and data volumes.

## 📂 Directory Structure (Draft)

```text
/
├── cmd/main.go           # Entry point
├── internal/
│   ├── auth/             # initData validation
│   ├── db/               # SQL migrations and queries
│   └── logic/            # Market and betting rules
├── web/                  # Static files (html, js, css)
├── data/                 # SQLite data folder (in .gitignore)
├── Dockerfile
└── docker-compose.yml
