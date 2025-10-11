package helpers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/damione1/planning-poker/internal/models"
)

// WSClient is a test WebSocket client
type WSClient struct {
	conn          *websocket.Conn
	messages      []models.WSMessage
	messagesMu    sync.RWMutex
	participantID string
	closed        bool
	closedMu      sync.RWMutex
}

// NewWSClient creates a new WebSocket test client
func NewWSClient() *WSClient {
	return &WSClient{
		messages: make([]models.WSMessage, 0),
	}
}

// Connect establishes a WebSocket connection to the given URL
func (c *WSClient) Connect(url string) error {
	ctx := context.Background()
	conn, _, err := websocket.Dial(ctx, url, &websocket.DialOptions{
		HTTPClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	c.conn = conn

	// Start receiving messages in background
	go c.receiveMessages()

	return nil
}

// receiveMessages continuously reads messages from the WebSocket
func (c *WSClient) receiveMessages() {
	for {
		c.closedMu.RLock()
		if c.closed {
			c.closedMu.RUnlock()
			return
		}
		c.closedMu.RUnlock()

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		_, data, err := c.conn.Read(ctx)
		cancel()

		if err != nil {
			// Connection closed or error
			return
		}

		var msg models.WSMessage
		if err := json.Unmarshal(data, &msg); err == nil {
			c.messagesMu.Lock()
			c.messages = append(c.messages, msg)
			c.messagesMu.Unlock()
		}
	}
}

// SendMessage sends a message to the WebSocket
func (c *WSClient) SendMessage(msg map[string]any) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	ctx := context.Background()
	return c.conn.Write(ctx, websocket.MessageText, data)
}

// SendVote sends a vote message
func (c *WSClient) SendVote(participantID, value string) error {
	return c.SendMessage(map[string]any{
		"type": "vote",
		"payload": map[string]any{
			"participantId": participantID,
			"value":         value,
		},
	})
}

// SendReveal sends a reveal message
func (c *WSClient) SendReveal() error {
	return c.SendMessage(map[string]any{
		"type": "reveal",
	})
}

// SendReset sends a reset message
func (c *WSClient) SendReset() error {
	return c.SendMessage(map[string]any{
		"type": "reset",
	})
}

// WaitForMessage waits for any message with a timeout
func (c *WSClient) WaitForMessage(timeout time.Duration) *models.WSMessage {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		c.messagesMu.RLock()
		if len(c.messages) > 0 {
			msg := c.messages[0]
			c.messagesMu.RUnlock()
			return &msg
		}
		c.messagesMu.RUnlock()

		time.Sleep(10 * time.Millisecond)
	}

	return nil
}

// WaitForMessageType waits for a specific message type
func (c *WSClient) WaitForMessageType(msgType string, timeout time.Duration) *models.WSMessage {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		c.messagesMu.RLock()
		for _, msg := range c.messages {
			if msg.Type == msgType {
				c.messagesMu.RUnlock()
				return &msg
			}
		}
		c.messagesMu.RUnlock()

		time.Sleep(10 * time.Millisecond)
	}

	return nil
}

// WaitForMessageWithTimeout is an alias for WaitForMessage
func (c *WSClient) WaitForMessageWithTimeout(timeout time.Duration) *models.WSMessage {
	return c.WaitForMessage(timeout)
}

// ReceivedMessages returns all received messages
func (c *WSClient) ReceivedMessages() []models.WSMessage {
	c.messagesMu.RLock()
	defer c.messagesMu.RUnlock()

	messages := make([]models.WSMessage, len(c.messages))
	copy(messages, c.messages)
	return messages
}

// ClearMessages clears all received messages
func (c *WSClient) ClearMessages() {
	c.messagesMu.Lock()
	c.messages = make([]models.WSMessage, 0)
	c.messagesMu.Unlock()
}

// SetParticipantID sets the participant ID for this client
func (c *WSClient) SetParticipantID(id string) {
	c.participantID = id
}

// IsConnected returns whether the connection is active
func (c *WSClient) IsConnected() bool {
	c.closedMu.RLock()
	defer c.closedMu.RUnlock()
	return !c.closed && c.conn != nil
}

// Close closes the WebSocket connection
func (c *WSClient) Close() {
	c.closedMu.Lock()
	c.closed = true
	c.closedMu.Unlock()

	if c.conn != nil {
		c.conn.Close(websocket.StatusNormalClosure, "")
	}
}
