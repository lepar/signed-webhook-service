# Build stage
FROM golang:1.25-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
# CGO_ENABLED=0 creates a statically linked binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' \
    -o /build/kii \
    ./cmd/main.go

# Runtime stage
FROM alpine:latest

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1000 kii && \
    adduser -D -u 1000 -G kii kii

# Set working directory
WORKDIR /app

# Copy the binary from builder
COPY --from=builder /build/kii /app/kii

# Copy config files
COPY --from=builder /build/cmd/config /app/cmd/config

# Change ownership to non-root user
RUN chown -R kii:kii /app

# Switch to non-root user
USER kii

# Expose port (default 8080, can be overridden via config)
EXPOSE 8080

# Run the application
ENTRYPOINT ["/app/kii"]
CMD ["server"]

