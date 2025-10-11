package security

import (
	"fmt"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/damiengoehrig/planning-poker/internal/models"
)

// WebSocket message type validation
var validMessageTypes = map[string]bool{
	models.MsgTypeVote:           true,
	models.MsgTypeReveal:         true,
	models.MsgTypeReset:          true,
	models.MsgTypeNextRound:      true,
	models.MsgTypeUpdateName:     true,
	models.MsgTypeUpdateRoomName: true,
	models.MsgTypeUpdateConfig:   true,
}

// IsValidMessageType checks if a WebSocket message type is valid
func IsValidMessageType(msgType string) bool {
	return validMessageTypes[msgType]
}

// RateLimiter provides per-connection rate limiting for WebSocket messages
type RateLimiter struct {
	mu        sync.Mutex
	tokens    map[*websocket.Conn]int
	lastReset time.Time
	maxTokens int
	window    time.Duration
}

// NewRateLimiter creates a new rate limiter
// maxTokens: maximum messages per window
// window: time window for rate limiting (e.g., 1 second)
func NewRateLimiter(maxTokens int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		tokens:    make(map[*websocket.Conn]int),
		lastReset: time.Now(),
		maxTokens: maxTokens,
		window:    window,
	}
}

// Allow checks if a connection is allowed to send a message
// Returns true if allowed, false if rate limit exceeded
func (rl *RateLimiter) Allow(conn *websocket.Conn) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Reset tokens if window has elapsed
	if time.Since(rl.lastReset) > rl.window {
		rl.tokens = make(map[*websocket.Conn]int)
		rl.lastReset = time.Now()
	}

	// Increment token count for this connection
	rl.tokens[conn]++

	// Check if limit exceeded
	return rl.tokens[conn] <= rl.maxTokens
}

// Remove cleans up rate limiter state for a disconnected connection
func (rl *RateLimiter) Remove(conn *websocket.Conn) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	delete(rl.tokens, conn)
}

// OriginValidator validates WebSocket connection origins
type OriginValidator struct {
	allowedPatterns []string
}

// NewOriginValidator creates a new origin validator
func NewOriginValidator(patterns []string) *OriginValidator {
	return &OriginValidator{
		allowedPatterns: patterns,
	}
}

// GetAcceptOptions returns websocket.AcceptOptions with origin patterns
func (ov *OriginValidator) GetAcceptOptions() *websocket.AcceptOptions {
	return &websocket.AcceptOptions{
		OriginPatterns: ov.allowedPatterns,
	}
}

// ValidateMessagePayload validates WebSocket message payload structure
func ValidateMessagePayload(msgType string, payload interface{}) error {
	payloadMap, ok := payload.(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid payload format")
	}

	switch msgType {
	case models.MsgTypeVote:
		// Vote must have value field
		if _, ok := payloadMap["value"].(string); !ok {
			return fmt.Errorf("vote payload must have string 'value' field")
		}

	case models.MsgTypeUpdateName, models.MsgTypeUpdateRoomName:
		// Name updates must have name field
		if _, ok := payloadMap["name"].(string); !ok {
			return fmt.Errorf("name update payload must have string 'name' field")
		}

	case models.MsgTypeUpdateConfig:
		// Config update must have config field
		if _, ok := payloadMap["config"]; !ok {
			return fmt.Errorf("config update payload must have 'config' field")
		}

	case models.MsgTypeReveal, models.MsgTypeReset, models.MsgTypeNextRound:
		// These message types don't require specific payload validation
		// Empty payload is acceptable
	}

	return nil
}
