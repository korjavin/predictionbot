package logger

import (
	"log"
	"time"
)

// Debug logs a debug message with consistent format
// Format: [DEBUG] timestamp=... user_id=... action=... details=...
func Debug(userID int64, action, details string) {
	timestamp := time.Now().Format(time.RFC3339)
	log.Printf("[DEBUG] timestamp=%s user_id=%d action=%s details=%s", timestamp, userID, action, details)
}
