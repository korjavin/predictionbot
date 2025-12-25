package bot

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"predictionbot/internal/logger"
	"predictionbot/internal/service"
	"predictionbot/internal/storage"

	"gopkg.in/telebot.v3"
)

// formatBalance formats balance as WSC
func formatBalance(balance int64) string {
	return fmt.Sprintf("%d WSC", balance)
}

// escapeMarkdown escapes special characters for Telegram Markdown mode
func escapeMarkdown(s string) string {
	escaped := s
	escaped = strings.ReplaceAll(escaped, `\`, `\\`)
	escaped = strings.ReplaceAll(escaped, "*", `\*`)
	escaped = strings.ReplaceAll(escaped, "_", `\_`)
	escaped = strings.ReplaceAll(escaped, "`", `\`)
	escaped = strings.ReplaceAll(escaped, "[", `\[`)
	escaped = strings.ReplaceAll(escaped, "]", `\]`)
	escaped = strings.ReplaceAll(escaped, "(", `\(`)
	escaped = strings.ReplaceAll(escaped, ")", `\)`)
	escaped = strings.ReplaceAll(escaped, ">", `\>`)
	escaped = strings.ReplaceAll(escaped, "#", `\#`)
	escaped = strings.ReplaceAll(escaped, "+", `\+`)
	escaped = strings.ReplaceAll(escaped, "-", `\-`)
	escaped = strings.ReplaceAll(escaped, "=", `\=`)
	escaped = strings.ReplaceAll(escaped, "|", `\|`)
	escaped = strings.ReplaceAll(escaped, ".", `\.`)
	escaped = strings.ReplaceAll(escaped, "!", `\!`)
	return escaped
}

// StartBot initializes and starts the Telegram bot
func StartBot() {
	// Get bot token from environment
	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	if botToken == "" {
		log.Fatal("TELEGRAM_BOT_TOKEN not set")
	}

	// Create bot
	b, err := telebot.NewBot(telebot.Settings{
		Token: botToken,
		Poller: &telebot.LongPoller{
			Timeout: 10,
		},
	})
	if err != nil {
		log.Fatalf("Failed to create bot: %v", err)
	}

	// Register /start command handler
	b.Handle("/start", func(c telebot.Context) error {
		telegramID := c.Sender().ID
		logger.Debug(telegramID, "command_start", fmt.Sprintf("username=%s first_name=%s", c.Sender().Username, c.Sender().FirstName))

		// Get or create user
		user, err := storage.GetUserByTelegramID(telegramID)
		if err != nil {
			logger.Debug(telegramID, "error", fmt.Sprintf("failed to get user: %v", err))
			return c.Send("Error retrieving user data. Please try again.")
		}
		if user == nil {
			// Create new user
			user, err = storage.CreateUser(
				telegramID,
				c.Sender().Username,
				c.Sender().FirstName,
			)
			if err != nil {
				logger.Debug(telegramID, "error", fmt.Sprintf("failed to create user: %v", err))
				return c.Send("Error creating user. Please try again.")
			}
			logger.Debug(telegramID, "user_created", fmt.Sprintf("welcome_bonus=1000 user_id=%d", user.ID))
		}

		// Get the web app URL from environment or use default
		webAppURL := os.Getenv("WEB_APP_URL")
		if webAppURL == "" {
			webAppURL = "http://localhost:8080"
		}

		// Create Web App button
		btn := telebot.InlineButton{
			Text:   "🎯 Open Prediction Market",
			WebApp: &telebot.WebApp{URL: webAppURL},
		}

		// Send welcome message with user info
		welcomeMsg := fmt.Sprintf("Welcome to the Prediction Market! 🎉\n\nHi, %s! You have %s.\n\nMake predictions on various topics and win rewards. Click the button below to start:",
			user.FirstName, formatBalance(user.Balance))
		logger.Debug(telegramID, "welcome_sent", fmt.Sprintf("balance=%d", user.Balance))
		return c.Send(welcomeMsg, &telebot.ReplyMarkup{
			InlineKeyboard: [][]telebot.InlineButton{
				{btn},
			},
		})
	})

	// Register /help command handler
	b.Handle("/help", func(c telebot.Context) error {
		telegramID := c.Sender().ID
		logger.Debug(telegramID, "command_help", "")
		helpText := "📚 *Available Commands*\n\n" +
			"/start - Start the bot and receive your welcome bonus\n" +
			"/balance - Check your current balance\n" +
			"/me - View your profile and bet history\n" +
			"/mybets - View your active bets\n" +
			"/list - View all active markets\n" +
			"/help - Show this help message\n\n" +
			"🎯 Open the Prediction Market web app to create markets and place bets!"
		return c.Send(helpText, &telebot.SendOptions{
			ParseMode: telebot.ModeMarkdown,
		})
	})

	// Register /balance command handler
	b.Handle("/balance", func(c telebot.Context) error {
		telegramID := c.Sender().ID
		logger.Debug(telegramID, "command_balance", "")

		user, err := storage.GetUserByTelegramID(telegramID)
		if err != nil {
			logger.Debug(telegramID, "error", fmt.Sprintf("failed to get user: %v", err))
			return c.Send("Error retrieving user data. Please try again.")
		}
		if user == nil {
			logger.Debug(telegramID, "error", "user_not_found")
			return c.Send("You haven't started the bot yet. Use /start to create your account!")
		}

		balanceText := fmt.Sprintf("💰 *Your Balance*\n\n"+
			"Current Balance: %s\n"+
			"\nUse the Prediction Market web app to place bets!",
			formatBalance(user.Balance))
		logger.Debug(telegramID, "balance_displayed", fmt.Sprintf("balance=%d", user.Balance))
		return c.Send(balanceText, &telebot.SendOptions{
			ParseMode: telebot.ModeMarkdown,
		})
	})

	// Register /me command handler
	b.Handle("/me", func(c telebot.Context) error {
		telegramID := c.Sender().ID
		logger.Debug(telegramID, "command_me", "")

		user, err := storage.GetUserByTelegramID(telegramID)
		if err != nil {
			logger.Debug(telegramID, "error", fmt.Sprintf("failed to get user: %v", err))
			return c.Send("Error retrieving user data. Please try again.")
		}
		if user == nil {
			logger.Debug(telegramID, "error", "user_not_found")
			return c.Send("You haven't started the bot yet. Use /start to create your account!")
		}

		// Build profile section
		profileText := fmt.Sprintf("👤 *Your Profile*\n\n"+
			"Name: %s\n"+
			"Username: @%s\n"+
			"Balance: %s\n"+
			"Member since: %s",
			user.FirstName,
			func() string {
				if user.Username != "" {
					return escapeMarkdown(user.Username)
				} else {
					return "N/A"
				}
			}(),
			formatBalance(user.Balance),
			user.CreatedAt.Format("January 2, 2006"))

		// Get user stats
		stats, err := storage.GetUserStats(user.ID)
		if err != nil {
			logger.Debug(telegramID, "error", fmt.Sprintf("failed to get user stats: %v", err))
			stats = &storage.UserStats{}
		}

		// Build stats section
		winRatePercent := float64(0)
		if stats.TotalBets > 0 {
			winRatePercent = stats.WinRate * 100
		}
		statsText := fmt.Sprintf("\n📊 *Your Stats*\n\n"+
			"Total Bets: %d\n"+
			"🟢 Wins: %d\n"+
			"🔴 Losses: %d\n"+
			"📈 Win Rate: %.1f%%",
			stats.TotalBets, stats.Wins, stats.Losses, winRatePercent)

		// Get user bets
		bets, err := storage.GetUserBets(user.ID)
		if err != nil {
			logger.Debug(telegramID, "error", fmt.Sprintf("failed to get user bets: %v", err))
			bets = []storage.BetHistoryItem{}
		}

		// Build bet history section
		var historyText string
		if len(bets) == 0 {
			historyText = "\n\n🎲 *Bet History*\n\nNo bets placed yet. Open the web app to start betting!"
		} else {
			historyText = "\n\n🎲 *Recent Bets*\n"

			// Limit to last 10 bets
			maxBets := 10
			if len(bets) < maxBets {
				maxBets = len(bets)
			}

			for i := 0; i < maxBets; i++ {
				bet := bets[i]

				// Truncate long questions
				question := bet.Question
				if len(question) > 40 {
					question = question[:37] + "..."
				}

				// Format status with emoji
				var statusEmoji, statusText string
				switch bet.Status {
				case storage.BetStatusPending:
					statusEmoji = "⏳"
					statusText = "PENDING"
				case storage.BetStatusWon:
					statusEmoji = "✅"
					statusText = "WON"
				case storage.BetStatusLost:
					statusEmoji = "❌"
					statusText = "LOST"
				case storage.BetStatusRefunded:
					statusEmoji = "🔄"
					statusText = "REFUNDED"
				}

				// Outcome emoji
				outcomeEmoji := "✅"
				if bet.OutcomeChosen == "NO" {
					outcomeEmoji = "🔴"
				}

				// Format payout
				payoutText := ""
				if bet.Status == storage.BetStatusWon && bet.Payout > 0 {
					payoutText = fmt.Sprintf(" | 💰 Payout: %d WSC", bet.Payout)
				}

				historyText += fmt.Sprintf("\n*%d.* %s\n"+
					"   📝 %s\n"+
					"   🎯 %s %s | %d WSC%s\n"+
					"   %s %s",
					i+1,
					statusEmoji,
					escapeMarkdown(question),
					outcomeEmoji,
					bet.OutcomeChosen,
					bet.Amount,
					payoutText,
					statusEmoji,
					statusText)
			}

			if len(bets) > maxBets {
				historyText += fmt.Sprintf("\n\n...and %d more bets", len(bets)-maxBets)
			}
		}

		// Combine all sections
		fullText := profileText + statsText + historyText

		logger.Debug(telegramID, "profile_displayed", fmt.Sprintf("user_id=%d balance=%d bets=%d", user.ID, user.Balance, len(bets)))
		return c.Send(fullText, &telebot.SendOptions{
			ParseMode: telebot.ModeMarkdown,
		})
	})

	// Register /list command handler
	b.Handle("/list", func(c telebot.Context) error {
		telegramID := c.Sender().ID
		logger.Debug(telegramID, "command_list", "")

		// Get all active markets with creator info
		markets, err := storage.ListActiveMarketsWithCreator()
		if err != nil {
			logger.Debug(telegramID, "error", fmt.Sprintf("failed to list markets: %v", err))
			return c.Send("Error retrieving markets. Please try again.")
		}

		// Handle empty list case
		if len(markets) == 0 {
			noMarketsText := "📊 *Active Markets*\n\n" +
				"No active markets at the moment.\n" +
				"Open the Prediction Market web app to create one!"
			return c.Send(noMarketsText, &telebot.SendOptions{
				ParseMode: telebot.ModeMarkdown,
			})
		}

		// Format the list of markets
		var listText string
		if telegramID == c.Sender().ID {
			listText = fmt.Sprintf("📊 *Active Markets* (%d)\n\n", len(markets))
		} else {
			listText = fmt.Sprintf("📊 *Active Markets* (%d)\n\n", len(markets))
		}

		for i, market := range markets {
			// Truncate long questions
			question := market.Question
			if len(question) > 50 {
				question = question[:47] + "..."
			}

			// Format pool amounts
			poolYes := market.PoolYes
			poolNo := market.PoolNo

			// Escape special characters in question
			escapedQuestion := escapeMarkdown(question)

			// Add market entry
			listText += fmt.Sprintf("*%d.* %s\n"+
				"   👤 %s\n"+
				"   💰 YES: %d | NO: %d\n"+
				"   ⏰ %s\n\n",
				i+1,
				escapedQuestion,
				escapeMarkdown(market.CreatorName),
				poolYes,
				poolNo,
				market.ExpiresAt)
		}

		// Add footer with instruction
		listText += "Use the Prediction Market web app to place bets!"

		logger.Debug(telegramID, "list_displayed", fmt.Sprintf("markets_count=%d", len(markets)))
		return c.Send(listText, &telebot.SendOptions{
			ParseMode: telebot.ModeMarkdown,
		})
	})

	// Register /mybets command handler
	b.Handle("/mybets", func(c telebot.Context) error {
		telegramID := c.Sender().ID
		logger.Debug(telegramID, "command_mybets", "")

		// Get user
		user, err := storage.GetUserByTelegramID(telegramID)
		if err != nil {
			logger.Debug(telegramID, "error", fmt.Sprintf("failed to get user: %v", err))
			return c.Send("Error retrieving user data. Please try again.")
		}
		if user == nil {
			logger.Debug(telegramID, "error", "user_not_found")
			return c.Send("You haven't started the bot yet. Use /start to create your account!")
		}

		// Get user's active bets
		bets, err := storage.GetUserActiveBets(user.ID)
		if err != nil {
			logger.Debug(telegramID, "error", fmt.Sprintf("failed to get active bets: %v", err))
			return c.Send("Error retrieving your bets. Please try again.")
		}

		// Handle empty list case
		if len(bets) == 0 {
			noBetsText := "🎯 *Your Active Bets*\n\n" +
				"You haven't placed any bets on active markets yet.\n" +
				"Open the Prediction Market web app to place a bet!"
			return c.Send(noBetsText, &telebot.SendOptions{
				ParseMode: telebot.ModeMarkdown,
			})
		}

		// Format the list of active bets
		mybetsText := fmt.Sprintf("🎯 *Your Active Bets* (%d)\n\n", len(bets))

		for i, bet := range bets {
			// Truncate long questions
			question := bet.Question
			if len(question) > 40 {
				question = question[:37] + "..."
			}

			// Calculate odds (simple pool-based odds)
			totalPool := bet.PoolYes + bet.PoolNo
			odds := float64(50)
			if totalPool > 0 {
				if bet.OutcomeChosen == "YES" {
					odds = float64(bet.PoolNo) / float64(totalPool) * 100
				} else {
					odds = float64(bet.PoolYes) / float64(totalPool) * 100
				}
			}

			// Calculate potential payout
			potentialPayout := bet.Amount
			if odds > 0 && odds < 100 {
				potentialPayout = bet.Amount * int64(100/odds)
			}

			// Outcome emoji
			outcomeEmoji := "✅"
			if bet.OutcomeChosen == "NO" {
				outcomeEmoji = "🔴"
			}

			mybetsText += fmt.Sprintf("*%d.* %s\n"+
				"   📝 %s\n"+
				"   🎯 %s %s | %d WSC\n"+
				"   💰 Pool: %d/%d | 🎲 %d%%\n"+
				"   💸 Potential: %d WSC\n"+
				"   ⏰ Expires: %s\n\n",
				i+1,
				escapeMarkdown(question),
				escapeMarkdown(question),
				outcomeEmoji,
				bet.OutcomeChosen,
				bet.Amount,
				bet.PoolYes,
				bet.PoolNo,
				int(odds),
				potentialPayout,
				bet.ExpiresAt)
		}

		// Add footer
		mybetsText += "Open the web app to manage your bets!"

		logger.Debug(telegramID, "mybets_displayed", fmt.Sprintf("bets_count=%d", len(bets)))
		return c.Send(mybetsText, &telebot.SendOptions{
			ParseMode: telebot.ModeMarkdown,
		})
	})

	// Register /resolve_yes command handler
	b.Handle("/resolve_yes", func(c telebot.Context) error {
		telegramID := c.Sender().ID
		logger.Debug(telegramID, "command_resolve_yes", "")

		// Parse market ID from command args
		args := c.Args()
		if len(args) < 1 {
			return c.Send("❌ *Usage:* /resolve_yes <market_id>\n\nPlease provide the market ID to resolve as YES.", &telebot.SendOptions{
				ParseMode: telebot.ModeMarkdown,
			})
		}

		marketID, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return c.Send("❌ *Invalid market ID*\n\nPlease provide a valid numeric market ID.", &telebot.SendOptions{
				ParseMode: telebot.ModeMarkdown,
			})
		}

		// Get user
		user, err := storage.GetUserByTelegramID(telegramID)
		if err != nil || user == nil {
			logger.Debug(telegramID, "error", "user_not_found")
			return c.Send("You haven't started the bot yet. Use /start to create your account!")
		}

		// Resolve market as YES
		payoutService := service.NewPayoutService()
		err = payoutService.ResolveMarket(context.Background(), marketID, user.ID, "YES")
		if err != nil {
			logger.Debug(telegramID, "resolve_error", fmt.Sprintf("market_id=%d error=%s", marketID, err.Error()))
			return c.Send(fmt.Sprintf("❌ *Resolution Failed*\n\n%s", err.Error()), &telebot.SendOptions{
				ParseMode: telebot.ModeMarkdown,
			})
		}

		logger.Debug(telegramID, "market_resolved_yes", fmt.Sprintf("market_id=%d", marketID))
		return c.Send(fmt.Sprintf("✅ *Market Resolved!*\n\nMarket #%d has been resolved as *YES*.\n\nPayouts will be distributed after the dispute period.", marketID), &telebot.SendOptions{
			ParseMode: telebot.ModeMarkdown,
		})
	})

	// Register /resolve_no command handler
	b.Handle("/resolve_no", func(c telebot.Context) error {
		telegramID := c.Sender().ID
		logger.Debug(telegramID, "command_resolve_no", "")

		// Parse market ID from command args
		args := c.Args()
		if len(args) < 1 {
			return c.Send("❌ *Usage:* /resolve_no <market_id>\n\nPlease provide the market ID to resolve as NO.", &telebot.SendOptions{
				ParseMode: telebot.ModeMarkdown,
			})
		}

		marketID, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return c.Send("❌ *Invalid market ID*\n\nPlease provide a valid numeric market ID.", &telebot.SendOptions{
				ParseMode: telebot.ModeMarkdown,
			})
		}

		// Get user
		user, err := storage.GetUserByTelegramID(telegramID)
		if err != nil || user == nil {
			logger.Debug(telegramID, "error", "user_not_found")
			return c.Send("You haven't started the bot yet. Use /start to create your account!")
		}

		// Resolve market as NO
		payoutService := service.NewPayoutService()
		err = payoutService.ResolveMarket(context.Background(), marketID, user.ID, "NO")
		if err != nil {
			logger.Debug(telegramID, "resolve_error", fmt.Sprintf("market_id=%d error=%s", marketID, err.Error()))
			return c.Send(fmt.Sprintf("❌ *Resolution Failed*\n\n%s", err.Error()), &telebot.SendOptions{
				ParseMode: telebot.ModeMarkdown,
			})
		}

		logger.Debug(telegramID, "market_resolved_no", fmt.Sprintf("market_id=%d", marketID))
		return c.Send(fmt.Sprintf("✅ *Market Resolved!*\n\nMarket #%d has been resolved as *NO*.\n\nPayouts will be distributed after the dispute period.", marketID), &telebot.SendOptions{
			ParseMode: telebot.ModeMarkdown,
		})
	})

	// Register /my_markets command handler
	b.Handle("/my_markets", func(c telebot.Context) error {
		telegramID := c.Sender().ID
		logger.Debug(telegramID, "command_my_markets", "")

		// Get user
		user, err := storage.GetUserByTelegramID(telegramID)
		if err != nil {
			logger.Debug(telegramID, "error", fmt.Sprintf("failed to get user: %v", err))
			return c.Send("Error retrieving user data. Please try again.")
		}
		if user == nil {
			logger.Debug(telegramID, "error", "user_not_found")
			return c.Send("You haven't started the bot yet. Use /start to create your account!")
		}

		// Get user's markets
		markets, err := storage.GetMarketsByCreator(user.ID)
		if err != nil {
			logger.Debug(telegramID, "error", fmt.Sprintf("failed to get markets: %v", err))
			return c.Send("Error retrieving your markets. Please try again.")
		}

		// Handle empty list case
		if len(markets) == 0 {
			noMarketsText := "📊 *Your Markets*\n\n" +
				"You haven't created any markets yet.\n" +
				"Open the Prediction Market web app to create one!"
			return c.Send(noMarketsText, &telebot.SendOptions{
				ParseMode: telebot.ModeMarkdown,
			})
		}

		// Format the list of markets
		myMarketsText := fmt.Sprintf("📊 *Your Markets* (%d)\n\n", len(markets))

		for i, market := range markets {
			// Truncate long questions
			question := market.Question
			if len(question) > 40 {
				question = question[:37] + "..."
			}

			// Format status emoji
			var statusEmoji, statusText string
			switch market.Status {
			case "ACTIVE":
				statusEmoji = "🟢"
				statusText = "ACTIVE"
			case "LOCKED":
				statusEmoji = "🔒"
				statusText = "LOCKED"
			case "RESOLVED":
				statusEmoji = "✅"
				statusText = fmt.Sprintf("RESOLVED %s", market.Outcome)
			case "FINALIZED":
				statusEmoji = "🏁"
				statusText = fmt.Sprintf("FINALIZED %s", market.Outcome)
			case "DISPUTED":
				statusEmoji = "⚠️"
				statusText = "DISPUTED"
			}

			myMarketsText += fmt.Sprintf("*%d.* %s\n"+
				"   📝 %s\n"+
				"   %s %s | 💰 %d/%d\n"+
				"   ⏰ %s\n\n",
				i+1,
				statusEmoji,
				escapeMarkdown(question),
				statusEmoji,
				statusText,
				market.PoolYes,
				market.PoolNo,
				market.ExpiresAt)
		}

		// Add footer with resolution commands for locked markets
		myMarketsText += "💡 Use /resolve_yes <id> or /resolve_no <id> to resolve locked markets."

		logger.Debug(telegramID, "my_markets_displayed", fmt.Sprintf("markets_count=%d", len(markets)))
		return c.Send(myMarketsText, &telebot.SendOptions{
			ParseMode: telebot.ModeMarkdown,
		})
	})

	log.Println("Bot started. Use /start command to test.")

	// Start polling for updates
	b.Start()
}
