package security

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"
)

// Input length constraints
const (
	MaxRoomNameLength        = 100
	MaxParticipantNameLength = 50
	MinNameLength            = 1
)

var (
	// PocketBase ID regex - 15 character alphanumeric
	pocketbaseIDRegex = regexp.MustCompile(`^[a-zA-Z0-9]{15}$`)
	// UUID validation regex (for potential future use)
	uuidRegex = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	// Name validation regex - Unicode letters, digits, spaces, apostrophes, hyphens, underscores, dots
	// \p{L} matches any Unicode letter (includes accented characters)
	// \p{N} matches any Unicode number
	// ' allows apostrophes (for French and English possessives)
	nameRegex = regexp.MustCompile(`^[\p{L}\p{N}\s'\-_.]+$`)
	// Dangerous characters that could be used for injection attacks
	dangerousCharsRegex = regexp.MustCompile(`[<>{}[\]\\;|&$()` + "`" + `]`)
)

// ValidateUUID validates that a string is a valid PocketBase ID or UUID format
// PocketBase uses 15-character alphanumeric IDs, not standard UUIDs
// Returns error if the string is not a valid ID format
func ValidateUUID(id string) error {
	if id == "" {
		return fmt.Errorf("ID cannot be empty")
	}

	// Check for PocketBase ID format (15 alphanumeric characters)
	if pocketbaseIDRegex.MatchString(id) {
		return nil
	}

	// Fallback: Check for standard UUID format (for compatibility)
	if uuidRegex.MatchString(strings.ToLower(id)) {
		if _, err := uuid.Parse(id); err != nil {
			return fmt.Errorf("malformed UUID: %w", err)
		}
		return nil
	}

	return fmt.Errorf("invalid ID format (expected 15-character PocketBase ID or UUID)")
}

// ValidateName validates a name string with length and character constraints
// Returns sanitized name and error if validation fails
func ValidateName(name string, maxLen int) (string, error) {
	// Trim leading/trailing whitespace
	name = strings.TrimSpace(name)

	// Check empty
	if name == "" {
		return "", fmt.Errorf("name cannot be empty")
	}

	// Check minimum length
	if len(name) < MinNameLength {
		return "", fmt.Errorf("name too short (min %d characters)", MinNameLength)
	}

	// Check maximum length
	if len(name) > maxLen {
		return "", fmt.Errorf("name too long (max %d characters)", maxLen)
	}

	// Check for invalid characters (must match allowed character set)
	if !nameRegex.MatchString(name) {
		return "", fmt.Errorf("name contains invalid characters (allowed: letters, numbers, spaces, apostrophes, hyphens, underscores, dots)")
	}

	// Check for dangerous characters that could be used for injection
	if dangerousCharsRegex.MatchString(name) {
		return "", fmt.Errorf("name contains potentially dangerous characters")
	}

	// Check for control characters (belt-and-suspenders with regex)
	for _, r := range name {
		if r < 32 || r == 127 {
			return "", fmt.Errorf("name contains control characters")
		}
	}

	return name, nil
}

// ValidateRoomName validates a room name
func ValidateRoomName(name string) (string, error) {
	return ValidateName(name, MaxRoomNameLength)
}

// ValidateParticipantName validates a participant name
func ValidateParticipantName(name string) (string, error) {
	return ValidateName(name, MaxParticipantNameLength)
}

// SanitizeErrorMessage removes sensitive information from error messages
// Returns a generic user-friendly error message
func SanitizeErrorMessage(err error) string {
	if err == nil {
		return ""
	}

	errStr := strings.ToLower(err.Error())

	// Common database/internal error patterns to sanitize
	sensitivePatterns := []string{
		"sql",
		"database",
		"record",
		"collection",
		"pocketbase",
		"constraint",
		"foreign key",
		"unique",
		"duplicate key",
		"no rows",
	}

	for _, pattern := range sensitivePatterns {
		if strings.Contains(errStr, pattern) {
			return "An error occurred while processing your request"
		}
	}

	// If no sensitive patterns detected, return original
	return err.Error()
}
