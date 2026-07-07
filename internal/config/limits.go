package config

import "time"

// WebSocket connection limits and constraints
const (
	// Connection limits
	MaxConnectionsPerRoom     = 50
	MaxRoomsPerInstance       = 1000
	MaxTotalConnections       = 10000

	// Rate limiting
	MaxMessagesPerSecond      = 10
	RateLimitWindow           = time.Second

	// Timeouts
	ConnectionTimeout         = 5 * time.Minute
	WriteTimeout              = 10 * time.Second
	ReadTimeout               = 60 * time.Second
	PingInterval              = 30 * time.Second

	// Message size limits
	MaxMessageBytes           = 32 * 1024 // 32 KiB, comfortably above our ~10 KiB config JSON max

	// Channel buffers
	ClientSendBufferSize      = 256
	HubBroadcastBufferSize    = 256
	HubRegisterBufferSize     = 100
	HubUnregisterBufferSize   = 100

	// Auto-reveal: delay between "all voters voted" and the server-triggered
	// reveal. Also sent to clients as the cosmetic countdown duration.
	AutoRevealDelay           = 1500 * time.Millisecond
)
