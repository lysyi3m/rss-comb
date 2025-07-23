# Build stage
# Pin to specific golang version with Alpine 3.22
FROM golang:1.24-alpine3.22 AS builder

# Install build dependencies (using latest available versions)
RUN apk add --no-cache \
    git \
    ca-certificates \
    tzdata

# Set working directory
WORKDIR /build

# Copy go mod and sum files first for better caching
# This layer will be cached unless dependencies change
COPY go.mod go.sum ./

# Download dependencies - this will be cached unless go.mod/go.sum changes
RUN go mod download

# Copy source code (this layer changes most frequently, so it's last)
COPY . .

# Build the application with optimized flags
ARG TARGETARCH
ARG VERSION=unknown
RUN CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH} go build \
    -ldflags="-w -s -extldflags '-static' -X github.com/lysyi3m/rss-comb/app/cfg.Version=${VERSION}" \
    -a -installsuffix cgo \
    -o rss-comb \
    app/main.go

# Final stage - pin Alpine version for reproducible builds
FROM alpine:3.22.0

# Add OpenContainers annotations
LABEL org.opencontainers.image.title="RSS Comb" \
      org.opencontainers.image.description="RSS/Atom feed proxy with normalization, deduplication, and filtering capabilities" \
      org.opencontainers.image.vendor="lysyi3m" \
      org.opencontainers.image.licenses="MIT" \
      org.opencontainers.image.url="https://github.com/lysyi3m/rss-comb" \
      org.opencontainers.image.source="https://github.com/lysyi3m/rss-comb" \
      org.opencontainers.image.documentation="https://github.com/lysyi3m/rss-comb/blob/main/README.md"

# Install only essential runtime dependencies
# ca-certificates: Required for HTTPS connections to external RSS feeds
# tzdata: Required for timezone support (TZ environment variable)
# Note: wget and nc are available via busybox (built into Alpine base image)
RUN apk add --no-cache \
    ca-certificates \
    tzdata

# Create non-root user (combine RUN commands for fewer layers)
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

# Set working directory
WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /build/rss-comb .

# Migrations are now embedded in the application binary

# Create feeds directory and set ownership (combine for fewer layers)
RUN mkdir -p feeds && \
    chown -R appuser:appgroup /app

# Switch to non-root user
USER appuser

# Set default environment variables
ENV GIN_MODE=release \
    PORT=8080 \
    FEEDS_DIR=/app/feeds \
    TZ=UTC

# Accept PORT as build argument with default
ARG PORT=8080
EXPOSE $PORT

# Optimized health check using busybox wget (available by default in Alpine)
# --spider: Don't download, just check if resource exists
# --quiet: Suppress output
# --tries=1: Only try once, don't retry
# --timeout=5: Timeout after 5 seconds
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget --spider --quiet --tries=1 --timeout=5 http://localhost:$PORT/health || exit 1

# Run the application
CMD ["./rss-comb"]
