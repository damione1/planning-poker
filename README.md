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
- ✅ Phase 6.1: Database persistence with PocketBase SQLite
- ✅ Phase 7: WebSocket message handlers (vote, reveal, reset)

**Remaining Phases:**

- ⏳ Phase 8: Session management enhancements
- ⏳ Phase 9: Room features (QR codes, vote statistics)
- ⏳ Phase 10: Polish & testing

## 🚀 Quick Start

### Local Development (Docker)

```bash
# Start development environment with live reload
make dev

# Or start in background
make docker-up

# View logs
make docker-logs

# Stop
make docker-down
```

### Manual Build & Run

```bash
# Build binary
go build -o main .

# Run with automatic migrations
./main serve --http=0.0.0.0:8090

# Access application
open http://localhost:8090

# Access admin UI (for database inspection)
open http://localhost:8090/_/
```

## 🗄️ Database Migrations

### Automatic Migrations (Default)

Migrations run **automatically** on application startup when `Automigrate: true` is configured:

```go
// main.go
migratecmd.MustRegister(app, app.RootCmd, migratecmd.Config{
    Automigrate: true, // ✅ Enabled by default
})
```

**On first startup**, the following collections are created:
- `rooms` - Room configuration and state
- `participants` - Participant tracking with roles
- `votes` - Vote history with round tracking

### Manual Migration Control

For production deployments or CI/CD pipelines, you may want manual control:

```bash
# Check migration status
./main migrate collections

# List pending migrations
./main migrate

# Run pending migrations
./main migrate up

# Rollback last migration
./main migrate down 1

# Create new migration (for development)
./main migrate create "add_new_field"
```

### Environment Variables

```bash
# Disable automigrate for manual control
AUTOMIGRATE=false ./main serve

# Set custom data directory (for persistent storage)
./main serve --dir=/app/pb_data

# Production example
./main serve --http=0.0.0.0:8090 --dir=/var/lib/pocketbase
```

### Production Deployment

**Option 1: Automigrate (Recommended for AWS Lightsail)**
```bash
# Migrations run on startup automatically
docker run -p 8090:8090 -v pb_data:/app/pb_data planning-poker
```

**Option 2: Manual Migration (CI/CD Pipeline)**
```bash
# Step 1: Run migrations before deployment
docker run planning-poker /app/main migrate up

# Step 2: Start application
docker run -p 8090:8090 planning-poker
```

**Terraform/IaC Example:**
```hcl
# user_data script for AWS Lightsail
#!/bin/bash
docker pull your-registry/planning-poker:latest
docker run -d \
  -p 8090:8090 \
  -v /opt/planning-poker/pb_data:/app/pb_data \
  --restart unless-stopped \
  your-registry/planning-poker:latest
# Migrations run automatically on container start
```

### Database Backup

```bash
# Backup SQLite database (before migrations)
cp pb_data/data.db pb_data/data.db.backup

# Or use PocketBase backup command
./main backup
```

### Verifying Migrations

```bash
# Check collections exist
./main migrate collections

# Access admin UI to inspect schema
open http://localhost:8090/_/

# Query database directly (for debugging)
sqlite3 pb_data/data.db "SELECT name FROM collections;"
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

- **Database Persistence**: SQLite via PocketBase (Phase 6.1 ✅ complete)
  - `rooms` collection with 24h TTL and automatic expiration cleanup
  - `participants` collection with session cookie tracking and connection status
  - `votes` collection with round-based history (ready for Phase 7)
  - All CRUD operations use PocketBase DAO layer
- **WebSocket Hub**: In-memory connection management with real-time broadcasting and participant tracking
- **Automatic Cleanup**: Background job removes expired rooms hourly
- **Anonymous Access**: No authentication required for room creation/joining
- **Session Management**: Cookie-based participant identification across reconnects

## 📁 Project Structure

```
planning-poker/
├── main.go                          # PocketBase app with route registration + migrations
├── pb_migrations/                   # Database migrations ✅
│   └── 1728561600_create_initial_schema.go  # Initial schema (rooms, participants, votes)
├── pb_data/                         # PocketBase data directory (gitignored)
│   ├── data.db                      # SQLite database
│   └── logs.db                      # Application logs
├── internal/
│   ├── models/                      # Data models
│   │   ├── room.go                  # Room with concurrent-safe methods
│   │   ├── participant.go           # Participant with roles
│   │   └── message.go               # WebSocket message types
│   ├── services/                    # Business logic
│   │   ├── room_manager.go          # Room management (transitioning to DB)
│   │   └── hub.go                   # WebSocket broadcast hub
│   └── handlers/                    # HTTP handlers
│       ├── home.go                  # Landing page with template rendering
│       ├── room.go                  # Room creation/view with template rendering
│       └── ws.go                    # WebSocket upgrade
├── web/                             # Frontend ✅
│   ├── templates/                   # Templ templates ✅
│   │   ├── base.templ               # Base HTML layout
│   │   ├── home.templ               # Landing page
│   │   ├── room.templ               # Room interface
│   │   ├── join_modal.templ         # Participant join modal
│   │   ├── participant_grid.templ   # Voter display grid
│   │   ├── voting_cards.templ       # Card selector
│   │   ├── controls.templ           # Reveal/Reset buttons
│   │   ├── share.templ              # Share controls
│   │   └── render.go                # Render helper
│   └── static/                      # CSS/JS ✅
│       ├── css/
│       │   └── styles.css           # Complete styling system
│       └── js/
│           └── alpine-components.js # Alpine.js data components
└── claudedocs/                      # Technical documentation
    ├── DATABASE_DESIGN.md           # Schema design & decisions
    └── PHASE_6.1_IMPLEMENTATION.md  # Migration progress tracker
```

## 🔧 Development

### Docker Commands

```bash
make dev           # Start dev environment with live reload
make dev-build     # Rebuild Docker images
make docker-up     # Start services in background
make docker-down   # Stop all services
make docker-logs   # Follow logs
make docker-clean  # Stop services and remove volumes
```

### Local Go Commands

```bash
go run . serve                # Start server (migrations run automatically)
go run . migrate collections  # Check migration status
go run . migrate up           # Run pending migrations
go build -o main .            # Build production binary
```

### Code Quality

```bash
make tidy          # Tidy go modules
make fmt           # Format code
go test ./...      # Run tests
```

## 📝 Implementation Plan

See [IMPLEMENTATION_PLAN.md](./IMPLEMENTATION_PLAN.md) for complete architectural guide.

## 🎯 Features

### ✅ Completed
- Database persistence (SQLite via PocketBase)
- Automatic migrations on startup
- WebSocket real-time infrastructure with message handlers
- Real-time voting with vote, reveal, and reset functionality
- Vote statistics calculation (total, average, value breakdown)
- Server-side rendered templates (templ)
- Responsive UI with participant grid
- Fibonacci & custom pointing methods (UI ready)
- Alpine.js components (card selection, room sharing)
- Complete CSS styling system
- Docker Compose for development and production
- Background cleanup job for expired rooms

### ⏳ In Progress
- Phase 8: Session management enhancements

### 📋 Planned
- Session persistence (Phase 8)
- QR code sharing (Phase 9)
- Vote statistics (Phase 9)
- Polish & testing (Phase 10)

## 📄 License

MIT
