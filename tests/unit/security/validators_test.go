package security_test

import (
	"strings"
	"testing"

	"github.com/damione1/planning-poker/internal/security"
	"github.com/stretchr/testify/assert"
)

func TestValidateUUID(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		// PocketBase ID format (15 alphanumeric characters)
		{"valid pocketbase id", "abc123def456ghi", false},
		{"valid pocketbase id uppercase", "ABC123DEF456GHI", false},
		{"valid pocketbase id mixed", "aBc123DeF456GhI", false},

		// UUID format (for compatibility)
		{"valid uuid v4", "550e8400-e29b-41d4-a716-446655440000", false},
		{"valid uuid lowercase", "550e8400-e29b-41d4-a716-446655440000", false},

		// Invalid cases
		{"empty", "", true},
		{"too short", "abc", true},
		{"too long pocketbase", "abc123def456ghijkl", true},
		{"pocketbase with dash", "abc-123-def-456", true},
		{"invalid uuid", "not-a-uuid", true},
		{"sql injection", "' OR '1'='1", true},
		{"xss attempt", "<script>alert('xss')</script>", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := security.ValidateUUID(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateRoomName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		// Valid cases
		{"valid simple name", "Sprint Planning", "Sprint Planning", false},
		{"valid with numbers", "Sprint 2024 Q1", "Sprint 2024 Q1", false},
		{"valid with hyphen", "Sprint-Planning", "Sprint-Planning", false},
		{"valid with underscore", "Sprint_Planning", "Sprint_Planning", false},
		{"valid with dot", "Sprint.Planning", "Sprint.Planning", false},
		{"valid with leading space", "  Sprint Planning", "Sprint Planning", false},
		{"valid with trailing space", "Sprint Planning  ", "Sprint Planning", false},
		{"minimum length", "S", "S", false},
		{"maximum length", strings.Repeat("a", 100), strings.Repeat("a", 100), false},

		// Invalid cases
		{"empty", "", "", true},
		{"whitespace only", "   ", "", true},
		{"too long", strings.Repeat("a", 101), "", true},
		{"xss attempt", "<script>alert('xss')</script>", "", true},
		{"sql injection", "'; DROP TABLE rooms--", "", true},
		{"special chars", "Sprint @ Planning", "", true},
		{"control characters", "Sprint\nPlanning", "", true},
		{"unicode emoji", "Sprint 🚀", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := security.ValidateRoomName(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestValidateParticipantName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		// Valid cases
		{"valid name", "Alice", "Alice", false},
		{"valid with space", "Alice Smith", "Alice Smith", false},
		{"valid with numbers", "Player123", "Player123", false},
		{"valid with hyphen", "Alice-Bob", "Alice-Bob", false},
		{"valid with underscore", "Alice_Bob", "Alice_Bob", false},
		{"minimum length", "A", "A", false},
		{"maximum length", strings.Repeat("a", 50), strings.Repeat("a", 50), false},
		{"trim whitespace", "  Alice  ", "Alice", false},

		// Invalid cases
		{"empty", "", "", true},
		{"whitespace only", "   ", "", true},
		{"too long", strings.Repeat("a", 51), "", true},
		{"xss attempt", "<script>alert('xss')</script>", "", true},
		{"img onerror", "<img src=x onerror=alert('xss')>", "", true},
		{"event handler", "<div onload=alert('xss')>Alice</div>", "", true},
		{"sql injection", "'; DROP TABLE--", "", true},
		{"special chars", "Alice@Bob", "", true},
		{"control chars", "Alice\x00Bob", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := security.ValidateParticipantName(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestSanitizeErrorMessage(t *testing.T) {
	tests := []struct {
		name     string
		input    error
		contains string
	}{
		{
			"nil error",
			nil,
			"",
		},
		{
			"sql error",
			assert.AnError, // Use a real error
			"",
		},
		{
			"generic error",
			assert.AnError,
			"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := security.SanitizeErrorMessage(tt.input)
			if tt.input == nil {
				assert.Empty(t, got)
			} else {
				assert.NotEmpty(t, got)
			}
		})
	}
}

func TestValidateName_EdgeCases(t *testing.T) {
	t.Run("null byte injection", func(t *testing.T) {
		_, err := security.ValidateParticipantName("Alice\x00Bob")
		assert.Error(t, err)
		// Error could be either "control characters" or "invalid characters"
		assert.True(t, err != nil)
	})

	t.Run("delete character", func(t *testing.T) {
		_, err := security.ValidateParticipantName("Alice\x7FBob")
		assert.Error(t, err)
		// Error could be either "control characters" or "invalid characters"
		assert.True(t, err != nil)
	})

	t.Run("newline injection", func(t *testing.T) {
		_, err := security.ValidateRoomName("Sprint\nPlanning")
		assert.Error(t, err)
	})

	t.Run("carriage return injection", func(t *testing.T) {
		_, err := security.ValidateRoomName("Sprint\rPlanning")
		assert.Error(t, err)
	})
}
