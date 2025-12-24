Task 10: Public News Channel (Broadcasting)
Context: To keep the community alive and engaged, we need a centralized "Public Square". The Bot will broadcast significant events (New Markets, Resolutions, Disputes) to a specific public Telegram Channel or Group. This serves as a news feed and a notification system for all users.

Tech Stack: Go (Telegram Bot API).

1. Channel Configuration
Create a new Telegram Channel (e.g., "@PredictionBotNews").
Add the Bot as an Admin to this channel.
Env Var: CHANNEL_ID (e.g., "-1001234567890" or "@mychannel").

2. Broadcasting Logic (internal/service/broadcaster.go)
Create a Broadcaster service.
Events to Broadcast:

New Market Created:
"🆕 New Market: Will Bitcoin hit $100k?
💰 Pool: 0 WSC
⏰ Ends: 2023-12-31"

Market Resolved:
"🏁 Market Resolved: Will it rain?
✅ Outcome: YES
🏆 Winners shared 5000 WSC!"

Dispute Raised (Optional, maybe too noisy? Let's skip for now or keep it brief).

Implementation:
Use bot.Send(recipient, message).
Recipient is the Channel ID.

3. Integration points
Call broadcaster.PublishNewMarket(market) in CreateMarket handler.
Call broadcaster.PublishResolution(market, payoutInfo) in Resolve/Finalize handler.

4. Definition of Done
Config: CHANNEL_ID is set.
Action: Creating a market via the Web App causes a message to appear in the Telegram Channel.
Action: Resolving a market causes a result message in the Channel.
Latency: Message appears within a few seconds.