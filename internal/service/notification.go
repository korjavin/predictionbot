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

// Global notification service instance
var globalNotificationService *NotificationService

// SetNotificationService sets the global notification service
func SetNotificationService(ns *NotificationService) {
	globalNotificationService = ns
}

// GetNotificationService returns the global notification service
func GetNotificationService() *NotificationService {
	return globalNotificationService
}

// NotificationService handles sending Telegram notifications
type NotificationService struct {
	bot       *telebot.Bot
	mu        sync.Mutex
	adminID   int64
	channelID string
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

	// Get channel ID from environment
	channelID := os.Getenv("CHANNEL_ID")

	return &NotificationService{
		bot:       b,
		adminID:   adminID,
		channelID: channelID,
	}, nil
}

// formatBalance formats balance as WSC
func formatBalance(balance int64) string {
	return fmt.Sprintf("%d WSC", balance)
}

// SendWinNotification sends a notification to a user when they win
func (s *NotificationService) SendWinNotification(userID int64, marketID int64, question string, betAmount int64, outcome string, payout int64, newBalance int64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Get user by internal ID to get telegram ID
	user, err := storage.GetUserByID(userID)
	if err != nil || user == nil {
		logger.Debug(userID, "notification_error", "failed to get user for win notification")
		return
	}

	profit := payout - betAmount
	message := fmt.Sprintf("🏆 You won %s on market #%d\n\n📝 %s\n\nYour bet: %s on %s\nPayout: %s\nProfit: %s\nNew Balance: %s",
		formatBalance(profit),
		marketID,
		truncateString(question, 50),
		formatBalance(betAmount),
		outcome,
		formatBalance(payout),
		formatBalance(profit),
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

	message := fmt.Sprintf("⚠️ Dispute Raised!\n\nMarket ID: #%d\nQuestion: %s\nDisputed by user ID: %d\n\nUse /resolve_disputes to review and resolve.",
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

// NotifyMarketCreatorDeadline sends a DM to the market creator when their market expires
func (s *NotificationService) NotifyMarketCreatorDeadline(market *storage.Market) {
	if market == nil {
		return
	}

	// Get the creator's user record
	user, err := storage.GetUserByID(market.CreatorID)
	if err != nil || user == nil {
		logger.Debug(market.CreatorID, "notification_error", "failed to get market creator")
		return
	}

	if user.TelegramID == 0 {
		logger.Debug(market.CreatorID, "notification_error", "creator has no telegram_id")
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Format the deadline notification message
	message := fmt.Sprintf("⏰ *Market Deadline Reached*\n\nYour market '#%d %s' has reached its deadline and is now locked.\n\n"+
		"Please resolve it to distribute winnings:\n"+
		"• Use the web app to resolve\n"+
		"• Or use commands: /resolve_yes %d or /resolve_no %d",
		market.ID,
		truncateString(market.Question, 50),
		market.ID,
		market.ID)

	_, err = s.bot.Send(&telebot.User{ID: user.TelegramID}, message, &telebot.SendOptions{
		ParseMode: telebot.ModeMarkdown,
	})
	if err != nil {
		logger.Debug(market.CreatorID, "notification_error", fmt.Sprintf("failed to send deadline notification: %v", err))
		log.Printf("Failed to send deadline notification to user %d (telegram_id: %d): %v", market.CreatorID, user.TelegramID, err)
	} else {
		logger.Debug(market.CreatorID, "deadline_notification_sent", fmt.Sprintf("market_id=%d", market.ID))
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

// --- Broadcaster Methods for Public News Channel ---

// PublishNewMarket broadcasts a new market to the public channel
func (s *NotificationService) PublishNewMarket(market *storage.Market, creatorName string) {
	if s.channelID == "" {
		// Channel not configured, skip broadcasting
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Format expiration date
	expiresAt := market.ExpiresAt.Format("2006-01-02 15:04")

	message := fmt.Sprintf("🆕 *New Market Created*\n\n*#%d* %s\n\n👤 Creator: %s\n⏰ Ends: %s\n\n🎯 Place your bets!",
		market.ID,
		escapeMarkdown(market.Question),
		escapeMarkdown(creatorName),
		expiresAt)

	// Send to channel
	recipient := s.getChannelRecipient()
	_, err := s.bot.Send(recipient, message, &telebot.SendOptions{
		ParseMode: telebot.ModeMarkdown,
	})
	if err != nil {
		logger.Debug(0, "broadcast_error", fmt.Sprintf("failed to publish new market: %v", err))
		log.Printf("Failed to publish new market to channel %s: %v", s.channelID, err)
	} else {
		logger.Debug(0, "broadcast_new_market", fmt.Sprintf("market_id=%d", market.ID))
	}
}

// PublishResolution broadcasts a market resolution to the public channel
func (s *NotificationService) PublishResolution(marketID int64, question string, outcome string, totalPool int64) {
	if s.channelID == "" {
		// Channel not configured, skip broadcasting
		logger.Debug(0, "broadcast_skipped", "CHANNEL_ID not configured")
		log.Printf("CHANNEL_ID not configured, skipping broadcast for market #%d", marketID)
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	logger.Debug(0, "broadcast_resolution_attempt", fmt.Sprintf("channel=%s market_id=%d outcome=%s pool=%d", s.channelID, marketID, outcome, totalPool))

	// Format outcome emoji
	outcomeEmoji := "✅"
	if outcome == "NO" {
		outcomeEmoji = "❌"
	}

	message := fmt.Sprintf("🏁 *Market Resolved*\n\n*#%d* %s\n\n%s Outcome: *%s*\n💰 Total Pool: %s\n\n⏰ *Dispute Period: 24 hours*\n\nIf you disagree with this outcome, use /dispute to raise a dispute\\.\nWinners will receive payouts after the dispute period ends\\.",
		marketID,
		escapeMarkdown(truncateString(question, 80)),
		outcomeEmoji,
		outcome,
		formatBalance(totalPool))

	logger.Debug(0, "broadcast_message_prepared", fmt.Sprintf("length=%d", len(message)))

	// Send to channel
	recipient := s.getChannelRecipient()
	_, err := s.bot.Send(recipient, message, &telebot.SendOptions{
		ParseMode: telebot.ModeMarkdown,
	})
	if err != nil {
		logger.Debug(0, "broadcast_error", fmt.Sprintf("channel=%s error=%v", s.channelID, err))
		log.Printf("Failed to publish resolution to channel %s: %v", s.channelID, err)
	} else {
		logger.Debug(0, "broadcast_resolution", fmt.Sprintf("market_id=%d outcome=%s channel=%s", marketID, outcome, s.channelID))
		log.Printf("Successfully published resolution for market #%d to channel %s", marketID, s.channelID)
	}
}

// getChannelRecipient returns the appropriate recipient for the configured channel
func (s *NotificationService) getChannelRecipient() telebot.Recipient {
	if strings.HasPrefix(s.channelID, "@") {
		return &telebot.Chat{Username: s.channelID}
	}
	return &telebot.Chat{ID: parseChannelID(s.channelID)}
}

// parseChannelID parses a channel ID string (supports numeric IDs)
func parseChannelID(channelID string) int64 {
	id, err := strconv.ParseInt(channelID, 10, 64)
	if err != nil {
		return 0
	}
	return id
}

// PublishDispute broadcasts a dispute notification to the public channel
func (s *NotificationService) PublishDispute(marketID int64, question string, outcome string) {
	if s.channelID == "" {
		logger.Debug(0, "broadcast_skipped", "CHANNEL_ID not configured")
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	logger.Debug(0, "broadcast_dispute_attempt", fmt.Sprintf("channel=%s market_id=%d", s.channelID, marketID))

	message := fmt.Sprintf("⚠️ *Dispute Raised*\n\n*#%d* %s\n\nA user has disputed the resolution of this market\\.\n\n💰 Payouts are frozen pending admin review\\.\nThe admin will review and make a final decision\\.",
		marketID,
		escapeMarkdown(truncateString(question, 80)))

	recipient := s.getChannelRecipient()
	_, err := s.bot.Send(recipient, message, &telebot.SendOptions{
		ParseMode: telebot.ModeMarkdown,
	})
	if err != nil {
		logger.Debug(0, "broadcast_error", fmt.Sprintf("channel=%s error=%v", s.channelID, err))
		log.Printf("Failed to publish dispute to channel %s: %v", s.channelID, err)
	} else {
		logger.Debug(0, "broadcast_dispute", fmt.Sprintf("market_id=%d channel=%s", marketID, s.channelID))
		log.Printf("Successfully published dispute for market #%d to channel %s", marketID, s.channelID)
	}
}

// PublishFinalization broadcasts market finalization and payout distribution
func (s *NotificationService) PublishFinalization(marketID int64, question string, outcome string, winnersCount int, totalPayout int64, wasDisputed bool) {
	if s.channelID == "" {
		logger.Debug(0, "broadcast_skipped", "CHANNEL_ID not configured")
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	logger.Debug(0, "broadcast_finalization_attempt", fmt.Sprintf("channel=%s market_id=%d winners=%d", s.channelID, marketID, winnersCount))

	outcomeEmoji := "✅"
	if outcome == "NO" {
		outcomeEmoji = "❌"
	}

	statusText := ""
	if wasDisputed {
		statusText = "\n\\(Reviewed and confirmed by admin\\)"
	}

	message := fmt.Sprintf("💰 *Payouts Distributed*\n\n*#%d* %s\n\n%s Final Outcome: *%s*%s\n💸 %d winners received payouts\n🏆 Total distributed: %s\n\nCongratulations to all winners\\!",
		marketID,
		escapeMarkdown(truncateString(question, 80)),
		outcomeEmoji,
		outcome,
		statusText,
		winnersCount,
		formatBalance(totalPayout))

	recipient := s.getChannelRecipient()
	_, err := s.bot.Send(recipient, message, &telebot.SendOptions{
		ParseMode: telebot.ModeMarkdown,
	})
	if err != nil {
		logger.Debug(0, "broadcast_error", fmt.Sprintf("channel=%s error=%v", s.channelID, err))
		log.Printf("Failed to publish finalization to channel %s: %v", s.channelID, err)
	} else {
		logger.Debug(0, "broadcast_finalization", fmt.Sprintf("market_id=%d winners=%d channel=%s", marketID, winnersCount, s.channelID))
		log.Printf("Successfully published finalization for market #%d to channel %s", marketID, s.channelID)
	}
}

// NotifyDisputeToCreator sends a notification to market creator that their market was disputed
func (s *NotificationService) NotifyDisputeToCreator(market *storage.Market, outcome string) {
	if market == nil {
		return
	}

	user, err := storage.GetUserByID(market.CreatorID)
	if err != nil || user == nil || user.TelegramID == 0 {
		logger.Debug(market.CreatorID, "notification_error", "failed to get creator for dispute notification")
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	message := fmt.Sprintf("⚠️ *Your market has been disputed*\n\nMarket #%d: %s\n\nYour resolution: *%s*\n\nAn admin will review and make the final decision.",
		market.ID,
		truncateString(market.Question, 50),
		outcome)

	_, err = s.bot.Send(&telebot.User{ID: user.TelegramID}, message, &telebot.SendOptions{
		ParseMode: telebot.ModeMarkdown,
	})
	if err != nil {
		logger.Debug(market.CreatorID, "notification_error", fmt.Sprintf("failed to send dispute creator notification: %v", err))
	} else {
		logger.Debug(market.CreatorID, "dispute_creator_notified", fmt.Sprintf("market_id=%d", market.ID))
	}
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
