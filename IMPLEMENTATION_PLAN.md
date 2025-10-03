# Planning Poker SaaS - Implementation Plan

**Complete architectural guide for building a modern, real-time Planning Poker application on PocketBase**

## Table of Contents
1. [Project Overview](#project-overview)
2. [Technology Stack](#technology-stack)
3. [Project Structure](#project-structure)
4. [Architecture Design](#architecture-design)
5. [Implementation Phases](#implementation-phases)
6. [Technical Specifications](#technical-specifications)
7. [Configuration & Deployment](#configuration--deployment)
8. [Best Practices](#best-practices)

---

## Project Overview

### Vision
Build a clean, modern Planning Poker SaaS with minimal JavaScript, server-side rendering, and real-time WebSocket communication. The application runs as a single container with embedded frontend and backend.

### Core Features
- **Public Rooms**: No authentication required (MVP)
- **Real-time Voting**: WebSocket-based live updates
- **Flexible Pointing**: Fibonacci or custom values
- **Modern UI**: Voters displayed around a virtual table
- **Easy Sharing**: QR code + URL copy functionality
- **Anonymous Participation**: Name-only identification
- **Role Selection**: Voter or Spectator

### User Flow
```
Landing Page
    ↓
Enter Room Name + Pointing Method (Fibonacci/Custom)
    ↓
Redirect to /room/{uuid}
    ↓
If not identified → Join Modal (Name + Role)
    ↓
Room Interface
    - Participants displayed in grid
    - Vote cards at bottom (voters only)
    - Reveal/Reset controls
    - QR code + URL sharing
```

---

## Technology Stack

### Backend
- **Framework**: PocketBase v0.30+ (Go-based backend framework with embedded Echo router, SQLite, authentication, admin UI)
- **WebSocket**: github.com/coder/websocket (modern, idiomatic Go library)
- **Templating**: github.com/a-h/templ (type-safe HTML generation)
- **Database**: SQLite (embedded in PocketBase, unused in MVP - using in-memory state only)
- **Language**: Go 1.23+

**Note**: PocketBase includes Echo v4 router, no separate Echo import needed

### Frontend
- **Hypermedia**: htmx 2.0 (dynamic HTML updates)
- **WebSocket Extension**: htmx-ext-ws (real-time communication)
- **Interactivity**: Alpine.js 3.14 (minimal JavaScript framework)
- **Styling**: Tailwind CSS or custom CSS
- **Build**: No build step required (CDN for htmx/Alpine)

### Development Tools
- **Live Reload**: Air (Go live reload tool)
- **Container**: Docker (single container deployment)
- **Version Control**: Git

### Philosophy
- **Minimal JavaScript**: htmx + Alpine.js handle all interactivity
- **Server-Side Rendering**: Templ generates HTML on server
- **No Build Complexity**: No npm, webpack, or bundlers required
- **Single Container**: Backend + frontend in one deployable unit
- **Progressive Enhancement**: Works without JavaScript (basic functionality)

---

## Project Structure

```
planning-poker/
├── main.go                           # PocketBase app initialization, route registration
├── go.mod                            # Go module definition
├── go.sum                            # Dependency checksums
├── Dockerfile                        # Single container build
├── .air.toml                         # Live reload configuration
├── .gitignore                        # Git ignore (pb_data/, tmp/, etc.)
├── README.md                         # Project documentation
│
├── internal/
│   ├── config/
│   │   └── config.go                # Environment configuration loading
│   │
│   ├── models/
│   │   ├── room.go                  # Room struct + concurrent-safe methods
│   │   ├── participant.go           # Participant struct + role enum
│   │   ├── vote.go                  # Vote struct
│   │   └── message.go               # WebSocket message types
│   │
│   ├── services/
│   │   ├── room_manager.go          # In-memory room state (CRUD + TTL cleanup)
│   │   ├── hub.go                   # WebSocket hub (broadcast pattern)
│   │   └── session.go               # Cookie-based participant sessions
│   │
│   ├── handlers/
│   │   ├── home.go                  # GET / - landing page
│   │   ├── room.go                  # POST /room, GET /room/:id
│   │   ├── join.go                  # POST /room/:id/join - participant join
│   │   └── ws.go                    # GET /ws/:roomId - WebSocket upgrade
│   │
│   └── utils/
│       ├── validator.go             # Input validation helpers
│       └── response.go              # HTTP response helpers
│
├── web/
│   ├── templates/
│   │   ├── base.templ               # Base HTML layout with scripts
│   │   ├── home.templ               # Landing page (room creation form)
│   │   ├── room.templ               # Room interface (main view)
│   │   │
│   │   └── components/
│   │       ├── join_modal.templ     # Name + role selection modal
│   │       ├── participant_grid.templ   # Voters displayed in grid/table
│   │       ├── voting_cards.templ   # Card selector buttons
│   │       ├── controls.templ       # Reveal/Reset buttons
│   │       ├── share.templ          # QR code + copy URL widget
│   │       └── stats.templ          # Vote statistics (average, consensus)
│   │
│   └── static/
│       ├── css/
│       │   └── styles.css           # Custom CSS or Tailwind build
│       ├── js/
│       │   └── alpine-components.js # Alpine.js data components
│       └── images/
│           └── logo.svg             # Branding assets (optional)
│
└── pb_data/                          # PocketBase data directory (gitignored)
    ├── data.db                       # SQLite database (unused in MVP)
    └── logs.db                       # Application logs
```

### File Purposes

**Core Application:**
- `main.go`: PocketBase initialization, route registration, server start
- `go.mod`: Dependencies (PocketBase, websocket, templ, uuid)

**Configuration:**
- `config/config.go`: Environment variables, app settings, defaults

**Data Models:**
- `models/room.go`: Room struct with participants, votes, state
- `models/participant.go`: Participant struct with name, role, connection status
- `models/vote.go`: Vote struct with participant ID and value
- `models/message.go`: WebSocket message types and payloads

**Business Logic:**
- `services/room_manager.go`: In-memory room storage, CRUD operations, cleanup
- `services/hub.go`: WebSocket hub for broadcasting to room participants
- `services/session.go`: Cookie-based participant identification

**HTTP Handlers:**
- `handlers/home.go`: Render landing page
- `handlers/room.go`: Create room, view room
- `handlers/join.go`: Join room as participant
- `handlers/ws.go`: WebSocket connection upgrade and message handling

**Templates:**
- `base.templ`: HTML layout, script includes, shared structure
- `home.templ`: Room creation form
- `room.templ`: Main room interface
- `components/*.templ`: Reusable UI components

**Static Assets:**
- `static/css/styles.css`: Styling
- `static/js/alpine-components.js`: Alpine.js reactive components

---

## Architecture Design

### System Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Client (Browser)                         │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐     │
│  │    htmx      │  │  Alpine.js   │  │  WebSocket   │     │
│  │ (HTTP/HTML)  │  │ (UI State)   │  │ (Real-time)  │     │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘     │
└─────────┼──────────────────┼──────────────────┼─────────────┘
          │                  │                  │
          │ HTTP             │ Minimal JS       │ WebSocket
          ↓                  ↓                  ↓
┌─────────────────────────────────────────────────────────────┐
│                  PocketBase Application                      │
│                                                              │
│  ┌────────────────────────────────────────────────────────┐ │
│  │                    HTTP Handlers                        │ │
│  │  [Home] [CreateRoom] [RoomView] [Join] [WebSocket]    │ │
│  └───────┬────────────────────────────────────────────────┘ │
│          │                                                   │
│  ┌───────▼────────────┐        ┌─────────────────────┐     │
│  │   Room Manager     │◄──────►│  WebSocket Hub      │     │
│  │  (In-Memory State) │        │ (Broadcast Pattern) │     │
│  └────────────────────┘        └─────────────────────┘     │
│          │                                                   │
│  ┌───────▼────────────┐                                     │
│  │   Templ Templates  │                                     │
│  │  (HTML Generation) │                                     │
│  └────────────────────┘                                     │
└─────────────────────────────────────────────────────────────┘
```

### In-Memory State Management

**RoomManager** - Concurrent-safe room storage:
```go
type RoomManager struct {
    rooms map[string]*Room  // Key: Room UUID
    mu    sync.RWMutex      // Protects concurrent access
}

// Operations:
- CreateRoom(name, pointingMethod, customValues) -> Room
- GetRoom(id) -> Room
- UpdateRoom(id, room) -> error
- DeleteRoom(id) -> error
- CleanupInactiveRooms() // Background goroutine
```

**Room Model** - Core data structure:
```go
type Room struct {
    ID              string                 // UUID
    Name            string                 // User-provided name
    PointingMethod  string                 // "fibonacci" or "custom"
    CustomValues    []string               // For custom pointing
    State           RoomState              // "voting" or "revealed"
    Participants    map[string]*Participant // Key: Participant UUID
    Votes           map[string]string      // Key: Participant UUID, Value: vote
    CreatedAt       time.Time
    LastActivity    time.Time
    mu              sync.RWMutex           // Protects room-level concurrent access
}

type RoomState string
const (
    StateVoting   RoomState = "voting"
    StateRevealed RoomState = "revealed"
)
```

**Participant Model**:
```go
type Participant struct {
    ID          string              // UUID
    Name        string              // User-provided name
    Role        ParticipantRole     // "voter" or "spectator"
    Connected   bool                // WebSocket connection status
    JoinedAt    time.Time
}

type ParticipantRole string
const (
    RoleVoter     ParticipantRole = "voter"
    RoleSpectator ParticipantRole = "spectator"
)
```

### WebSocket Hub Pattern

**Hub Architecture** - Central message broadcaster:
```go
type Hub struct {
    // Room connections: roomId -> set of WebSocket connections
    rooms map[string]map[*websocket.Conn]bool

    // Message broadcast channel
    broadcast chan *Message

    // Connection registration
    register   chan *Registration
    unregister chan *Registration

    // Concurrent access protection
    mu sync.RWMutex
}

type Registration struct {
    RoomID string
    Conn   *websocket.Conn
}

type Message struct {
    RoomID  string
    Type    string
    Payload interface{}
}
```

**Hub Lifecycle**:
```
1. Hub.Run() starts in goroutine on app start
2. Client connects → WebSocket upgrade
3. Hub registers connection for room
4. Client sends message → Handler processes → Hub broadcasts
5. Hub sends to all connections in room
6. Client disconnects → Hub unregisters
```

**Message Flow**:
```
Client → ws.Handler → Parse → RoomManager (update state) → Hub.broadcast chan
       ↓
Hub.run() goroutine
       ↓
For each connection in room → Send JSON message
       ↓
Client receives → htmx processes → DOM update
```

### WebSocket Message Protocol

**Client → Server Messages**:
```json
// Join room as participant
{
  "type": "join",
  "roomId": "550e8400-e29b-41d4-a716-446655440000",
  "name": "Alice",
  "role": "voter"
}

// Cast vote
{
  "type": "vote",
  "roomId": "550e8400-e29b-41d4-a716-446655440000",
  "value": "5"
}

// Reveal all votes (voters only)
{
  "type": "reveal",
  "roomId": "550e8400-e29b-41d4-a716-446655440000"
}

// Reset voting round (voters only)
{
  "type": "reset",
  "roomId": "550e8400-e29b-41d4-a716-446655440000"
}
```

**Server → Client Messages**:
```json
// Participant joined
{
  "type": "participant_joined",
  "participant": {
    "id": "123e4567-e89b-12d3-a456-426614174000",
    "name": "Bob",
    "role": "voter",
    "connected": true
  }
}

// Participant left
{
  "type": "participant_left",
  "participantId": "123e4567-e89b-12d3-a456-426614174000"
}

// Vote cast (shows participant voted, but not value)
{
  "type": "vote_cast",
  "participantId": "123e4567-e89b-12d3-a456-426614174000",
  "hasVoted": true
}

// Votes revealed (show all votes)
{
  "type": "votes_revealed",
  "votes": [
    {"participantId": "...", "participantName": "Alice", "value": "5"},
    {"participantId": "...", "participantName": "Bob", "value": "8"}
  ],
  "stats": {
    "average": 6.5,
    "min": 5,
    "max": 8
  }
}

// Room reset (new voting round)
{
  "type": "room_reset"
}
```

### Session Management

**Cookie-based Participant Identification**:
```
1. User visits /room/:id for first time
2. No participant cookie → Show join modal
3. User submits name + role → Create participant
4. Set cookie: participant_id={uuid}, path=/room/:id
5. Future visits → Auto-identify, skip modal
6. WebSocket reconnect → Rejoin with same participant ID
```

**Cookie Structure**:
```
Name: planning_poker_participant
Value: {participantId}|{roomId}
Path: /room/{roomId}
MaxAge: 24 hours
HttpOnly: true
SameSite: Lax
```

---

## Implementation Phases

### Phase 1: Project Setup & Foundation (Week 1, Days 1-2)

**Goal**: Initialize project with dependencies and basic structure

**Tasks**:
1. Initialize Go module:
   ```bash
   mkdir planning-poker && cd planning-poker
   go mod init github.com/yourusername/planning-poker
   ```

2. Install dependencies:
   ```bash
   go get github.com/pocketbase/pocketbase@latest
   go get github.com/coder/websocket@latest
   go get github.com/a-h/templ/cmd/templ@latest
   go get github.com/google/uuid@latest
   go get github.com/skip2/go-qrcode@latest
   ```

   **Note**: PocketBase bundles Echo router internally - no separate Echo installation needed

3. Create directory structure:
   ```bash
   mkdir -p internal/{config,models,services,handlers,utils}
   mkdir -p web/{templates/components,static/{css,js,images}}
   ```

4. Create `main.go`:
   ```go
   package main

   import (
       "log"
       "github.com/pocketbase/pocketbase"
       "github.com/pocketbase/pocketbase/core"
   )

   func main() {
       app := pocketbase.New()

       app.OnServe().BindFunc(func(se *core.ServeEvent) error {
           // Route registration will go here
           se.Router.GET("/", func(re *core.RequestEvent) error {
               return re.String(200, "Planning Poker - Coming Soon")
           })
           return se.Next()
       })

       if err := app.Start(); err != nil {
           log.Fatal(err)
       }
   }
   ```

5. Create `.gitignore`:
   ```
   pb_data/
   tmp/
   *.log
   .env
   .DS_Store
   ```

6. Create `.air.toml` for live reload:
   ```toml
   root = "."
   tmp_dir = "tmp"

   [build]
   cmd = "templ generate && go build -o ./tmp/main ."
   bin = "tmp/main serve --http=127.0.0.1:8090"
   include_ext = ["go", "templ"]
   exclude_dir = ["tmp", "pb_data", "web/static"]
   delay = 1000
   ```

7. Install Air globally:
   ```bash
   go install github.com/air-verse/air@latest
   ```

8. Test setup:
   ```bash
   air
   # Visit http://localhost:8090
   ```

**Deliverables**:
- Working Go module with PocketBase
- Project structure created
- Live reload configured
- Basic "Hello World" endpoint

---

### Phase 2: Data Models & State Management (Week 1, Days 3-4)

**Goal**: Define core data structures and in-memory storage

**Files to Create**:

**`internal/models/room.go`**:
```go
package models

import (
    "sync"
    "time"
)

type RoomState string

const (
    StateVoting   RoomState = "voting"
    StateRevealed RoomState = "revealed"
)

type Room struct {
    ID              string
    Name            string
    PointingMethod  string   // "fibonacci" or "custom"
    CustomValues    []string // For custom pointing
    State           RoomState
    Participants    map[string]*Participant
    Votes           map[string]string
    CreatedAt       time.Time
    LastActivity    time.Time
    mu              sync.RWMutex
}

func NewRoom(id, name, pointingMethod string, customValues []string) *Room {
    return &Room{
        ID:             id,
        Name:           name,
        PointingMethod: pointingMethod,
        CustomValues:   customValues,
        State:          StateVoting,
        Participants:   make(map[string]*Participant),
        Votes:          make(map[string]string),
        CreatedAt:      time.Now(),
        LastActivity:   time.Now(),
    }
}

func (r *Room) AddParticipant(p *Participant) {
    r.mu.Lock()
    defer r.mu.Unlock()
    r.Participants[p.ID] = p
    r.LastActivity = time.Now()
}

func (r *Room) RemoveParticipant(participantID string) {
    r.mu.Lock()
    defer r.mu.Unlock()
    delete(r.Participants, participantID)
    delete(r.Votes, participantID)
    r.LastActivity = time.Now()
}

func (r *Room) CastVote(participantID, value string) {
    r.mu.Lock()
    defer r.mu.Unlock()
    r.Votes[participantID] = value
    r.LastActivity = time.Now()
}

func (r *Room) RevealVotes() {
    r.mu.Lock()
    defer r.mu.Unlock()
    r.State = StateRevealed
    r.LastActivity = time.Now()
}

func (r *Room) ResetVoting() {
    r.mu.Lock()
    defer r.mu.Unlock()
    r.State = StateVoting
    r.Votes = make(map[string]string)
    r.LastActivity = time.Now()
}

func (r *Room) GetVoteStats() map[string]interface{} {
    r.mu.RLock()
    defer r.mu.RUnlock()

    // Calculate stats only for numeric votes
    // Implementation details in Phase 9
    return map[string]interface{}{
        "total": len(r.Votes),
    }
}
```

**`internal/models/participant.go`**:
```go
package models

import "time"

type ParticipantRole string

const (
    RoleVoter     ParticipantRole = "voter"
    RoleSpectator ParticipantRole = "spectator"
)

type Participant struct {
    ID        string
    Name      string
    Role      ParticipantRole
    Connected bool
    JoinedAt  time.Time
}

func NewParticipant(id, name string, role ParticipantRole) *Participant {
    return &Participant{
        ID:        id,
        Name:      name,
        Role:      role,
        Connected: false,
        JoinedAt:  time.Now(),
    }
}
```

**`internal/models/message.go`**:
```go
package models

type WSMessage struct {
    Type    string      `json:"type"`
    RoomID  string      `json:"roomId,omitempty"`
    Payload interface{} `json:"payload,omitempty"`
}

// Client → Server message types
const (
    MsgTypeJoin   = "join"
    MsgTypeVote   = "vote"
    MsgTypeReveal = "reveal"
    MsgTypeReset  = "reset"
)

// Server → Client message types
const (
    MsgTypeParticipantJoined = "participant_joined"
    MsgTypeParticipantLeft   = "participant_left"
    MsgTypeVoteCast          = "vote_cast"
    MsgTypeVotesRevealed     = "votes_revealed"
    MsgTypeRoomReset         = "room_reset"
)
```

**`internal/services/room_manager.go`**:
```go
package services

import (
    "fmt"
    "sync"
    "time"
    "github.com/yourusername/planning-poker/internal/models"
)

type RoomManager struct {
    rooms map[string]*models.Room
    mu    sync.RWMutex
}

func NewRoomManager() *RoomManager {
    rm := &RoomManager{
        rooms: make(map[string]*models.Room),
    }

    // Start cleanup goroutine
    go rm.cleanupLoop()

    return rm
}

func (rm *RoomManager) CreateRoom(id, name, pointingMethod string, customValues []string) *models.Room {
    rm.mu.Lock()
    defer rm.mu.Unlock()

    room := models.NewRoom(id, name, pointingMethod, customValues)
    rm.rooms[id] = room
    return room
}

func (rm *RoomManager) GetRoom(id string) (*models.Room, error) {
    rm.mu.RLock()
    defer rm.mu.RUnlock()

    room, exists := rm.rooms[id]
    if !exists {
        return nil, fmt.Errorf("room not found")
    }
    return room, nil
}

func (rm *RoomManager) DeleteRoom(id string) {
    rm.mu.Lock()
    defer rm.mu.Unlock()
    delete(rm.rooms, id)
}

func (rm *RoomManager) cleanupLoop() {
    ticker := time.NewTicker(1 * time.Hour)
    defer ticker.Stop()

    for range ticker.C {
        rm.cleanupInactiveRooms()
    }
}

func (rm *RoomManager) cleanupInactiveRooms() {
    rm.mu.Lock()
    defer rm.mu.Unlock()

    cutoff := time.Now().Add(-24 * time.Hour)

    for id, room := range rm.rooms {
        room.mu.RLock()
        lastActivity := room.LastActivity
        room.mu.RUnlock()

        if lastActivity.Before(cutoff) {
            delete(rm.rooms, id)
        }
    }
}
```

**Deliverables**:
- Room, Participant, Message models defined
- RoomManager with concurrent-safe operations
- Automatic room cleanup (24h TTL)

---

### Phase 3: WebSocket Infrastructure (Week 1-2, Days 5-7)

**Goal**: Implement WebSocket hub for real-time broadcasting

**`internal/services/hub.go`**:
```go
package services

import (
    "context"
    "encoding/json"
    "log"
    "sync"
    "github.com/coder/websocket"
    "github.com/yourusername/planning-poker/internal/models"
)

type Hub struct {
    // Room connections: roomId -> set of connections
    rooms map[string]map[*websocket.Conn]bool

    // Broadcast message to room
    broadcast chan *BroadcastMessage

    // Register connection to room
    register chan *Registration

    // Unregister connection from room
    unregister chan *Registration

    mu sync.RWMutex
}

type Registration struct {
    RoomID string
    Conn   *websocket.Conn
}

type BroadcastMessage struct {
    RoomID  string
    Message *models.WSMessage
}

func NewHub() *Hub {
    return &Hub{
        rooms:      make(map[string]map[*websocket.Conn]bool),
        broadcast:  make(chan *BroadcastMessage, 256),
        register:   make(chan *Registration),
        unregister: make(chan *Registration),
    }
}

func (h *Hub) Run() {
    for {
        select {
        case reg := <-h.register:
            h.registerConnection(reg)

        case reg := <-h.unregister:
            h.unregisterConnection(reg)

        case msg := <-h.broadcast:
            h.broadcastToRoom(msg)
        }
    }
}

func (h *Hub) registerConnection(reg *Registration) {
    h.mu.Lock()
    defer h.mu.Unlock()

    if h.rooms[reg.RoomID] == nil {
        h.rooms[reg.RoomID] = make(map[*websocket.Conn]bool)
    }
    h.rooms[reg.RoomID][reg.Conn] = true
}

func (h *Hub) unregisterConnection(reg *Registration) {
    h.mu.Lock()
    defer h.mu.Unlock()

    if connections, ok := h.rooms[reg.RoomID]; ok {
        if _, exists := connections[reg.Conn]; exists {
            delete(connections, reg.Conn)
            reg.Conn.Close(websocket.StatusNormalClosure, "")

            // Clean up empty rooms
            if len(connections) == 0 {
                delete(h.rooms, reg.RoomID)
            }
        }
    }
}

func (h *Hub) broadcastToRoom(msg *BroadcastMessage) {
    h.mu.RLock()
    connections := h.rooms[msg.RoomID]
    h.mu.RUnlock()

    if connections == nil {
        return
    }

    data, err := json.Marshal(msg.Message)
    if err != nil {
        log.Printf("Error marshaling message: %v", err)
        return
    }

    for conn := range connections {
        go func(c *websocket.Conn) {
            err := c.Write(context.Background(), websocket.MessageText, data)
            if err != nil {
                log.Printf("Error writing to WebSocket: %v", err)
            }
        }(conn)
    }
}

func (h *Hub) BroadcastToRoom(roomID string, message *models.WSMessage) {
    h.broadcast <- &BroadcastMessage{
        RoomID:  roomID,
        Message: message,
    }
}
```

**Deliverables**:
- WebSocket Hub with broadcast pattern
- Connection registration/unregistration
- Room-specific message broadcasting
- Concurrent-safe connection management

---

### Phase 4: HTTP Handlers (Week 2, Days 8-10)

**Goal**: Create HTTP endpoints for pages and WebSocket

**`internal/handlers/home.go`**:
```go
package handlers

import (
    "github.com/pocketbase/pocketbase/core"
    "github.com/yourusername/planning-poker/web/templates"
)

func Home(re *core.RequestEvent) error {
    component := templates.Home()
    return templates.Render(re.Response, re.Request, component)
}
```

**`internal/handlers/room.go`**:
```go
package handlers

import (
    "net/http"
    "github.com/google/uuid"
    "github.com/labstack/echo/v4"
    "github.com/pocketbase/pocketbase/core"
    "github.com/yourusername/planning-poker/internal/services"
    "github.com/yourusername/planning-poker/web/templates"
)

type RoomHandlers struct {
    roomManager *services.RoomManager
}

func NewRoomHandlers(rm *services.RoomManager) *RoomHandlers {
    return &RoomHandlers{roomManager: rm}
}

func (h *RoomHandlers) CreateRoom(re *core.RequestEvent) error {
    name := re.Request.FormValue("name")
    pointingMethod := re.Request.FormValue("pointingMethod")
    customValues := re.Request.Form["customValues"]

    // Validation
    if name == "" {
        return re.JSON(http.StatusBadRequest, map[string]string{
            "error": "Room name is required",
        })
    }

    // Create room
    roomID := uuid.New().String()
    h.roomManager.CreateRoom(roomID, name, pointingMethod, customValues)

    // Redirect to room
    return re.Redirect(http.StatusSeeOther, "/room/"+roomID)
}

// Note: re.Request uses standard *http.Request
// re.Response uses http.ResponseWriter
// PocketBase's RequestEvent wraps these for convenience

func (h *RoomHandlers) RoomView(re *core.RequestEvent) error {
    roomID := re.Request.PathValue("id")

    room, err := h.roomManager.GetRoom(roomID)
    if err != nil {
        return re.Redirect(http.StatusSeeOther, "/")
    }

    // Check for participant cookie
    cookie, err := re.Request.Cookie("planning_poker_participant")
    var participant *models.Participant

    if err == nil {
        // Parse participant from cookie
        // Implementation in Phase 8
    }

    component := templates.Room(room, participant)
    return templates.Render(re.Response, re.Request, component)
}
```

**`internal/handlers/ws.go`**:
```go
package handlers

import (
    "context"
    "encoding/json"
    "log"
    "github.com/coder/websocket"
    "github.com/pocketbase/pocketbase/core"
    "github.com/yourusername/planning-poker/internal/models"
    "github.com/yourusername/planning-poker/internal/services"
)

type WSHandler struct {
    hub         *services.Hub
    roomManager *services.RoomManager
}

func NewWSHandler(hub *services.Hub, rm *services.RoomManager) *WSHandler {
    return &WSHandler{
        hub:         hub,
        roomManager: rm,
    }
}

func (h *WSHandler) HandleWebSocket(re *core.RequestEvent) error {
    roomID := re.Request.PathValue("roomId")

    // Verify room exists
    _, err := h.roomManager.GetRoom(roomID)
    if err != nil {
        return re.JSON(404, map[string]string{"error": "Room not found"})
    }

    // Upgrade to WebSocket
    conn, err := websocket.Accept(re.Response, re.Request, &websocket.AcceptOptions{
        OriginPatterns: []string{"*"}, // Configure based on environment
    })
    if err != nil {
        return err
    }
    defer conn.Close(websocket.StatusInternalError, "")

    // Register connection
    h.hub.register <- &services.Registration{
        RoomID: roomID,
        Conn:   conn,
    }
    defer func() {
        h.hub.unregister <- &services.Registration{
            RoomID: roomID,
            Conn:   conn,
        }
    }()

    // Message loop
    ctx := context.Background()
    for {
        _, data, err := conn.Read(ctx)
        if err != nil {
            break
        }

        var msg models.WSMessage
        if err := json.Unmarshal(data, &msg); err != nil {
            log.Printf("Error unmarshaling message: %v", err)
            continue
        }

        h.handleMessage(roomID, &msg)
    }

    return nil
}

func (h *WSHandler) handleMessage(roomID string, msg *models.WSMessage) {
    room, err := h.roomManager.GetRoom(roomID)
    if err != nil {
        return
    }

    switch msg.Type {
    case models.MsgTypeVote:
        // Handle vote (implementation in Phase 7)
    case models.MsgTypeReveal:
        // Handle reveal (implementation in Phase 7)
    case models.MsgTypeReset:
        // Handle reset (implementation in Phase 7)
    }
}
```

**Update `main.go`** to register routes:
```go
package main

import (
    "log"
    "net/http"
    "github.com/pocketbase/pocketbase"
    "github.com/pocketbase/pocketbase/core"
    "github.com/yourusername/planning-poker/internal/handlers"
    "github.com/yourusername/planning-poker/internal/services"
)

func main() {
    app := pocketbase.New()

    // Initialize services
    roomManager := services.NewRoomManager()
    hub := services.NewHub()
    go hub.Run()

    // Initialize handlers
    roomHandlers := handlers.NewRoomHandlers(roomManager)
    wsHandler := handlers.NewWSHandler(hub, roomManager)

    app.OnServe().BindFunc(func(se *core.ServeEvent) error {
        // Static files
        se.Router.GET("/static/*", echo.WrapHandler(
            http.StripPrefix("/static/",
                http.FileServer(http.Dir("web/static")))))

        // Page routes
        se.Router.GET("/", handlers.Home)
        se.Router.POST("/room", roomHandlers.CreateRoom)
        se.Router.GET("/room/:id", roomHandlers.RoomView)

        // WebSocket route
        se.Router.GET("/ws/:roomId", wsHandler.HandleWebSocket)

        return se.Next()
    })

    if err := app.Start(); err != nil {
        log.Fatal(err)
    }
}
```

**Deliverables**:
- Home page handler
- Room creation and view handlers
- WebSocket connection upgrade
- Route registration in main.go

---

### Phase 5: Templ Templates (Week 2, Days 11-12)

**Goal**: Create server-side rendered HTML templates

**`web/templates/base.templ`**:
```templ
package templates

templ Base(title string) {
    <!DOCTYPE html>
    <html lang="en">
    <head>
        <meta charset="UTF-8"/>
        <meta name="viewport" content="width=device-width, initial-scale=1.0"/>
        <title>{title} - Planning Poker</title>
        <link rel="stylesheet" href="/static/css/styles.css"/>
        <script src="https://unpkg.com/htmx.org@2.0.0"></script>
        <script src="https://unpkg.com/htmx-ext-ws@2.0.0/ws.js"></script>
        <script src="https://unpkg.com/alpinejs@3.14.1/dist/cdn.min.js" defer></script>
    </head>
    <body>
        <div class="container">
            { children... }
        </div>
    </body>
    </html>
}
```

**`web/templates/home.templ`**:
```templ
package templates

templ Home() {
    @Base("Create Room") {
        <div class="home-page">
            <h1>Planning Poker</h1>
            <p>Create a room to start estimating</p>

            <form method="POST" action="/room" x-data="roomForm()">
                <div class="form-group">
                    <label for="name">Room Name</label>
                    <input type="text" id="name" name="name" required/>
                </div>

                <div class="form-group">
                    <label>Pointing Method</label>
                    <label>
                        <input type="radio" name="pointingMethod" value="fibonacci"
                               x-model="pointingMethod" checked/>
                        Fibonacci (1, 2, 3, 5, 8, 13, 21)
                    </label>
                    <label>
                        <input type="radio" name="pointingMethod" value="custom"
                               x-model="pointingMethod"/>
                        Custom Values
                    </label>
                </div>

                <div class="form-group" x-show="pointingMethod === 'custom'">
                    <label>Custom Values (comma-separated)</label>
                    <input type="text" name="customValues"
                           placeholder="XS, S, M, L, XL"/>
                </div>

                <button type="submit">Create Room</button>
            </form>
        </div>
    }
}
```

**`web/templates/room.templ`**:
```templ
package templates

import "github.com/yourusername/planning-poker/internal/models"

templ Room(room *models.Room, participant *models.Participant) {
    @Base(room.Name) {
        <div class="room-page" hx-ext="ws" ws-connect={"/ws/" + room.ID}>

            if participant == nil {
                @JoinModal(room.ID)
            }

            <header>
                <h1>{room.Name}</h1>
                @ShareControls(room.ID)
            </header>

            <div id="participants">
                @ParticipantGrid(room.Participants, room.State, room.Votes)
            </div>

            if participant != nil && participant.Role == models.RoleVoter {
                <div id="voting-cards">
                    if room.PointingMethod == "fibonacci" {
                        @VotingCards([]string{"1", "2", "3", "5", "8", "13", "21", "?", "☕"})
                    } else {
                        @VotingCards(room.CustomValues)
                    }
                </div>

                @Controls()
            }
        </div>
    }
}
```

**`web/templates/components/join_modal.templ`**:
```templ
package templates

templ JoinModal(roomID string) {
    <div class="modal" x-data="{ show: true }">
        <div class="modal-content" x-show="show">
            <h2>Join Room</h2>
            <form hx-post={"/room/" + roomID + "/join"} hx-swap="outerHTML">
                <div class="form-group">
                    <label for="name">Your Name</label>
                    <input type="text" id="name" name="name" required/>
                </div>

                <div class="form-group">
                    <label>Role</label>
                    <label>
                        <input type="radio" name="role" value="voter" checked/>
                        Voter
                    </label>
                    <label>
                        <input type="radio" name="role" value="spectator"/>
                        Spectator
                    </label>
                </div>

                <button type="submit">Join</button>
            </form>
        </div>
    </div>
}
```

**`web/templates/components/participant_grid.templ`**:
```templ
package templates

import "github.com/yourusername/planning-poker/internal/models"

templ ParticipantGrid(participants map[string]*models.Participant, state models.RoomState, votes map[string]string) {
    <div class="participant-grid">
        for _, p := range participants {
            if p.Role == models.RoleVoter {
                <div class="participant-card">
                    <div class="participant-avatar">
                        {p.Name[0:1]}
                    </div>
                    <div class="participant-name">{p.Name}</div>
                    <div class="participant-vote">
                        if vote, hasVoted := votes[p.ID]; hasVoted {
                            if state == models.StateRevealed {
                                <span class="vote-revealed">{vote}</span>
                            } else {
                                <span class="vote-hidden">✓</span>
                            }
                        } else {
                            <span class="vote-pending">⏱</span>
                        }
                    </div>
                </div>
            }
        }
    </div>

    <div class="spectators">
        <h3>Spectators</h3>
        for _, p := range participants {
            if p.Role == models.RoleSpectator {
                <span class="spectator">{p.Name}</span>
            }
        }
    </div>
}
```

**`web/templates/components/voting_cards.templ`**:
```templ
package templates

templ VotingCards(values []string) {
    <div class="voting-cards" x-data="cardSelector()">
        for _, value := range values {
            <button class="card"
                    :class="{ 'selected': selected === '{value}' }"
                    @click="selectCard('{value}')">
                {value}
            </button>
        }
    </div>
}
```

**`web/templates/components/controls.templ`**:
```templ
package templates

templ Controls() {
    <div class="controls">
        <button ws-send='{"type":"reveal"}'>Reveal Votes</button>
        <button ws-send='{"type":"reset"}'>Reset</button>
    </div>
}
```

**`web/templates/components/share.templ`**:
```templ
package templates

templ ShareControls(roomID string) {
    <div class="share-controls" x-data="roomSharing()">
        <button @click="copyUrl()">
            <span x-show="!copied">Copy URL</span>
            <span x-show="copied">Copied!</span>
        </button>
        <button @click="showQR = !showQR">QR Code</button>

        <div x-show="showQR" class="qr-code">
            <!-- QR code generation in Phase 6 -->
        </div>
    </div>
}
```

**Helper function for rendering**:
```go
// web/templates/render.go
package templates

import (
    "net/http"
    "github.com/a-h/templ"
)

func Render(w http.ResponseWriter, r *http.Request, component templ.Component) error {
    return component.Render(r.Context(), w)
}
```

**Generate templates**:
```bash
templ generate
```

**Deliverables**:
- Base template with script includes
- Home page template
- Room interface template
- Reusable component templates
- Template rendering helper

---

### Phase 6: Frontend Interactivity (Week 2-3, Days 13-15)

**Goal**: Add Alpine.js components and styling

**`web/static/js/alpine-components.js`**:
```javascript
// Room creation form
Alpine.data('roomForm', () => ({
    pointingMethod: 'fibonacci',
}));

// Card selector
Alpine.data('cardSelector', () => ({
    selected: null,

    selectCard(value) {
        this.selected = value;

        // Send vote via WebSocket
        const message = JSON.stringify({
            type: 'vote',
            value: value
        });

        // htmx-ext-ws handles sending
        const wsElement = document.querySelector('[ws-connect]');
        if (wsElement && wsElement.send) {
            wsElement.send(message);
        }
    }
}));

// Room sharing
Alpine.data('roomSharing', () => ({
    showQR: false,
    copied: false,

    async copyUrl() {
        try {
            await navigator.clipboard.writeText(window.location.href);
            this.copied = true;
            setTimeout(() => {
                this.copied = false;
            }, 2000);
        } catch (err) {
            console.error('Failed to copy:', err);
        }
    },

    generateQR() {
        // QR code generation using qrcodejs or similar
        // Will implement in Phase 9
    }
}));
```

**`web/static/css/styles.css`** (Basic styling):
```css
* {
    margin: 0;
    padding: 0;
    box-sizing: border-box;
}

body {
    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
    background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
    min-height: 100vh;
    display: flex;
    align-items: center;
    justify-content: center;
}

.container {
    max-width: 1200px;
    margin: 0 auto;
    padding: 2rem;
}

/* Home page */
.home-page {
    background: white;
    padding: 3rem;
    border-radius: 1rem;
    box-shadow: 0 10px 40px rgba(0, 0, 0, 0.1);
    max-width: 500px;
}

.home-page h1 {
    font-size: 2.5rem;
    margin-bottom: 0.5rem;
    color: #333;
}

.form-group {
    margin-bottom: 1.5rem;
}

.form-group label {
    display: block;
    margin-bottom: 0.5rem;
    font-weight: 500;
    color: #555;
}

.form-group input[type="text"] {
    width: 100%;
    padding: 0.75rem;
    border: 2px solid #e0e0e0;
    border-radius: 0.5rem;
    font-size: 1rem;
}

button {
    background: #667eea;
    color: white;
    border: none;
    padding: 0.75rem 2rem;
    border-radius: 0.5rem;
    font-size: 1rem;
    cursor: pointer;
    transition: background 0.2s;
}

button:hover {
    background: #5568d3;
}

/* Room page */
.room-page {
    background: white;
    padding: 2rem;
    border-radius: 1rem;
    box-shadow: 0 10px 40px rgba(0, 0, 0, 0.1);
}

/* Participant grid */
.participant-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
    gap: 1.5rem;
    margin: 2rem 0;
}

.participant-card {
    text-align: center;
    padding: 1.5rem;
    border: 2px solid #e0e0e0;
    border-radius: 0.5rem;
}

.participant-avatar {
    width: 80px;
    height: 80px;
    margin: 0 auto 1rem;
    background: #667eea;
    color: white;
    border-radius: 50%;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 2rem;
    font-weight: bold;
}

.vote-revealed {
    font-size: 1.5rem;
    font-weight: bold;
    color: #667eea;
}

.vote-hidden {
    font-size: 1.5rem;
    color: #4caf50;
}

.vote-pending {
    font-size: 1.5rem;
    color: #999;
}

/* Voting cards */
.voting-cards {
    display: flex;
    gap: 1rem;
    justify-content: center;
    margin: 2rem 0;
}

.card {
    width: 80px;
    height: 120px;
    border: 3px solid #667eea;
    background: white;
    border-radius: 0.5rem;
    font-size: 2rem;
    font-weight: bold;
    cursor: pointer;
    transition: all 0.2s;
}

.card:hover {
    transform: translateY(-5px);
    box-shadow: 0 5px 15px rgba(102, 126, 234, 0.3);
}

.card.selected {
    background: #667eea;
    color: white;
}

/* Controls */
.controls {
    display: flex;
    gap: 1rem;
    justify-content: center;
    margin-top: 2rem;
}

/* Modal */
.modal {
    position: fixed;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    background: rgba(0, 0, 0, 0.5);
    display: flex;
    align-items: center;
    justify-content: center;
}

.modal-content {
    background: white;
    padding: 2rem;
    border-radius: 1rem;
    max-width: 400px;
    width: 100%;
}

/* Share controls */
.share-controls {
    display: flex;
    gap: 0.5rem;
}

.qr-code {
    margin-top: 1rem;
}
```

**Deliverables**:
- Alpine.js components for interactivity
- CSS styling for all components
- Responsive design
- Card selection state management

---

### Phase 7: WebSocket Message Handlers (Week 3, Days 16-18)

**Goal**: Implement full WebSocket message handling

**Update `internal/handlers/ws.go`**:
```go
func (h *WSHandler) handleMessage(roomID string, msg *models.WSMessage) {
    room, err := h.roomManager.GetRoom(roomID)
    if err != nil {
        return
    }

    switch msg.Type {
    case models.MsgTypeVote:
        h.handleVote(room, msg)
    case models.MsgTypeReveal:
        h.handleReveal(room)
    case models.MsgTypeReset:
        h.handleReset(room)
    }
}

func (h *WSHandler) handleVote(room *models.Room, msg *models.WSMessage) {
    payload := msg.Payload.(map[string]interface{})
    participantID := payload["participantId"].(string)
    value := payload["value"].(string)

    // Cast vote
    room.CastVote(participantID, value)

    // Broadcast vote cast notification (without revealing value)
    h.hub.BroadcastToRoom(room.ID, &models.WSMessage{
        Type: models.MsgTypeVoteCast,
        Payload: map[string]interface{}{
            "participantId": participantID,
            "hasVoted":      true,
        },
    })
}

func (h *WSHandler) handleReveal(room *models.Room) {
    room.RevealVotes()

    // Prepare votes for broadcast
    room.mu.RLock()
    votes := make([]map[string]interface{}, 0)
    for participantID, value := range room.Votes {
        participant := room.Participants[participantID]
        votes = append(votes, map[string]interface{}{
            "participantId":   participantID,
            "participantName": participant.Name,
            "value":           value,
        })
    }
    room.mu.RUnlock()

    // Calculate stats
    stats := room.GetVoteStats()

    // Broadcast revealed votes
    h.hub.BroadcastToRoom(room.ID, &models.WSMessage{
        Type: models.MsgTypeVotesRevealed,
        Payload: map[string]interface{}{
            "votes": votes,
            "stats": stats,
        },
    })
}

func (h *WSHandler) handleReset(room *models.Room) {
    room.ResetVoting()

    // Broadcast reset notification
    h.hub.BroadcastToRoom(room.ID, &models.WSMessage{
        Type: models.MsgTypeRoomReset,
    })
}
```

**Deliverables**:
- Vote message handler
- Reveal message handler
- Reset message handler
- Broadcast notifications for all actions

---

### Phase 8: Session Management (Week 3, Days 19-20)

**Goal**: Implement cookie-based participant sessions

**`internal/services/session.go`**:
```go
package services

import (
    "net/http"
    "strings"
    "github.com/google/uuid"
    "github.com/yourusername/planning-poker/internal/models"
)

type SessionService struct{}

func NewSessionService() *SessionService {
    return &SessionService{}
}

func (s *SessionService) GetParticipantID(r *http.Request, roomID string) (string, bool) {
    cookie, err := r.Cookie("planning_poker_participant")
    if err != nil {
        return "", false
    }

    parts := strings.Split(cookie.Value, "|")
    if len(parts) != 2 {
        return "", false
    }

    participantID, cookieRoomID := parts[0], parts[1]
    if cookieRoomID != roomID {
        return "", false
    }

    return participantID, true
}

func (s *SessionService) SetParticipantCookie(w http.ResponseWriter, roomID, participantID string) {
    cookie := &http.Cookie{
        Name:     "planning_poker_participant",
        Value:    participantID + "|" + roomID,
        Path:     "/room/" + roomID,
        MaxAge:   86400, // 24 hours
        HttpOnly: true,
        SameSite: http.SameSiteLaxMode,
    }
    http.SetCookie(w, cookie)
}

func (s *SessionService) CreateParticipant(name string, role models.ParticipantRole) *models.Participant {
    return models.NewParticipant(uuid.New().String(), name, role)
}
```

**Update `internal/handlers/join.go`**:
```go
package handlers

import (
    "net/http"
    "github.com/pocketbase/pocketbase/core"
    "github.com/yourusername/planning-poker/internal/models"
    "github.com/yourusername/planning-poker/internal/services"
)

type JoinHandler struct {
    roomManager    *services.RoomManager
    sessionService *services.SessionService
    hub            *services.Hub
}

func NewJoinHandler(rm *services.RoomManager, ss *services.SessionService, hub *services.Hub) *JoinHandler {
    return &JoinHandler{
        roomManager:    rm,
        sessionService: ss,
        hub:            hub,
    }
}

func (h *JoinHandler) JoinRoom(re *core.RequestEvent) error {
    roomID := re.Request.PathValue("id")
    name := re.Request.FormValue("name")
    roleStr := re.Request.FormValue("role")

    room, err := h.roomManager.GetRoom(roomID)
    if err != nil {
        return re.Redirect(http.StatusSeeOther, "/")
    }

    // Validate role
    role := models.ParticipantRole(roleStr)
    if role != models.RoleVoter && role != models.RoleSpectator {
        role = models.RoleVoter
    }

    // Create participant
    participant := h.sessionService.CreateParticipant(name, role)
    room.AddParticipant(participant)

    // Set cookie
    h.sessionService.SetParticipantCookie(re.Response, roomID, participant.ID)

    // Broadcast participant joined
    h.hub.BroadcastToRoom(roomID, &models.WSMessage{
        Type: models.MsgTypeParticipantJoined,
        Payload: map[string]interface{}{
            "id":        participant.ID,
            "name":      participant.Name,
            "role":      participant.Role,
            "connected": true,
        },
    })

    // Reload page to show room interface
    return re.Redirect(http.StatusSeeOther, "/room/"+roomID)
}
```

**Update `main.go`** to include session service:
```go
sessionService := services.NewSessionService()
joinHandler := handlers.NewJoinHandler(roomManager, sessionService, hub)

se.Router.POST("/room/:id/join", joinHandler.JoinRoom)
```

**Deliverables**:
- Cookie-based session management
- Participant creation and persistence
- Join room handler with cookie setting
- Participant joined broadcast

---

### Phase 9: Room Features (Week 3-4, Days 21-25)

**Goal**: Implement pointing methods and sharing features

**Update `internal/models/room.go`** with stats calculation:
```go
func (r *Room) GetVoteStats() map[string]interface{} {
    r.mu.RLock()
    defer r.mu.RUnlock()

    if len(r.Votes) == 0 {
        return map[string]interface{}{
            "total": 0,
        }
    }

    // Parse numeric votes
    var numericVotes []float64
    for _, vote := range r.Votes {
        // Try to parse as number
        var val float64
        _, err := fmt.Sscanf(vote, "%f", &val)
        if err == nil {
            numericVotes = append(numericVotes, val)
        }
    }

    stats := map[string]interface{}{
        "total": len(r.Votes),
    }

    if len(numericVotes) == 0 {
        return stats
    }

    // Calculate average
    sum := 0.0
    min := numericVotes[0]
    max := numericVotes[0]

    for _, val := range numericVotes {
        sum += val
        if val < min {
            min = val
        }
        if val > max {
            max = val
        }
    }

    stats["average"] = sum / float64(len(numericVotes))
    stats["min"] = min
    stats["max"] = max

    return stats
}
```

**Add QR code generation**:

Install QR code library:
```bash
go get github.com/skip2/go-qrcode
```

**Create `internal/handlers/qr.go`**:
```go
package handlers

import (
    "github.com/pocketbase/pocketbase/core"
    "github.com/skip2/go-qrcode"
)

func GenerateQR(re *core.RequestEvent) error {
    roomID := re.Request.PathValue("id")
    url := "https://yourapp.com/room/" + roomID // Use actual domain

    png, err := qrcode.Encode(url, qrcode.Medium, 256)
    if err != nil {
        return err
    }

    re.Response.Header().Set("Content-Type", "image/png")
    _, err = re.Response.Write(png)
    return err
}
```

**Update `web/templates/components/share.templ`**:
```templ
templ ShareControls(roomID string) {
    <div class="share-controls" x-data="roomSharing()">
        <button @click="copyUrl()">
            <span x-show="!copied">📋 Copy URL</span>
            <span x-show="copied">✓ Copied!</span>
        </button>
        <button @click="showQR = !showQR">📱 QR Code</button>

        <div x-show="showQR" class="qr-code">
            <img src={"/room/" + roomID + "/qr"} alt="QR Code"/>
        </div>
    </div>
}
```

**Register QR route in `main.go`**:
```go
se.Router.GET("/room/:id/qr", handlers.GenerateQR)
```

**Deliverables**:
- Vote statistics calculation (average, min, max)
- QR code generation endpoint
- URL copy functionality
- Complete sharing features

---

### Phase 10: Polish & Testing (Week 4, Days 26-28)

**Goal**: Error handling, validation, testing, and improvements

**Add validation** in `internal/utils/validator.go`:
```go
package utils

import (
    "errors"
    "strings"
)

func ValidateRoomName(name string) error {
    name = strings.TrimSpace(name)
    if name == "" {
        return errors.New("room name is required")
    }
    if len(name) > 50 {
        return errors.New("room name must be 50 characters or less")
    }
    return nil
}

func ValidateParticipantName(name string) error {
    name = strings.TrimSpace(name)
    if name == "" {
        return errors.New("name is required")
    }
    if len(name) > 30 {
        return errors.New("name must be 30 characters or less")
    }
    return nil
}

func ValidateCustomValues(values []string) error {
    if len(values) == 0 {
        return errors.New("custom values are required")
    }
    if len(values) > 12 {
        return errors.New("maximum 12 custom values allowed")
    }
    for _, val := range values {
        if len(val) > 10 {
            return errors.New("custom values must be 10 characters or less")
        }
    }
    return nil
}
```

**Add error responses** in `internal/utils/response.go`:
```go
package utils

import (
    "github.com/pocketbase/pocketbase/core"
)

func ErrorResponse(re *core.RequestEvent, status int, message string) error {
    return re.JSON(status, map[string]string{
        "error": message,
    })
}

func SuccessResponse(re *core.RequestEvent, data interface{}) error {
    return re.JSON(200, data)
}
```

**Update handlers with validation**:
```go
// In room.go CreateRoom handler
if err := utils.ValidateRoomName(name); err != nil {
    return utils.ErrorResponse(re, 400, err.Error())
}

if pointingMethod == "custom" {
    if err := utils.ValidateCustomValues(customValues); err != nil {
        return utils.ErrorResponse(re, 400, err.Error())
    }
}
```

**Add graceful shutdown**:
```go
// In main.go
func main() {
    app := pocketbase.New()

    // ... setup code ...

    // Graceful shutdown
    app.OnTerminate().BindFunc(func(e *core.TerminateEvent) error {
        // Close all WebSocket connections
        // Cleanup resources
        return nil
    })

    if err := app.Start(); err != nil {
        log.Fatal(err)
    }
}
```

**Testing checklist**:
- [ ] Create room with Fibonacci
- [ ] Create room with custom values
- [ ] Join as voter
- [ ] Join as spectator
- [ ] Cast votes
- [ ] Reveal votes
- [ ] Reset round
- [ ] Multiple participants
- [ ] WebSocket reconnection
- [ ] Cookie persistence
- [ ] QR code generation
- [ ] URL copying
- [ ] Room cleanup (TTL)

**Deliverables**:
- Input validation
- Error handling
- Graceful shutdown
- Testing and bug fixes
- Performance optimization

---

## Configuration & Deployment

### Dependencies (go.mod)

```go
module github.com/yourusername/planning-poker

go 1.23

require (
    github.com/pocketbase/pocketbase v0.30.0  // Includes Echo v4, SQLite, auth, admin UI
    github.com/coder/websocket v1.8.14
    github.com/a-h/templ v0.3.943
    github.com/google/uuid v1.6.0
    github.com/skip2/go-qrcode v0.0.0-20200617195104-da1b6568686e
)
```

**PocketBase Bundled Features**:
- Echo v4 router (se.Router is *echo.Echo)
- SQLite database with migrations
- Built-in authentication system
- Admin dashboard UI at /_/
- File storage system
- Real-time subscriptions (alternative to our WebSocket hub if needed)
- API hooks and events system

### Dockerfile

```dockerfile
# Build stage
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Install templ CLI
RUN go install github.com/a-h/templ/cmd/templ@latest

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Generate templ templates
RUN templ generate

# Build binary
RUN CGO_ENABLED=0 GOOS=linux go build -o planning-poker .

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy binary
COPY --from=builder /app/planning-poker .

# Copy static files
COPY --from=builder /app/web/static ./web/static

EXPOSE 8090

CMD ["./planning-poker", "serve", "--http=0.0.0.0:8090"]
```

### Build and Run

```bash
# Development (with live reload)
air

# Production build
docker build -t planning-poker .
docker run -p 8090:8090 planning-poker
```

### Environment Configuration

**`internal/config/config.go`**:
```go
package config

import (
    "os"
    "strconv"
    "time"
)

type Config struct {
    Host            string
    Port            int
    Domain          string
    RoomTTL         time.Duration
    MaxParticipants int
    MaxRooms        int
}

func Load() *Config {
    return &Config{
        Host:            getEnv("HOST", "0.0.0.0"),
        Port:            getEnvInt("PORT", 8090),
        Domain:          getEnv("DOMAIN", "http://localhost:8090"),
        RoomTTL:         24 * time.Hour,
        MaxParticipants: 50,
        MaxRooms:        1000,
    }
}

func getEnv(key, defaultValue string) string {
    if value := os.Getenv(key); value != "" {
        return value
    }
    return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
    if value := os.Getenv(key); value != "" {
        if i, err := strconv.Atoi(value); err == nil {
            return i
        }
    }
    return defaultValue
}
```

---

## Best Practices

### Concurrency Safety

**Always use mutexes for shared state**:
```go
// Read lock for read operations
room.mu.RLock()
participants := room.Participants
room.mu.RUnlock()

// Write lock for write operations
room.mu.Lock()
room.Votes[participantID] = value
room.mu.Unlock()
```

**Use channels for communication**:
```go
// Hub uses channels for thread-safe message passing
h.broadcast <- &BroadcastMessage{...}
```

### Error Handling

**Validate all inputs**:
```go
if err := utils.ValidateRoomName(name); err != nil {
    return utils.ErrorResponse(re, 400, err.Error())
}
```

**Handle WebSocket errors gracefully**:
```go
_, data, err := conn.Read(ctx)
if err != nil {
    // Log error and close connection
    log.Printf("WebSocket read error: %v", err)
    break
}
```

### Security

**Input validation**:
- Sanitize room names and participant names
- Limit input lengths
- Validate vote values against allowed values

**WebSocket origin checking**:
```go
&websocket.AcceptOptions{
    OriginPatterns: []string{"yourdomain.com"},
}
```

**XSS protection**:
- Templ automatically escapes HTML
- Never use `templ.Raw()` with user input

### Performance

**Room cleanup**:
- Background goroutine runs hourly
- Removes rooms inactive for 24+ hours
- Prevents memory leaks

**Connection limits**:
```go
if len(room.Participants) >= config.MaxParticipants {
    return errors.New("room is full")
}
```

**Message size limits**:
```go
conn.SetReadLimit(32768) // 32KB
```

---

## Summary

This implementation plan provides a complete, step-by-step guide to building a Planning Poker SaaS application using:

- **PocketBase** as the foundation
- **coder/websocket** for real-time communication
- **Templ** for type-safe HTML generation
- **htmx + Alpine.js** for minimal JavaScript interactivity
- **In-memory state** for simplicity
- **Single container** deployment

### Key Features Delivered:
✅ Public rooms (no authentication)
✅ Real-time voting with WebSocket
✅ Fibonacci and custom pointing methods
✅ Modern UI with participant grid
✅ QR code and URL sharing
✅ Anonymous participation
✅ Voter and spectator roles
✅ Vote statistics (average, min, max)
✅ Session persistence with cookies
✅ Room cleanup (24h TTL)

### Timeline: ~4 weeks
- Week 1: Foundation, models, WebSocket infrastructure
- Week 2: Handlers, templates, frontend
- Week 3: Message handling, sessions, features
- Week 4: Polish, testing, deployment

---

**Next Steps:**
1. Follow phases sequentially
2. Test each phase before moving forward
3. Refer to code examples for implementation details
4. Adjust styling and UX based on preferences

Good luck building your Planning Poker SaaS! 🚀
