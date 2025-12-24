package service

import (
	"testing"
)

func TestTruncateString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{
			name:     "string shorter than max",
			input:    "Hello",
			maxLen:   10,
			expected: "Hello",
		},
		{
			name:     "string equal to max",
			input:    "Hello",
			maxLen:   5,
			expected: "Hello",
		},
		{
			name:     "string longer than max",
			input:    "Hello World",
			maxLen:   8,
			expected: "Hello...",
		},
		{
			name:     "empty string",
			input:    "",
			maxLen:   10,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncateString(tt.input, tt.maxLen)
			if result != tt.expected {
				t.Errorf("truncateString(%q, %d) = %q, want %q", tt.input, tt.maxLen, result, tt.expected)
			}
		})
	}
}

func TestParseChannelID(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int64
	}{
		{
			name:     "username format",
			input:    "@predictionbot",
			expected: 0,
		},
		{
			name:     "supergroup format",
			input:    "-1001234567890",
			expected: -1001234567890,
		},
		{
			name:     "plain negative number",
			input:    "-123456789",
			expected: -123456789,
		},
		{
			name:     "plain positive number",
			input:    "123456789",
			expected: 123456789,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseChannelID(tt.input)
			if result != tt.expected {
				t.Errorf("parseChannelID(%q) = %d, want %d", tt.input, result, tt.expected)
			}
		})
	}
}

func TestFormatBalance(t *testing.T) {
	tests := []struct {
		name     string
		cents    int64
		expected string
	}{
		{
			name:     "zero",
			cents:    0,
			expected: "0.00 WSC",
		},
		{
			name:     "one dollar",
			cents:    100,
			expected: "1.00 WSC",
		},
		{
			name:     "whole dollars",
			cents:    50000,
			expected: "500.00 WSC",
		},
		{
			name:     "decimal cents",
			cents:    12345,
			expected: "123.45 WSC",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatBalance(tt.cents)
			if result != tt.expected {
				t.Errorf("formatBalance(%d) = %q, want %q", tt.cents, result, tt.expected)
			}
		})
	}
}
