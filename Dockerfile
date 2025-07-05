# Build stage
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /build

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Install migrate tool
RUN go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' \
    -a -installsuffix cgo \
    -o rss-comb \
    app/main.go

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS requests, tzdata for timezone support, wget for health checks, and netcat for database checks
RUN apk --no-cache add ca-certificates tzdata wget netcat-openbsd

# Create non-root user
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

# Set working directory
WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /build/rss-comb .

# Copy migrate binary from builder stage
COPY --from=builder /go/bin/migrate /usr/local/bin/migrate

# Copy migrations
COPY --from=builder /build/migrations ./migrations

# Create feeds directory
RUN mkdir -p feeds && \
    chown -R appuser:appgroup /app

# Switch to non-root user
USER appuser

# Set default environment variables
ENV GIN_MODE=release
ENV PORT=8080
ENV FEEDS_DIR=/app/feeds

# Expose port (uses build-time ARG or defaults to 8080)
ARG PORT=8080
EXPOSE $PORT

# Health check (uses PORT environment variable at runtime)
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:$PORT/health || exit 1

# Run the application
CMD ["./rss-comb"]