# Build stage
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make

# Set working directory
WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo \
    -ldflags="-s -w" \
    -o forest ./cmd/forest

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS and curl for health checks
RUN apk --no-cache add ca-certificates curl

# Create non-root user
RUN addgroup -g 1000 forest && \
    adduser -D -u 1000 -G forest forest

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/forest .

# Change ownership
RUN chown -R forest:forest /app

# Switch to non-root user
USER forest

# Expose default NATS port (if running NATS locally)
# Note: In production, NATS should run separately
EXPOSE 4222 8222

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD pgrep -x forest || exit 1

# Run the application
ENTRYPOINT ["/app/forest"]
