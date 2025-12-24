package bot

import (
	"fmt"
	"log"
	"os"
	"strings"

	"predictionbot/internal/logger"
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
			"/me - View your profile information\n" +
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
		logger.Debug(telegramID, "profile_displayed", fmt.Sprintf("user_id=%d balance=%d", user.ID, user.Balance))
		return c.Send(profileText, &telebot.SendOptions{
			ParseMode: telebot.ModeMarkdown,
		})
	})

	log.Println("Bot started. Use /start command to test.")

	// Start polling for updates
	b.Start()
}
