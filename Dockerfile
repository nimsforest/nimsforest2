# Build stage
FROM golang:1.24-bookworm AS builder

# Install build dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    git \
    make \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Set working directory
WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a \
    -ldflags="-s -w" \
    -o forest ./cmd/forest

# Final stage - Debian Bookworm slim
FROM debian:bookworm-slim

# Install runtime dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    curl \
    procps \
    && rm -rf /var/lib/apt/lists/*

# Create non-root user
RUN groupadd -g 1000 forest && \
    useradd -u 1000 -g forest -s /bin/bash -m forest

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
