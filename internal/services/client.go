package services

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/damione1/planning-poker/internal/config"
	"github.com/damione1/planning-poker/internal/models"
)

// Client represents a single WebSocket connection with its own send goroutine
type Client struct {
	conn          *websocket.Conn
	send          chan []byte
	hub           *Hub
	roomID        string
	participantID string

	// Rate limiting
	messageCount  int
	rateLimitMu   sync.Mutex
	lastReset     time.Time

	// Lifecycle
	ctx           context.Context
	cancel        context.CancelFunc
	closed        bool
	closeMu       sync.Mutex
}

// NewClient creates a new client instance
func NewClient(conn *websocket.Conn, hub *Hub, roomID, participantID string) *Client {
	ctx, cancel := context.WithCancel(context.Background())

	return &Client{
		conn:          conn,
		send:          make(chan []byte, config.ClientSendBufferSize),
		hub:           hub,
		roomID:        roomID,
		participantID: participantID,
		lastReset:     time.Now(),
		ctx:           ctx,
		cancel:        cancel,
	}
}

// Start begins the client's read and write pumps
func (c *Client) Start() {
	go c.writePump()
	go c.readPump()
}

// writePump handles outgoing messages to the WebSocket connection
func (c *Client) writePump() {
	ticker := time.NewTicker(config.PingInterval)
	defer func() {
		ticker.Stop()
		c.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				// Channel closed, connection is closing
				_ = c.conn.Close(websocket.StatusNormalClosure, "")
				return
			}

			// Use context with timeout for write operations
			writeCtx, cancel := context.WithTimeout(c.ctx, config.WriteTimeout)
			err := c.conn.Write(writeCtx, websocket.MessageText, message)
			cancel()

			if err != nil {
				log.Printf("❌ Write error (room=%s, participant=%s): %v", c.roomID, c.participantID, err)
				c.hub.metrics.IncrementBroadcastErrors()
				return
			}
			c.hub.metrics.IncrementMessagesSent()

		case <-ticker.C:
			// Send ping to keep connection alive
			pingCtx, cancel := context.WithTimeout(c.ctx, config.WriteTimeout)
			err := c.conn.Ping(pingCtx)
			cancel()

			if err != nil {
				log.Printf("❌ Ping error (room=%s): %v", c.roomID, err)
				return
			}

		case <-c.ctx.Done():
			return
		}
	}
}

// readPump handles incoming messages from the WebSocket connection
func (c *Client) readPump() {
	defer func() {
		c.hub.Unregister(c.roomID, c)
		c.Close()
	}()

	for {
		// Use context with timeout for read operations
		readCtx, cancel := context.WithTimeout(c.ctx, config.PongTimeout)
		_, message, err := c.conn.Read(readCtx)
		cancel()

		if err != nil {
			if websocket.CloseStatus(err) != websocket.StatusNormalClosure {
				log.Printf("❌ Read error (room=%s, participant=%s): %v", c.roomID, c.participantID, err)
				c.hub.metrics.IncrementConnectionErrors()
			}
			return
		}

		// Rate limiting check
		if !c.checkRateLimit() {
			log.Printf("⚠️  Rate limit exceeded (room=%s, participant=%s)", c.roomID, c.participantID)
			c.hub.metrics.IncrementRateLimitViolations()

			// Send rate limit error to client
			errMsg := &models.WSMessage{
				Type: "error",
				Payload: map[string]string{
					"message": "Rate limit exceeded. Please slow down.",
				},
			}
			c.hub.SendToClient(c, errMsg)
			continue
		}

		c.hub.metrics.IncrementMessagesReceived()

		// Forward message to hub for processing
		c.hub.handleMessage <- &ClientMessage{
			Client:  c,
			Message: message,
		}
	}
}

// checkRateLimit verifies the client hasn't exceeded message rate limits
func (c *Client) checkRateLimit() bool {
	c.rateLimitMu.Lock()
	defer c.rateLimitMu.Unlock()

	now := time.Now()
	if now.Sub(c.lastReset) > config.RateLimitWindow {
		c.messageCount = 0
		c.lastReset = now
	}

	c.messageCount++
	return c.messageCount <= config.MaxMessagesPerSecond
}

// Send queues a message for sending to the client
func (c *Client) Send(message []byte) bool {
	c.closeMu.Lock()
	defer c.closeMu.Unlock()

	if c.closed {
		return false
	}

	select {
	case c.send <- message:
		return true
	default:
		// Channel full, client is too slow
		log.Printf("⚠️  Send buffer full, closing slow client (room=%s, participant=%s)", c.roomID, c.participantID)
		c.hub.metrics.IncrementBroadcastErrors()
		go c.Close()
		return false
	}
}

// Close cleanly shuts down the client connection
func (c *Client) Close() {
	c.closeMu.Lock()
	defer c.closeMu.Unlock()

	if c.closed {
		return
	}

	c.closed = true
	c.cancel()
	close(c.send)
	_ = c.conn.Close(websocket.StatusNormalClosure, "")
}

// ClientMessage represents a message received from a client
type ClientMessage struct {
	Client  *Client
	Message []byte
}
