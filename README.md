# Planning Poker

[![Go Version](https://img.shields.io/badge/Go-1.24.3-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![PocketBase](https://img.shields.io/badge/PocketBase-0.30-B8DBE4?style=flat)](https://pocketbase.io/)
[![Go Report Card](https://goreportcard.com/badge/github.com/damione1/planning-poker)](https://goreportcard.com/report/github.com/damione1/planning-poker)
[![GitHub Release](https://img.shields.io/github/v/release/damione1/planning-poker)](https://github.com/damione1/planning-poker/releases)
[![GitHub Issues](https://img.shields.io/github/issues/damione1/planning-poker)](https://github.com/damione1/planning-poker/issues)

Real-time Planning Poker application built with Go, PocketBase, htmx, and Alpine.js. Features WebSocket-based real-time collaboration, persistent state management, and flexible access control.

## Features

- **Real-time Collaboration**: WebSocket-based instant updates across all participants
- **Anonymous Access**: No authentication required - create and join rooms instantly
- **Flexible Voting**: Support for Fibonacci, Modified Fibonacci, and custom value sets
- **Role-Based Permissions**: Voter and Spectator roles with configurable access controls
- **Persistent State**: SQLite database with automatic migrations and 24-hour room expiration
- **Responsive UI**: Clean, mobile-friendly interface built with htmx and Alpine.js
- **Vote Statistics**: Real-time calculation of averages and value distribution

## Quick Start

### Development with Docker

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

Application runs at http://localhost:8090

### Manual Build

```bash
# Build binary
go build -o main .

# Run with automatic migrations
./main serve --http=0.0.0.0:8090

# Access application
open http://localhost:8090

# Access admin UI (database inspection)
open http://localhost:8090/_/
```

## How It Works

### Architecture

**Backend**:
- **PocketBase v0.30**: All-in-one backend with Echo router, SQLite, and admin UI
- **WebSocket Hub**: Manages real-time connections and broadcasts with rate limiting
- **State Management**: Room state derived from current voting round
- **Automatic Cleanup**: Background job removes expired rooms hourly

**Frontend**:
- **htmx 2.0**: Declarative AJAX and WebSocket handling
- **Alpine.js 3.14**: Reactive UI components and state management
- **Templ**: Type-safe Go templating engine

**Data Model**:
```
rooms → rounds → votes
   ↓       ↓
   └→ participants
```

- `rooms`: Room configuration and metadata (24h TTL)
- `rounds`: Voting rounds with state (voting/revealed/completed)
- `participants`: Users with roles (voter/spectator) and connection status
- `votes`: Individual votes linked to participants and rounds

### WebSocket Protocol

**Client → Server**:
- `vote`: Cast or update a vote
- `reveal`: Transition round to revealed state (show all votes)
- `reset`: Clear votes and return to voting state
- `next_round`: Complete current round and start new one
- `update_name`: Change participant name
- `update_room_name`: Change room name (creator only)
- `update_config`: Update room permissions (creator only)

**Server → Client**:
- `room_state`: Complete state sync on connect/reconnect
- `participant_joined`: User joined the room
- `participant_left`: User left the room
- `vote_cast`: Vote recorded (value hidden)
- `vote_updated`: Vote changed in revealed state (value shown)
- `votes_revealed`: All votes revealed with statistics
- `room_reset`: Voting round reset
- `round_completed`: New round started
- `name_updated`: Participant name changed
- `room_name_updated`: Room name changed
- `config_updated`: Room permissions updated
- `room_expired`: Room has expired (actions blocked)

### Security Features

- **Origin Validation**: Configurable WebSocket origin allowlist
- **Rate Limiting**: 10 messages per second per connection
- **Input Sanitization**: All user inputs validated and sanitized
- **UUID Validation**: All IDs validated before database operations
- **Secure Cookies**: Session cookies with secure flag (production)
- **Message Validation**: WebSocket message type and payload validation

## Development

### Prerequisites

- Go 1.24.3+
- Docker & Docker Compose (for containerized development)
- Make

### Project Structure

```
planning-poker/
├── main.go                    # Application entry point
├── pb_migrations/             # Database migrations
├── internal/
│   ├── models/               # Data models
│   ├── services/             # Business logic
│   ├── handlers/             # HTTP/WebSocket handlers
│   └── security/             # Validation and security
├── web/
│   ├── templates/            # Templ templates
│   └── static/               # CSS and JavaScript
└── tests/                    # Integration tests
```

### Database Migrations

Migrations run automatically on startup (configurable via `Automigrate: true` in `main.go`).

**Manual Migration Control**:
```bash
# Check status
./main migrate collections

# Run migrations
./main migrate up

# Rollback last migration
./main migrate down 1
```

**Environment Variables**:
```bash
# Disable automigrate
AUTOMIGRATE=false ./main serve

# Set data directory
./main serve --dir=/app/pb_data

# Configure WebSocket origins
WS_ALLOWED_ORIGINS=localhost:*,example.com:* ./main serve
```

### Testing

```bash
# Run tests
make test

# Run specific test
go test ./tests -v -run TestRoomCreation
```

**Integration Test Coverage**:
- Room creation and expiration
- Participant joining and role management
- Vote casting and statistics
- Round lifecycle (reveal, reset, next round)
- WebSocket connection and reconnection
- Permissions and access control

## Deployment

### Docker Production

```bash
# Build image
docker build -f Dockerfile -t planning-poker .

# Run container
docker run -p 8090:8090 \
  -v pb_data:/app/pb_data \
  --restart unless-stopped \
  planning-poker
```

### Binary Deployment

```bash
# Build for Linux
GOOS=linux GOARCH=amd64 go build -o main .

# Deploy and run
./main serve --http=0.0.0.0:8090 --dir=/var/lib/pocketbase
```

### Environment Configuration

**Development**:
```env
DEV_MODE=true
WS_ALLOWED_ORIGINS=localhost:*,127.0.0.1:*
```

**Production**:
```env
DEV_MODE=false
WS_ALLOWED_ORIGINS=yourdomain.com:*
AUTOMIGRATE=true
```

## License

MIT
