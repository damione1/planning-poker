# Multi-stage build for production
FROM golang:1.25-alpine AS builder

# Install build dependencies including Node.js for Tailwind
RUN apk add --no-cache git nodejs npm

# Install templ
RUN go install github.com/a-h/templ/cmd/templ@latest

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy package.json and install npm dependencies
COPY package.json package-lock.json ./
RUN npm ci

# Copy source code
COPY . .

# Build frontend assets (Tailwind + htmx)
RUN npm run build

# Generate templ templates
RUN templ generate

# Build the application with version info
ARG VERSION=dev
ARG BUILD_TIME
ARG GIT_COMMIT

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w -X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME} -X main.GitCommit=${GIT_COMMIT}" \
    -o /app/planning-poker \
    .

# Production image
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata wget

# Create non-root user
RUN addgroup -g 1000 planning-poker && \
    adduser -D -u 1000 -G planning-poker planning-poker

WORKDIR /app

# Copy binary and web assets from builder
COPY --from=builder /app/planning-poker .
COPY --from=builder /app/web ./web

# Create data directory for PocketBase
RUN mkdir -p /app/pb_data && \
    chown -R planning-poker:planning-poker /app

# Switch to non-root user
USER planning-poker

EXPOSE 8090

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8090 || exit 1

CMD ["./planning-poker", "serve", "--http=0.0.0.0:8090"]
