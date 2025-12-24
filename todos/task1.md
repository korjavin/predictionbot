Here is the English version of the first task description, formatted in Markdown. You can copy this file and give it to your coding agent/developer.

---

# Task 01: Project Skeleton, HTTP Server & Telegram Auth

**Context:**
We are starting the development of the Telegram Prediction Market Bot. The goal of this task is to set up the basic project infrastructure in Go. The application must run inside Docker, serve static Frontend files, and be able to authenticate Telegram users via the `initData` protocol.

**Tech Stack:** Go 1.22+, Vanilla JS, Docker.

## 1. Project Structure

Create the following directory structure:

```text
/
├── cmd/bot/main.go           # Application entry point
├── internal/
│   ├── config/               # Environment variable loading
│   ├── transport/http/       # HTTP handlers and middleware
│   └── transport/telegram/   # Telegram Bot logic
├── web/                      # Frontend
│   ├── index.html            # "Hello World" page
│   └── app.js                # JS to handle initData and API testing
├── go.mod                    # Go module definition
├── Dockerfile                # Multi-stage build
└── docker-compose.yml        # Orchestration

```

## 2. Sub-tasks

### A. HTTP Server & Static Files

* Implement an HTTP server (standard `net/http` is preferred).
* The server must serve static files from the `web/` directory at the root path `/`.
* Add a test endpoint `/api/ping` that returns JSON: `{"status": "ok", "user_id": 12345}` (user_id comes from auth).

### B. Telegram Authentication Middleware

* Implement an HTTP Middleware that validates the `X-Telegram-Init-Data` header.
* **Validation Logic:**
1. Parse the `initData` string (query parameters).
2. Validate the cryptographic signature (HMAC-SHA256).
* **Secret Key generation:** `HMAC_SHA256("WebAppData", BOT_TOKEN)`.
* **Data Check String:** All key-value pairs (except hash) sorted alphabetically and joined by `\n`.
* **Comparison:** `HMAC_SHA256(DataCheckString, SecretKey)` must match the received `hash`.


3. Check `auth_date`: reject requests if the data is older than 24 hours.


* If validation fails: Return `401 Unauthorized`.
* If validation succeeds: Extract the `user.id` and inject it into the request Context.

### C. Telegram Bot Listener

* Use a Go library (recommended: `gopkg.in/telebot.v3`).
* On startup, the app must connect to the Telegram API (Long Polling).
* Handle the `/start` command: The bot should reply with a welcome message and an **Inline Button** that opens the Web App.

### D. Frontend (Minimal)

* **index.html:** Display a simple text: "Prediction Market Loading...".
* **app.js:** On page load:
1. Retrieve `window.Telegram.WebApp.initData`.
2. Perform a `fetch('/api/ping')` request, including the `X-Telegram-Init-Data` header.
3. If the server responds with 200 OK, change the page text to: "Auth Success! Welcome, [User Name]".
4. If the server responds with 401, show "Auth Failed".



### E. Docker Environment

* Create a `Dockerfile` (use `golang:alpine` for the build stage and `alpine` for the runtime stage).
* Create a `docker-compose.yml` that exposes port `8080` and passes the `TELEGRAM_BOT_TOKEN` environment variable.

## 3. Definition of Done

1. Running `docker-compose up` successfully builds and starts the container.
2. Sending `/start` to the bot in Telegram triggers a reply with the Web App button.
3. Opening the Web App inside Telegram loads the page.
4. The page text updates to "Auth Success", confirming that the JS sent the data and the Go backend successfully validated the cryptographic signature.
5. No errors in the container logs.