# 🔮 Telegram Prediction Market Bot (Educational Project)

## About
This is an open-source educational project implementing a **Prediction Market** within Telegram.
The project is designed to learn **Go (Golang)**, **Telegram Web Apps (TWA)** mechanics, and web service architecture.

**Core Concept:** Users can place bets on the outcome of future events using a virtual in-game currency called **WiseCoin (WSC)**.

## 🚀 How It Works

### 1. Onboarding
When a user launches the bot, they automatically receive starting capital (e.g., **1000 WSC**). The interface is implemented as a **Web App**—a website opening directly inside Telegram, providing a seamless, native-app-like user experience.

### 2. Creating Markets
Any user can create a prediction market.
* **Example:** "Will it snow in New York on December 31st?"
* **Conditions:** The creator sets the deadline for placing bets and the date when the event will be resolved.

### 3. Betting
The project uses the **Parimutuel Betting (Pool System)** mechanic:
* All bets on a specific market are aggregated into a single pool.
* Once the event occurs, the pool is distributed among the winners proportional to their contribution.
* Odds are not fixed at the time of the bet; they are determined by the final distribution of funds in the pool.

> *Example:* If 1000 coins are bet on "YES" and 500 coins on "NO", and "YES" wins, the total pool (1500) is shared among the "YES" bettors, taking the money from the "NO" side.

### 4. Oracle & Dispute Mechanism
To simplify the architecture, we use a two-step "Social Consensus" system:
1.  **Resolution:** After the event date passes, the **Market Creator** is responsible for setting the outcome (YES/NO).
2.  **Dispute Period:** Once a verdict is set, a 24-hour appeal window opens. If users disagree with the creator's decision, they can raise a dispute.
3.  **Judgement:** In case of a dispute, the Moderator (Bot Owner) intervenes to make the final decision and penalize dishonest creators.

## 🛠 Tech Stack
* **Language:** Go (Golang)
* **Database:** SQLite
* **Frontend:** Vanilla JS + HTML5 (Telegram Web App)
* **Deployment:** Docker + Portainer

---
*Developed for educational purposes.*
