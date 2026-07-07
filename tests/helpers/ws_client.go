package helpers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"testing"
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

// Connect establishes a WebSocket connection to the given URL, anonymously
// (no session cookie). Anonymous connections are read-only in production
// (see internal/handlers/ws.go's handleMessage membership gate): they
// receive room_state/broadcasts but cannot vote/reveal/reset.
func (c *WSClient) Connect(url string) error {
	return c.connect(url, nil)
}

// ConnectWithHeader establishes a WebSocket connection to the given URL,
// sending the given HTTP headers (e.g. a Cookie header) with the handshake
// request. Use this to connect as a specific already-created participant -
// see ConnectTestClientAs.
func (c *WSClient) ConnectWithHeader(url string, header http.Header) error {
	return c.connect(url, header)
}

func (c *WSClient) connect(url string, header http.Header) error {
	ctx := context.Background()
	conn, _, err := websocket.Dial(ctx, url, &websocket.DialOptions{
		HTTPClient: &http.Client{
			Timeout: 5 * time.Second,
		},
		HTTPHeader: header,
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

// SendVote sends a vote message using the client's participant ID
func (c *WSClient) SendVote(t *testing.T, value string) {
	t.Helper()
	if err := c.SendMessage(map[string]any{
		"type": "vote",
		"payload": map[string]any{
			"participantId": c.participantID,
			"value":         value,
		},
	}); err != nil {
		t.Fatalf("Failed to send vote: %v", err)
	}
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

// WaitForMessageCount waits until at least n messages of the given type have
// been received, returning them (in receipt order) once that count is
// reached. It returns nil if the timeout elapses first. Unlike
// WaitForMessageType (which always returns the *first* match and so cannot
// distinguish a second broadcast of the same type from the first), this is
// the right tool for asserting on the Nth occurrence of a message type -
// e.g. the vote_cast broadcast for a second voter.
func (c *WSClient) WaitForMessageCount(msgType string, n int, timeout time.Duration) []models.WSMessage {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		c.messagesMu.RLock()
		matches := make([]models.WSMessage, 0, n)
		for _, msg := range c.messages {
			if msg.Type == msgType {
				matches = append(matches, msg)
			}
		}
		c.messagesMu.RUnlock()

		if len(matches) >= n {
			return matches
		}

		time.Sleep(10 * time.Millisecond)
	}

	return nil
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
		_ = c.conn.Close(websocket.StatusNormalClosure, "") // Best effort close
	}
}

// ExpectMessage waits for a specific message type and fails the test if not received
func (c *WSClient) ExpectMessage(t *testing.T, msgType string, timeout time.Duration) *models.WSMessage {
	t.Helper()
	msg := c.WaitForMessageType(msgType, timeout)
	if msg == nil {
		t.Fatalf("Expected message type %s within %v, but none received", msgType, timeout)
	}
	return msg
}

// TryReadMessage attempts to read a message with a timeout, returns nil if none available
func (c *WSClient) TryReadMessage(timeout time.Duration) *models.WSMessage {
	return c.WaitForMessage(timeout)
}

// ExpectNoMessageType asserts that no message of the given type is observed
// within window. It polls messages already buffered plus any that arrive
// during window and fails immediately once a match is seen, otherwise it
// waits out the full window before concluding absence. This is the bounded
// drain-then-assert-absence pattern for negative assertions - preferred over
// a bare time.Sleep followed by a single check, since it fails fast and
// documents exactly how long "no message arrived" was verified for.
func (c *WSClient) ExpectNoMessageType(t *testing.T, msgType string, window time.Duration) {
	t.Helper()

	deadline := time.Now().Add(window)
	for time.Now().Before(deadline) {
		c.messagesMu.RLock()
		for _, msg := range c.messages {
			if msg.Type == msgType {
				c.messagesMu.RUnlock()
				t.Fatalf("expected no %q message within %v, but one was received", msgType, window)
				return
			}
		}
		c.messagesMu.RUnlock()

		time.Sleep(10 * time.Millisecond)
	}
}

// ConnectTestClient creates a WebSocket client and connects to a room
// anonymously (no session cookie). Anonymous clients are read-only in
// production - use ConnectTestClientAs to connect as a specific participant
// able to vote/reveal/reset.
func ConnectTestClient(t *testing.T, ts *TestHTTPServer, roomID string) *WSClient {
	t.Helper()

	client := NewWSClient()
	wsURL := fmt.Sprintf("ws://%s/ws/%s", ts.URL, roomID)

	if err := client.Connect(wsURL); err != nil {
		t.Fatalf("Failed to connect WebSocket client: %v", err)
	}

	return client
}

// ConnectTestClientAs creates a WebSocket client and connects to a room,
// authenticated as an already-created participant via the same per-room
// session cookie the production JoinRoom handler sets (see
// internal/handlers/room.go's participantCookieName: "pp_participant_<roomID>").
// This is required for the connection to be treated as a mutating
// participant rather than a read-only anonymous observer - see
// WSHandler.HandleWebSocket / handleMessage's membership gate.
func ConnectTestClientAs(t *testing.T, ts *TestHTTPServer, roomID, participantID, sessionCookie string) *WSClient {
	t.Helper()

	client := NewWSClient()
	client.SetParticipantID(participantID)

	wsURL := fmt.Sprintf("ws://%s/ws/%s", ts.URL, roomID)

	header := http.Header{}
	header.Set("Cookie", fmt.Sprintf("pp_participant_%s=%s", roomID, sessionCookie))

	if err := client.ConnectWithHeader(wsURL, header); err != nil {
		t.Fatalf("Failed to connect WebSocket client as participant %s: %v", participantID, err)
	}

	return client
}
