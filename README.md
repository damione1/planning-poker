# Planning Poker SaaS

Real-time Planning Poker application built with PocketBase, Go, htmx, and Alpine.js.

## 🎯 Current Status

**Phases Completed:**
- ✅ Phase 1: Project setup with Go modules and PocketBase
- ✅ Phase 2: Core data models (Room, Participant, Message)
- ✅ Phase 3: WebSocket Hub for real-time broadcasting
- ✅ Phase 4: HTTP handlers (Home, Room Creation, WebSocket)

**Remaining Phases:**
- ⏳ Phase 5: Templ templates
- ⏳ Phase 6: Frontend (Alpine.js + htmx)
- ⏳ Phase 7: WebSocket message handlers
- ⏳ Phase 8: Session management
- ⏳ Phase 9: Room features (QR codes, stats)
- ⏳ Phase 10: Polish & testing

## 🚀 Quick Start

```bash
# Build
go build -o main .

# Run
./main serve --http=127.0.0.1:8090

# Visit
open http://localhost:8090
```

## 🏗️ Architecture

### Technology Stack
- **Backend**: PocketBase v0.30 (includes Echo router, SQLite, auth, admin UI)
- **WebSocket**: github.com/coder/websocket v1.8.14
- **Templating**: github.com/a-h/templ (planned)
- **Frontend**: htmx 2.0 + Alpine.js 3.14 (planned)
- **Language**: Go 1.23+

### Key Design Decisions

**✅ PocketBase Benefits Confirmed:**
1. **No separate Echo import needed** - PocketBase bundles Echo v4
2. **Built-in admin dashboard** at `/_/`
3. **SQLite included** (unused in MVP - using in-memory state)
4. **Authentication system** ready for future use
5. **File storage** available for future features

**Current Implementation:**
- In-memory room state (no database in MVP)
- Concurrent-safe operations with sync.RWMutex
- WebSocket hub with broadcast pattern
- 24-hour room TTL with automatic cleanup

## 📁 Project Structure

```
planning-poker/
├── main.go                          # PocketBase app with route registration
├── internal/
│   ├── models/                      # Data models
│   │   ├── room.go                  # Room with concurrent-safe methods
│   │   ├── participant.go           # Participant with roles
│   │   └── message.go               # WebSocket message types
│   ├── services/                    # Business logic
│   │   ├── room_manager.go          # In-memory room CRUD + cleanup
│   │   └── hub.go                   # WebSocket broadcast hub
│   └── handlers/                    # HTTP handlers
│       ├── home.go                  # Landing page
│       ├── room.go                  # Room creation/view
│       └── ws.go                    # WebSocket upgrade
└── web/                             # Frontend (to be implemented)
    ├── templates/                   # Templ templates
    └── static/                      # CSS/JS/images
```

## 🔧 Development

```bash
# Install templ generator
go install github.com/a-h/templ/cmd/templ@latest

# Install Air for live reload (optional)
go install github.com/air-verse/air@latest

# Run with live reload
air
```

## 📝 Implementation Plan

See [IMPLEMENTATION_PLAN.md](./IMPLEMENTATION_PLAN.md) for complete architectural guide.

## 🎯 Features (Planned)

- ✅ In-memory room state management
- ✅ WebSocket infrastructure
- ⏳ Real-time voting
- ⏳ Fibonacci & custom pointing
- ⏳ Participant grid UI
- ⏳ QR code sharing
- ⏳ Vote statistics
- ⏳ Session persistence

## 📄 License

MIT
