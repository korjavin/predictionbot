package auth

import (
	"context"
	"testing"
)

func TestGetUserIDFromContext(t *testing.T) {
	tests := []struct {
		name     string
		userID   int64
		expectOk bool
	}{
		{
			name:     "valid user ID",
			userID:   12345,
			expectOk: true,
		},
		{
			name:     "zero user ID",
			userID:   0,
			expectOk: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.WithValue(context.Background(), UserIDKey, tt.userID)
			userID, ok := GetUserIDFromContext(ctx)
			if ok != tt.expectOk {
				t.Errorf("Expected ok=%v, got ok=%v", tt.expectOk, ok)
			}
			if ok && userID != tt.userID {
				t.Errorf("Expected userID=%d, got userID=%d", tt.userID, userID)
			}
		})
	}
}

func TestGetUserIDFromContextMissing(t *testing.T) {
	ctx := context.Background()
	_, ok := GetUserIDFromContext(ctx)
	if ok {
		t.Error("Expected ok=false for missing user ID in context")
	}
}

func TestUserIDKey(t *testing.T) {
	// Verify the key type and value
	key := UserIDKey
	if key == "" {
		t.Error("UserIDKey should not be empty string")
	}
}
