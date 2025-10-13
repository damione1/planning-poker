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
	PongTimeout               = 90 * time.Second // 3x ping interval for network delay tolerance

	// Channel buffers
	ClientSendBufferSize      = 256
	HubBroadcastBufferSize    = 256
	HubRegisterBufferSize     = 100
	HubUnregisterBufferSize   = 100
)
