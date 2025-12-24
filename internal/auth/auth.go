package auth

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"predictionbot/internal/logger"
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

	// Parse the initData string using url.ParseQuery
	// This automatically URL-decodes the values
	parsedData, err := url.ParseQuery(initData)
	if err != nil {
		return 0, fmt.Errorf("failed to parse initData: %w", err)
	}

	// Extract hash and other data
	var hash string
	data := make(map[string]string)

	for key, values := range parsedData {
		if len(values) == 0 {
			continue
		}
		value := values[0] // Use first value

		if key == "hash" {
			hash = value
		} else {
			// Include ALL fields except hash in the data check string
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

	// Trim any whitespace from bot token (common issue)
	botToken = strings.TrimSpace(botToken)


	// Create the data check string (sorted by key)
	// IMPORTANT: The keys must be sorted alphabetically!
	var dataCheckKeys []string
	for key := range data {
		dataCheckKeys = append(dataCheckKeys, key)
	}
	// Sort the keys
	// Using simple bubble sort to avoid importing "sort" package
	for i := 0; i < len(dataCheckKeys); i++ {
		for j := i + 1; j < len(dataCheckKeys); j++ {
			if dataCheckKeys[i] > dataCheckKeys[j] {
				dataCheckKeys[i], dataCheckKeys[j] = dataCheckKeys[j], dataCheckKeys[i]
			}
		}
	}

	var dataCheck []string
	for _, key := range dataCheckKeys {
		dataCheck = append(dataCheck, fmt.Sprintf("%s=%s", key, data[key]))
	}
	dataCheckString := strings.Join(dataCheck, "\n")


	// Compute the secret key: HMAC_SHA256(key="WebAppData", message=bot_token)
	// The constant string "WebAppData" is used as the key
	secretKey := hmac.New(sha256.New, []byte("WebAppData"))
	secretKey.Write([]byte(botToken))
	secret := secretKey.Sum(nil)

	// Compute the expected hash: HMAC_SHA256(<secret>, <data_check_string>)
	h := hmac.New(sha256.New, secret)
	h.Write([]byte(dataCheckString))
	computedHash := hex.EncodeToString(h.Sum(nil))

	// Compare hashes
	if hash != computedHash {
		logger.Debug(0, "auth_invalid_hash", "hash_mismatch")
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
		logger.Debug(0, "auth_expired", fmt.Sprintf("auth_date=%d now=%d", authDate, now))
		return 0, fmt.Errorf("auth_date is too old")
	}

	// Extract user ID
	userStr, ok := data["user"]
	if !ok {
		return 0, fmt.Errorf("user not found in initData")
	}

	// Parse user JSON to extract id (already URL-decoded by ParseQuery)
	// Simple parsing: look for "id":number pattern
	userID, err := extractUserID(userStr)
	if err != nil {
		return 0, fmt.Errorf("failed to parse user: %w", err)
	}

	logger.Debug(userID, "auth_validated", fmt.Sprintf("auth_date=%d", authDate))
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
		logger.Debug(telegramID, "user_found", fmt.Sprintf("user_id=%d", user.ID))
		return user, nil
	}

	// Create new user with welcome bonus
	user, err = storage.CreateUser(telegramID, username, firstName)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	logger.Debug(telegramID, "user_created", fmt.Sprintf("user_id=%d welcome_bonus=1000", user.ID))
	log.Printf("Created new user %d with welcome bonus", telegramID)
	return user, nil
}

// writeJSONError writes a JSON error response
func writeJSONError(w http.ResponseWriter, statusCode int, errorMessage string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	fmt.Fprintf(w, `{"error": "%s"}`, errorMessage)
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
			logger.Debug(0, "auth_missing_header", fmt.Sprintf("path=%s", r.URL.Path))
			log.Printf("[AUTH] Missing X-Telegram-Init-Data header for %s", r.URL.Path)
			writeJSONError(w, http.StatusUnauthorized, "Missing authentication data")
			return
		}

		// Parse initData to get user info
		parsedData, err := url.ParseQuery(initData)
		if err != nil {
			logger.Debug(0, "auth_parse_failed", fmt.Sprintf("path=%s error=%v", r.URL.Path, err))
			log.Printf("[AUTH] Failed to parse initData for %s: %v", r.URL.Path, err)
			writeJSONError(w, http.StatusUnauthorized, "Invalid initData format")
			return
		}

		userValues := parsedData["user"]
		if len(userValues) == 0 {
			logger.Debug(0, "auth_missing_user", fmt.Sprintf("path=%s", r.URL.Path))
			log.Printf("[AUTH] User data not found in initData for %s", r.URL.Path)
			writeJSONError(w, http.StatusUnauthorized, "User data not found")
			return
		}

		userStr := userValues[0] // ParseQuery already URL-decoded it

		// Extract user info
		username, firstName, err := extractUserInfo(userStr)
		if err != nil {
			logger.Debug(0, "auth_extract_failed", fmt.Sprintf("path=%s error=%v", r.URL.Path, err))
			log.Printf("[AUTH] Failed to extract user info for %s: %v", r.URL.Path, err)
			writeJSONError(w, http.StatusUnauthorized, "Invalid user data format")
			return
		}

		userID, err := ValidateInitData(initData)
		if err != nil {
			logger.Debug(0, "auth_validation_failed", fmt.Sprintf("path=%s error=%v", r.URL.Path, err))
			log.Printf("[AUTH] Validation failed for %s: %v", r.URL.Path, err)
			writeJSONError(w, http.StatusUnauthorized, "Authentication failed: "+err.Error())
			return
		}

		logger.Debug(userID, "auth_middleware_success", fmt.Sprintf("path=%s", r.URL.Path))
		log.Printf("[AUTH] Success: user_id=%d path=%s", userID, r.URL.Path)

		// Get or create user (auto-registration with welcome bonus)
		_, err = GetOrCreateUser(userID, username, firstName)
		if err != nil {
			logger.Debug(userID, "auth_user_failed", fmt.Sprintf("error=%v", err))
			log.Printf("[AUTH] Failed to get/create user %d: %v", userID, err)
			writeJSONError(w, http.StatusInternalServerError, "Failed to load user profile")
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
