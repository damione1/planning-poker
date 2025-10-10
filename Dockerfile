# Multi-stage build for production
FROM golang:1.25-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git gcc musl-dev

# Install templ
RUN go install github.com/a-h/templ/cmd/templ@latest

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Generate templ templates
RUN templ generate

# Build the application
RUN CGO_ENABLED=1 GOOS=linux go build -o /app/main .

# Production image
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/main .
COPY --from=builder /app/web ./web

# Create data directory for SQLite
RUN mkdir -p /app/pb_data

EXPOSE 8090

CMD ["./main", "serve", "--http=0.0.0.0:8090"]
