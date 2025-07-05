# Build stage
# Pin to specific golang version with Alpine 3.22
FROM golang:1.24.4-alpine3.22 AS builder

# Pin Alpine package versions for build dependencies
RUN apk add --no-cache \
    git=2.49.0-r0 \
    ca-certificates=20241121-r2 \
    tzdata=2025b-r0

# Set working directory
WORKDIR /build

# Copy go mod and sum files first for better caching
# This layer will be cached unless dependencies change
COPY go.mod go.sum ./

# Download dependencies - this will be cached unless go.mod/go.sum changes
RUN go mod download

# Install specific version of migrate tool - pinned for reproducibility
RUN go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@v4.18.3

# Copy source code (this layer changes most frequently, so it's last)
COPY . .

# Build the application with optimized flags
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' \
    -a -installsuffix cgo \
    -o rss-comb \
    app/main.go

# Final stage - pin Alpine version for reproducible builds
FROM alpine:3.22.0

# Install runtime dependencies with pinned versions
RUN apk add --no-cache \
    ca-certificates=20241121-r2 \
    tzdata=2025b-r0 \
    wget=1.25.0-r1 \
    netcat-openbsd=1.229.1-r0

# Create non-root user (combine RUN commands for fewer layers)
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

# Set working directory
WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /build/rss-comb .

# Copy migrate binary from builder stage
COPY --from=builder /go/bin/migrate /usr/local/bin/migrate

# Copy migrations directory
COPY --from=builder /build/migrations ./migrations

# Create feeds directory and set ownership (combine for fewer layers)
RUN mkdir -p feeds && \
    chown -R appuser:appgroup /app

# Switch to non-root user
USER appuser

# Set default environment variables
ENV GIN_MODE=release \
    PORT=8080 \
    FEEDS_DIR=/app/feeds

# Accept PORT as build argument with default
ARG PORT=8080
EXPOSE $PORT

# Health check with environment variable support
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:$PORT/health || exit 1

# Run the application
CMD ["./rss-comb"]