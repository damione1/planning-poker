# Multi-stage build for production
FROM golang:1.25-alpine AS builder

# Install build dependencies including Node.js for Tailwind
RUN apk add --no-cache git gcc musl-dev nodejs npm

# Install templ
RUN go install github.com/a-h/templ/cmd/templ@latest

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy package.json and install npm dependencies
COPY package.json package-lock.json* ./
RUN npm install

# Copy source code
COPY . .

# Build Tailwind CSS
RUN npm run build:css

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
