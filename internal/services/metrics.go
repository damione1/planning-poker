package services

import (
	"runtime"
	"sync/atomic"
	"time"
)

// Metrics tracks WebSocket server performance and resource usage
type Metrics struct {
	// Connection metrics
	activeConnections   int64
	totalConnections    int64
	activeRooms         int64

	// Message metrics
	messagesReceived    int64
	messagesSent        int64
	lastMessageTime     int64 // Unix timestamp

	// Error metrics
	connectionErrors    int64
	broadcastErrors     int64
	rateLimitViolations int64

	// Resource metrics
	startTime           time.Time
}

// NewMetrics creates a new metrics tracker
func NewMetrics() *Metrics {
	return &Metrics{
		startTime: time.Now(),
	}
}

// Connection tracking
func (m *Metrics) IncrementConnections() {
	atomic.AddInt64(&m.activeConnections, 1)
	atomic.AddInt64(&m.totalConnections, 1)
}

func (m *Metrics) DecrementConnections() {
	atomic.AddInt64(&m.activeConnections, -1)
}

func (m *Metrics) IncrementRooms() {
	atomic.AddInt64(&m.activeRooms, 1)
}

func (m *Metrics) DecrementRooms() {
	atomic.AddInt64(&m.activeRooms, -1)
}

// Message tracking
func (m *Metrics) IncrementMessagesReceived() {
	atomic.AddInt64(&m.messagesReceived, 1)
	atomic.StoreInt64(&m.lastMessageTime, time.Now().Unix())
}

func (m *Metrics) IncrementMessagesSent() {
	atomic.AddInt64(&m.messagesSent, 1)
}

// Error tracking
func (m *Metrics) IncrementConnectionErrors() {
	atomic.AddInt64(&m.connectionErrors, 1)
}

func (m *Metrics) IncrementBroadcastErrors() {
	atomic.AddInt64(&m.broadcastErrors, 1)
}

func (m *Metrics) IncrementRateLimitViolations() {
	atomic.AddInt64(&m.rateLimitViolations, 1)
}

// MetricsSnapshot represents a point-in-time view of metrics
type MetricsSnapshot struct {
	// Connection metrics
	ActiveConnections   int64   `json:"active_connections"`
	TotalConnections    int64   `json:"total_connections"`
	ActiveRooms         int64   `json:"active_rooms"`

	// Message metrics
	MessagesReceived    int64   `json:"messages_received"`
	MessagesSent        int64   `json:"messages_sent"`
	MessagesPerSecond   float64 `json:"messages_per_second"`
	LastMessageTime     string  `json:"last_message_time"`

	// Error metrics
	ConnectionErrors    int64   `json:"connection_errors"`
	BroadcastErrors     int64   `json:"broadcast_errors"`
	RateLimitViolations int64   `json:"rate_limit_violations"`

	// Resource metrics
	UptimeSeconds       int64   `json:"uptime_seconds"`
	MemoryUsageMB       uint64  `json:"memory_usage_mb"`
	NumGoroutines       int     `json:"num_goroutines"`

	// Health indicators
	CPUUsagePercent     float64 `json:"cpu_usage_percent,omitempty"`
	HealthStatus        string  `json:"health_status"`
}

// Snapshot returns a point-in-time view of all metrics
func (m *Metrics) Snapshot() MetricsSnapshot {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	uptime := time.Since(m.startTime)
	messagesPerSec := float64(atomic.LoadInt64(&m.messagesReceived)) / uptime.Seconds()

	lastMsgTime := atomic.LoadInt64(&m.lastMessageTime)
	lastMsgTimeStr := "never"
	if lastMsgTime > 0 {
		lastMsgTimeStr = time.Unix(lastMsgTime, 0).Format(time.RFC3339)
	}

	snapshot := MetricsSnapshot{
		ActiveConnections:   atomic.LoadInt64(&m.activeConnections),
		TotalConnections:    atomic.LoadInt64(&m.totalConnections),
		ActiveRooms:         atomic.LoadInt64(&m.activeRooms),
		MessagesReceived:    atomic.LoadInt64(&m.messagesReceived),
		MessagesSent:        atomic.LoadInt64(&m.messagesSent),
		MessagesPerSecond:   messagesPerSec,
		LastMessageTime:     lastMsgTimeStr,
		ConnectionErrors:    atomic.LoadInt64(&m.connectionErrors),
		BroadcastErrors:     atomic.LoadInt64(&m.broadcastErrors),
		RateLimitViolations: atomic.LoadInt64(&m.rateLimitViolations),
		UptimeSeconds:       int64(uptime.Seconds()),
		MemoryUsageMB:       memStats.Alloc / 1024 / 1024,
		NumGoroutines:       runtime.NumGoroutine(),
		HealthStatus:        m.calculateHealthStatus(),
	}

	return snapshot
}

// calculateHealthStatus determines overall system health
func (m *Metrics) calculateHealthStatus() string {
	activeConns := atomic.LoadInt64(&m.activeConnections)
	activeRooms := atomic.LoadInt64(&m.activeRooms)
	errors := atomic.LoadInt64(&m.connectionErrors) + atomic.LoadInt64(&m.broadcastErrors)

	// Critical: Over 90% capacity or high error rate
	if activeConns > 9000 || activeRooms > 900 {
		return "critical"
	}

	// Warning: Over 80% capacity or some errors
	if activeConns > 8000 || activeRooms > 800 || errors > 100 {
		return "warning"
	}

	// Healthy
	return "healthy"
}
