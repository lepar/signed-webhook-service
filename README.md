# Signed Webhook Service

A Go-based HTTP service that accepts signed webhooks with HMAC SHA256 validation, replay attack prevention, and ledger management.

## Features

- **Signed Webhook Validation**: HMAC SHA256 signature validation
- **Replay Attack Prevention**: Nonce and timestamp-based protection
- **Ledger Management**: Thread-safe balance tracking with decimal precision
- **Hexagonal Architecture**: Clean separation of concerns
- **Structured Logging**: JSON logging with request IDs
- **Configurable**: YAML-based configuration with environment variable support

## Quick Start

### Local Development

```bash
# Run directly
CONFIG_ENV=local go run cmd/main.go server

# Build and run
go build -o kii cmd/main.go
CONFIG_ENV=local ./kii server
```

### Docker

```bash
# Build the image
docker build -t kii:latest .

# Run the container
docker run -p 8080:8080 \
  -e CONFIG_ENV=local \
  -e KII_WEBHOOK_HMAC_SECRET=your-secret-key \
  kii:latest

# Or use docker-compose
docker-compose up
```

## Configuration

Configuration is managed through YAML files in `cmd/config/server/`:

- `app-config.yaml.tpl` - Template configuration
- `local.yaml` - Local development configuration

Set `CONFIG_ENV` environment variable to select the environment (defaults to `local`).

### Environment Variables

- `CONFIG_ENV` - Configuration environment (default: `local`)
- `KII_SERVER_PORT` or `PORT` - Server port (default: `8080`)
- `KII_WEBHOOK_HMAC_SECRET` or `HMAC_SECRET` - HMAC secret key
- `KII_WEBHOOK_TIMESTAMP_TOLERANCE` or `TIMESTAMP_TOLERANCE_MINUTES` - Timestamp tolerance (e.g., `5m`)

## API Endpoints

### POST /webhook

Accepts signed webhooks with the following headers:
- `X-Timestamp`: UNIX timestamp
- `X-Nonce`: Unique nonce
- `X-Signature`: HMAC SHA256 signature

Request body:
```json
{
  "user": "string",
  "asset": "string",
  "amount": "string"
}
```

### GET /balance/{user}

Returns the balance for a specific user:

```json
{
  "user": "string",
  "balances": {
    "asset1": "string",
    "asset2": "string"
  }
}
```

## Architecture

The service follows hexagonal architecture (ports and adapters):

```
internal/
├── domain/          # Core business logic (entities, ports)
├── application/     # Use cases
└── infrastructure/  # Adapters (HTTP, validators, repositories)
```

## Building

```bash
# Build binary
go build -o kii cmd/main.go

# Build Docker image
docker build -t kii:latest .
```

## Testing

```bash
go test ./...
```

```Test the endpoints with a script
./test_webhooks.sh
```