package bot

import (
	"log"
	"os"

	"gopkg.in/telebot.v3"
)

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

		// Send welcome message with button
		return c.Send("Welcome to the Prediction Market! 🎉\n\nMake predictions on various topics and win rewards. Click the button below to start:", &telebot.ReplyMarkup{
			InlineKeyboard: [][]telebot.InlineButton{
				{btn},
			},
		})
	})

	log.Println("Bot started. Use /start command to test.")

	// Start polling for updates
	b.Start()
}
