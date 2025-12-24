package service

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"

	"predictionbot/internal/logger"
	"predictionbot/internal/storage"

	"gopkg.in/telebot.v3"
)

// NotificationService handles sending Telegram notifications
type NotificationService struct {
	bot     *telebot.Bot
	mu      sync.Mutex
	adminID int64
}

// NewNotificationService creates a new notification service
func NewNotificationService() (*NotificationService, error) {
	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	if botToken == "" {
		return nil, fmt.Errorf("TELEGRAM_BOT_TOKEN not set")
	}

	b, err := telebot.NewBot(telebot.Settings{
		Token: botToken,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create bot: %w", err)
	}

	// Get admin ID from environment
	adminIDStr := os.Getenv("ADMIN_TELEGRAM_ID")
	var adminID int64
	if adminIDStr != "" {
		adminID, _ = strconv.ParseInt(adminIDStr, 10, 64)
	}

	return &NotificationService{
		bot:     b,
		adminID: adminID,
	}, nil
}

// formatBalance converts cents to WSC format
func formatBalance(cents int64) string {
	wsc := float64(cents) / 100.0
	return fmt.Sprintf("%.2f WSC", wsc)
}

// SendWinNotification sends a notification to a user when they win
func (s *NotificationService) SendWinNotification(userID int64, marketID int64, question string, payout int64, newBalance int64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Get user by internal ID to get telegram ID
	user, err := storage.GetUserByID(userID)
	if err != nil || user == nil {
		logger.Debug(userID, "notification_error", "failed to get user for win notification")
		return
	}

	message := fmt.Sprintf("🏆 Congratulations! You won %s on market '#%d %s'. New Balance: %s",
		formatBalance(payout),
		marketID,
		truncateString(question, 50),
		formatBalance(newBalance))

	_, err = s.bot.Send(&telebot.User{ID: user.TelegramID}, message)
	if err != nil {
		logger.Debug(userID, "notification_error", fmt.Sprintf("failed to send win notification: %v", err))
		log.Printf("Failed to send win notification to user %d: %v", user.TelegramID, err)
	} else {
		logger.Debug(userID, "win_notification_sent", fmt.Sprintf("market_id=%d payout=%d", marketID, payout))
	}
}

// SendRefundNotification sends a notification to a user when they get a refund
func (s *NotificationService) SendRefundNotification(userID int64, marketID int64, question string, amount int64, newBalance int64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	user, err := storage.GetUserByID(userID)
	if err != nil || user == nil {
		logger.Debug(userID, "notification_error", "failed to get user for refund notification")
		return
	}

	message := fmt.Sprintf("💰 Refund received: %s has been returned for market '#%d %s'. New Balance: %s",
		formatBalance(amount),
		marketID,
		truncateString(question, 50),
		formatBalance(newBalance))

	_, err = s.bot.Send(&telebot.User{ID: user.TelegramID}, message)
	if err != nil {
		logger.Debug(userID, "notification_error", fmt.Sprintf("failed to send refund notification: %v", err))
		log.Printf("Failed to send refund notification to user %d: %v", user.TelegramID, err)
	}
}

// SendDisputeAlert sends an alert to the admin when a dispute is raised
func (s *NotificationService) SendDisputeAlert(marketID int64, question string, disputeUserID int64) {
	if s.adminID == 0 {
		log.Printf("Admin ID not set, skipping dispute alert for market #%d", marketID)
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	message := fmt.Sprintf("⚠️ Dispute Raised!\n\nMarket ID: #%d\nQuestion: %s\nDisputed by user ID: %d\n\nPlease review and resolve.",
		marketID,
		truncateString(question, 100),
		disputeUserID)

	_, err := s.bot.Send(&telebot.User{ID: s.adminID}, message)
	if err != nil {
		logger.Debug(disputeUserID, "notification_error", fmt.Sprintf("failed to send dispute alert: %v", err))
		log.Printf("Failed to send dispute alert to admin %d: %v", s.adminID, err)
	} else {
		logger.Debug(disputeUserID, "dispute_alert_sent", fmt.Sprintf("market_id=%d", marketID))
	}
}

// SendLossNotification sends a notification to a user when they lose
func (s *NotificationService) SendLossNotification(userID int64, marketID int64, question string, amount int64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	user, err := storage.GetUserByID(userID)
	if err != nil || user == nil {
		logger.Debug(userID, "notification_error", "failed to get user for loss notification")
		return
	}

	message := fmt.Sprintf("📉 Market resolved: Your bet of %s on market '#%d %s' did not win.",
		formatBalance(amount),
		marketID,
		truncateString(question, 50))

	_, err = s.bot.Send(&telebot.User{ID: user.TelegramID}, message)
	if err != nil {
		logger.Debug(userID, "notification_error", fmt.Sprintf("failed to send loss notification: %v", err))
	}
}

// truncateString truncates a string to maxLen and adds ellipsis if needed
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return strings.TrimSpace(s[:maxLen-3]) + "..."
}

// GetBot returns the underlying telebot instance (for bot commands)
func (s *NotificationService) GetBot() *telebot.Bot {
	return s.bot
}
