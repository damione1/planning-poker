# Planning Poker SaaS

Real-time Planning Poker application built with PocketBase, Go, htmx, and Alpine.js.

## 🎯 Current Status

**Phases Completed:**
- ✅ Phase 1: Project setup with Go modules and PocketBase
- ✅ Phase 2: Core data models (Room, Participant, Message)
- ✅ Phase 3: WebSocket Hub for real-time broadcasting
- ✅ Phase 4: HTTP handlers (Home, Room Creation, WebSocket)
- ✅ Phase 5: Templ templates (Base, Home, Room, Components)
- ✅ Phase 6: Frontend (Alpine.js components + Complete CSS)

**Remaining Phases:**
- ⏳ Phase 7: WebSocket message handlers (vote, reveal, reset)
- ⏳ Phase 8: Session management (cookie-based participant identification)
- ⏳ Phase 9: Room features (QR codes, vote statistics)
- ⏳ Phase 10: Polish & testing

## 🚀 Quick Start

```bash
# Development mode with live reload
make dev

# Or build and run manually
make build
./tmp/main serve --http=127.0.0.1:8090

# Visit
open http://localhost:8090
```

## 🏗️ Architecture

### Technology Stack
- **Backend**: PocketBase v0.30 (includes Echo router, SQLite, auth, admin UI)
- **WebSocket**: github.com/coder/websocket v1.8.14
- **Templating**: github.com/a-h/templ v0.3.943 ✅
- **Frontend**: htmx 2.0 + Alpine.js 3.14 ✅
- **Language**: Go 1.24.3

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
│       ├── home.go                  # Landing page with template rendering
│       ├── room.go                  # Room creation/view with template rendering
│       └── ws.go                    # WebSocket upgrade
└── web/                             # Frontend ✅
    ├── templates/                   # Templ templates ✅
    │   ├── base.templ               # Base HTML layout
    │   ├── home.templ               # Landing page
    │   ├── room.templ               # Room interface
    │   ├── join_modal.templ         # Participant join modal
    │   ├── participant_grid.templ   # Voter display grid
    │   ├── voting_cards.templ       # Card selector
    │   ├── controls.templ           # Reveal/Reset buttons
    │   ├── share.templ              # Share controls
    │   └── render.go                # Render helper
    └── static/                      # CSS/JS ✅
        ├── css/
        │   └── styles.css           # Complete styling system
        └── js/
            └── alpine-components.js # Alpine.js data components
```

## 🔧 Development

```bash
# Install development tools (templ + air)
make install-tools

# Run development server with live reload
make dev

# Other useful commands
make help          # Show all available commands
make build         # Build production binary
make clean         # Clean build artifacts
make templ-generate # Generate templ templates only
make tidy          # Tidy go modules
make fmt           # Format code
```

## 📝 Implementation Plan

See [IMPLEMENTATION_PLAN.md](./IMPLEMENTATION_PLAN.md) for complete architectural guide.

## 🎯 Features

- ✅ In-memory room state management
- ✅ WebSocket infrastructure
- ✅ Server-side rendered templates (templ)
- ✅ Responsive UI with participant grid
- ✅ Fibonacci & custom pointing methods (UI ready)
- ✅ Alpine.js components (card selection, room sharing)
- ✅ Complete CSS styling system
- ⏳ Real-time voting (Phase 7)
- ⏳ QR code sharing (Phase 9)
- ⏳ Vote statistics (Phase 9)
- ⏳ Session persistence (Phase 8)

## 📄 License

MIT
