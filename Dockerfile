# Multi-stage build for production
FROM golang:1.25-alpine AS builder

# Install build dependencies including Node.js for Tailwind
RUN apk add --no-cache git nodejs npm wget

WORKDIR /app

# Install templ binary directly (faster than go install for multi-arch builds)
ARG TARGETARCH
RUN TEMPL_VERSION=v0.3.819 && \
    case ${TARGETARCH} in \
        amd64) TEMPL_ARCH=x86_64 ;; \
        arm64) TEMPL_ARCH=arm64 ;; \
        *) echo "Unsupported architecture: ${TARGETARCH}" && exit 1 ;; \
    esac && \
    wget -q https://github.com/a-h/templ/releases/download/${TEMPL_VERSION}/templ_Linux_${TEMPL_ARCH}.tar.gz && \
    tar -xzf templ_Linux_${TEMPL_ARCH}.tar.gz -C /usr/local/bin templ && \
    rm templ_Linux_${TEMPL_ARCH}.tar.gz && \
    templ version

# Copy go mod files first for better layer caching
COPY go.mod go.sum ./

# Download Go modules with cache mount for faster subsequent builds
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

# Copy package.json and install npm dependencies with cache mount
COPY package.json package-lock.json ./
RUN --mount=type=cache,target=/root/.npm \
    npm ci

# Copy source code
COPY . .

# Build frontend assets (Tailwind + htmx) with cache mount
RUN --mount=type=cache,target=/root/.npm \
    npm run build

# Generate templ templates
RUN templ generate

# Build the application with version info and cache mounts
ARG VERSION=dev
ARG BUILD_TIME
ARG GIT_COMMIT
ARG TARGETARCH

RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH} go build \
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
