package config

import "time"

// WebSocket connection limits and constraints
const (
	// Connection limits
	MaxConnectionsPerRoom = 50
	MaxRoomsPerInstance   = 1000
	MaxTotalConnections   = 10000

	// Rate limiting
	MaxMessagesPerSecond = 10
	RateLimitWindow      = time.Second

	// Timeouts
	WriteTimeout = 10 * time.Second
	PingInterval = 30 * time.Second

	// Message size limits
	MaxMessageBytes = 32 * 1024 // 32 KiB, comfortably above our ~10 KiB config JSON max

	// Per-IP HTTP rate limits for POST /room and POST /room/{id}/join.
	// These bound the room-creation and join floods a single client can
	// generate; see internal/security.IPRateLimiter for enforcement.
	RoomCreatesPerIPPerMinute = 10
	RoomJoinsPerIPPerMinute   = 30

	// MaxRoomsInDB hard-caps the number of room rows the database can hold.
	// Rooms live for 24h (see RoomManager.CreateRoom's expires_at), so this
	// needs headroom above MaxRoomsPerInstance (which only counts rooms with
	// an active in-memory hub entry) to avoid rejecting room creation while
	// the hub itself is nowhere near capacity.
	MaxRoomsInDB = 2000

	// MaxParticipantsPerRoom caps DB participant rows per room, matching
	// MaxConnectionsPerRoom (the equivalent limit for live WebSocket
	// connections).
	MaxParticipantsPerRoom = 50

	// Channel buffers
	ClientSendBufferSize    = 256
	HubBroadcastBufferSize  = 256
	HubRegisterBufferSize   = 100
	HubUnregisterBufferSize = 100

	// Auto-reveal: delay between "all voters voted" and the server-triggered
	// reveal. Also sent to clients as the cosmetic countdown duration.
	AutoRevealDelay = 1500 * time.Millisecond
)
