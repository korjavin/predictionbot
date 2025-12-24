Это прекрасная идея. Это добавляет элемент "Social Proof" (Социального доказательства) и FOMO (Fear Of Missing Out). Когда пользователи видят в общем канале, что "Вася только что выиграл 5000 монет" или "Появился новый интересный рынок", это побуждает их зайти в бота.

По сути, мы делаем ленту новостей (News Feed), но не внутри Web App, а на уровне нативного Telegram-канала, что гораздо эффективнее для возврата пользователей (Retention).

Вот описание Task 10.

Task 10: Public News Channel (Broadcasting)
Context: To keep the community alive and engaged, we need a centralized "Public Square". The Bot will broadcast significant events (New Markets, Resolutions, Disputes) to a specific public Telegram Channel or Group. This serves as a news feed and a notification system for all users.

Tech Stack: Go (Telebot), Environment Variables.

1. Configuration (.env)
Add a new variable: TELEGRAM_NEWS_CHANNEL_ID.

Example: -1001234567890 (Telegram Channel IDs usually start with -100).

Requirement: The Bot must be added to this channel/group as an Administrator (with "Post Messages" permission).

2. Backend Infrastructure (internal/service/notification)
A. Broadcaster Service
Extend the NotificationService created in Task 07.

Implement Broadcast(text string, options ...interface{}) error.

Logic:

Check if TELEGRAM_NEWS_CHANNEL_ID is set. If empty, do nothing (feature disabled).

Use the Bot API to send a message to that ID.

Important: Run this in a goroutine so that sending a message to Telegram doesn't slow down the HTTP response for the user creating the market.

3. Event Triggers (Integration points)
You need to inject the Broadcaster into your business services and trigger messages on specific events:

A. New Market Created (MarketService.Create)
Trigger: When a user successfully creates a market.

Message Template:

🆕 New Market Created!

❓ Question: Will Bitcoin hit $100k? ⏰ Deadline: 24 hours left

👉 Open the App to place your bets!

B. Market Finalized / Payout (PayoutService)
Trigger: When a market transitions to FINALIZED.

Message Template:

🏁 Market Resolved

❓ "Will Bitcoin hit $100k?" ✅ Outcome: YES 💰 Total Pool: 50,000 WSC distributed to winners!

C. Dispute Raised (DisputeService)
Trigger: When a user clicks "Dispute".

Message Template:

⚖️ Dispute Alert!

The result of market "Will it rain?" is being challenged. The High Court (Admin) will review it shortly.

4. Definition of Done
Config: The app reads the Channel ID from .env.

Permissions: The bot successfully posts to the channel (assuming correct admin rights).

Async: If the Telegram API is slow or down, the User's request (e.g., creating a market) does not fail or hang. The broadcast happens in the background.

Formatting: Messages use clean formatting (Bold, Emojis) to look professional.

