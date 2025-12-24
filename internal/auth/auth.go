package auth

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"predictionbot/internal/storage"
)

// ContextKey is the key type for context values
type ContextKey string

const (
	// UserIDKey is the context key for user ID
	UserIDKey ContextKey = "user_id"
)

// ValidateInitData validates the Telegram initData string
// It checks the HMAC-SHA256 signature and the auth_date
func ValidateInitData(initData string) (int64, error) {
	// Parse the initData string
	parts := strings.Split(initData, "&")
	if len(parts) == 0 {
		return 0, fmt.Errorf("empty initData")
	}

	// Extract hash and other data
	var hash string
	data := make(map[string]string)

	for _, part := range parts {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			continue
		}
		key, value := kv[0], kv[1]
		if key == "hash" {
			hash = value
		} else {
			data[key] = value
		}
	}

	if hash == "" {
		return 0, fmt.Errorf("hash not found in initData")
	}

	// Get the bot token
	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	if botToken == "" {
		return 0, fmt.Errorf("TELEGRAM_BOT_TOKEN not set")
	}

	// Create the data check string (sorted by key)
	var dataCheck []string
	for key, value := range data {
		dataCheck = append(dataCheck, fmt.Sprintf("%s=%s", key, value))
	}
	dataCheckString := strings.Join(dataCheck, "\n")

	// Compute the expected hash
	h := hmac.New(sha256.New, []byte(botToken))
	h.Write([]byte(dataCheckString))
	computedHash := hex.EncodeToString(h.Sum(nil))

	// Compare hashes
	if hash != computedHash {
		return 0, fmt.Errorf("invalid hash")
	}

	// Check auth_date (must be less than 24 hours old)
	authDateStr, ok := data["auth_date"]
	if !ok {
		return 0, fmt.Errorf("auth_date not found")
	}

	var authDate int64
	if _, err := fmt.Sscanf(authDateStr, "%d", &authDate); err != nil {
		return 0, fmt.Errorf("invalid auth_date format")
	}

	now := time.Now().Unix()
	maxAge := int64(24 * 60 * 60) // 24 hours in seconds

	if now-authDate > maxAge {
		return 0, fmt.Errorf("auth_date is too old")
	}

	// Extract user ID
	userStr, ok := data["user"]
	if !ok {
		return 0, fmt.Errorf("user not found in initData")
	}

	// Parse user JSON to extract id
	// Simple parsing: look for "id":number pattern
	userID, err := extractUserID(userStr)
	if err != nil {
		return 0, fmt.Errorf("failed to parse user: %w", err)
	}

	return userID, nil
}

// extractUserID extracts the user ID from the user JSON string
func extractUserID(userJSON string) (int64, error) {
	// Look for "id": followed by digits
	prefix := `"id":`
	idx := strings.Index(userJSON, prefix)
	if idx == -1 {
		return 0, fmt.Errorf("id field not found")
	}

	// Find the number after "id":
	start := idx + len(prefix)
	var numStr string
	for i := start; i < len(userJSON); i++ {
		if userJSON[i] >= '0' && userJSON[i] <= '9' {
			numStr += string(userJSON[i])
		} else if len(numStr) > 0 {
			break
		}
	}

	if len(numStr) == 0 {
		return 0, fmt.Errorf("user id not found")
	}

	var userID int64
	if _, err := fmt.Sscanf(numStr, "%d", &userID); err != nil {
		return 0, err
	}

	return userID, nil
}

// extractUserInfo extracts username and first_name from the user JSON string
func extractUserInfo(userJSON string) (username, firstName string, err error) {
	// Extract first_name
	firstNamePrefix := `"first_name":"`
	idx := strings.Index(userJSON, firstNamePrefix)
	if idx != -1 {
		start := idx + len(firstNamePrefix)
		var end int
		for i := start; i < len(userJSON); i++ {
			if userJSON[i] == '"' {
				end = i
				break
			}
		}
		if end > start {
			firstName = userJSON[start:end]
		}
	}

	// Extract username (optional)
	usernamePrefix := `"username":"`
	idx = strings.Index(userJSON, usernamePrefix)
	if idx != -1 {
		start := idx + len(usernamePrefix)
		var end int
		for i := start; i < len(userJSON); i++ {
			if userJSON[i] == '"' {
				end = i
				break
			}
		}
		if end > start {
			username = userJSON[start:end]
		}
	}

	if firstName == "" {
		return "", "", fmt.Errorf("first_name not found in user JSON")
	}

	return username, firstName, nil
}

// GetOrCreateUser retrieves an existing user or creates a new one with welcome bonus
func GetOrCreateUser(telegramID int64, username, firstName string) (*storage.User, error) {
	// Try to get existing user
	user, err := storage.GetUserByTelegramID(telegramID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	if user != nil {
		return user, nil
	}

	// Create new user with welcome bonus
	user, err = storage.CreateUser(telegramID, username, firstName)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	log.Printf("Created new user %d with welcome bonus", telegramID)
	return user, nil
}

// Middleware returns an HTTP middleware that validates Telegram initData
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip auth for non-API routes (static files)
		if !strings.HasPrefix(r.URL.Path, "/api/") {
			next.ServeHTTP(w, r)
			return
		}

		// Skip auth for health check endpoints if needed
		if r.URL.Path == "/api/ping" {
			next.ServeHTTP(w, r)
			return
		}

		initData := r.Header.Get("X-Telegram-Init-Data")
		if initData == "" {
			http.Error(w, "Unauthorized: missing X-Telegram-Init-Data header", http.StatusUnauthorized)
			return
		}

		// Parse initData to get user info
		parts := strings.Split(initData, "&")
		var userStr string
		for _, part := range parts {
			kv := strings.SplitN(part, "=", 2)
			if len(kv) == 2 && kv[0] == "user" {
				userStr = kv[1]
				break
			}
		}

		if userStr == "" {
			http.Error(w, "Unauthorized: user data not found", http.StatusUnauthorized)
			return
		}

		// Extract user info
		username, firstName, err := extractUserInfo(userStr)
		if err != nil {
			log.Printf("Failed to extract user info: %v", err)
			http.Error(w, "Unauthorized: invalid user data", http.StatusUnauthorized)
			return
		}

		userID, err := ValidateInitData(initData)
		if err != nil {
			log.Printf("Auth failed: %v", err)
			http.Error(w, "Unauthorized: invalid initData", http.StatusUnauthorized)
			return
		}

		// Get or create user (auto-registration with welcome bonus)
		_, err = GetOrCreateUser(userID, username, firstName)
		if err != nil {
			log.Printf("Failed to get/create user: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		// Add user ID to context
		ctx := r.Context()
		ctx = contextWithUserID(ctx, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// contextWithUserID adds the user ID to the context
func contextWithUserID(ctx context.Context, userID int64) context.Context {
	return context.WithValue(ctx, UserIDKey, userID)
}

// GetUserIDFromContext retrieves the user ID from the context
func GetUserIDFromContext(ctx context.Context) (int64, bool) {
	userID, ok := ctx.Value(UserIDKey).(int64)
	return userID, ok
}
