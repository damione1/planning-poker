package helpers

import (
	"context"
	"net"
	"sync"
	"time"

	"github.com/coder/websocket"
)

// MockWSConn implements a mock websocket.Conn for testing
type MockWSConn struct {
	mu          sync.RWMutex
	messages    [][]byte
	closed      bool
	closeStatus websocket.StatusCode
	closeReason string
	writeErr    error
	readErr     error
	localAddr   net.Addr
	remoteAddr  net.Addr
	subprotocol string
}

// NewMockWSConn creates a new mock WebSocket connection
func NewMockWSConn() *MockWSConn {
	return &MockWSConn{
		messages:    make([][]byte, 0),
		localAddr:   &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 8080},
		remoteAddr:  &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 9090},
		subprotocol: "",
	}
}

// Write records a message being sent
func (m *MockWSConn) Write(ctx context.Context, typ websocket.MessageType, data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return net.ErrClosed
	}

	if m.writeErr != nil {
		return m.writeErr
	}

	// Store a copy of the data
	dataCopy := make([]byte, len(data))
	copy(dataCopy, data)
	m.messages = append(m.messages, dataCopy)

	return nil
}

// Read simulates reading a message
func (m *MockWSConn) Read(ctx context.Context) (websocket.MessageType, []byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.closed {
		return 0, nil, net.ErrClosed
	}

	if m.readErr != nil {
		return 0, nil, m.readErr
	}

	// For testing, we don't actually read messages
	// Tests should use ReceivedMessages() to verify sent messages
	select {
	case <-ctx.Done():
		return 0, nil, ctx.Err()
	case <-time.After(100 * time.Millisecond):
		return 0, nil, context.DeadlineExceeded
	}
}

// Close marks the connection as closed
func (m *MockWSConn) Close(status websocket.StatusCode, reason string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.closed = true
	m.closeStatus = status
	m.closeReason = reason
	return nil
}

// CloseRead is a no-op for the mock
func (m *MockWSConn) CloseRead(ctx context.Context) context.Context {
	return ctx
}

// SetReadLimit is a no-op for the mock
func (m *MockWSConn) SetReadLimit(limit int64) {
	// No-op
}

// Ping sends a ping message
func (m *MockWSConn) Ping(ctx context.Context) error {
	return m.Write(ctx, websocket.MessageText, []byte("ping"))
}

// Subprotocol returns the negotiated subprotocol
func (m *MockWSConn) Subprotocol() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.subprotocol
}

// ReceivedMessages returns all messages sent through this connection
func (m *MockWSConn) ReceivedMessages() [][]byte {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Return a copy to avoid race conditions
	result := make([][]byte, len(m.messages))
	for i, msg := range m.messages {
		msgCopy := make([]byte, len(msg))
		copy(msgCopy, msg)
		result[i] = msgCopy
	}
	return result
}

// IsClosed returns whether the connection is closed
func (m *MockWSConn) IsClosed() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.closed
}

// CloseStatus returns the close status code
func (m *MockWSConn) CloseStatus() websocket.StatusCode {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.closeStatus
}

// CloseReason returns the close reason
func (m *MockWSConn) CloseReason() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.closeReason
}

// SetWriteErr sets an error to be returned on Write calls
func (m *MockWSConn) SetWriteErr(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.writeErr = err
}

// SetReadErr sets an error to be returned on Read calls
func (m *MockWSConn) SetReadErr(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.readErr = err
}

// ClearMessages clears all recorded messages
func (m *MockWSConn) ClearMessages() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages = make([][]byte, 0)
}
