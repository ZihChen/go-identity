# Development Commands

## Build and Run

```bash
# Run specific service locally (recommended for development)
go run main.go web       # Start web server on :8080
go run main.go consumer  # Start KDS consumer
go run main.go worker    # Start background worker

# Docker Compose for local development
docker-compose up -d --build

# Build specific service
go build -o bin/web cmd/web/*.go
go build -o bin/consumer cmd/consumer/*.go
go build -o bin/worker cmd/worker/*.go
```

## Testing

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run specific package tests
go test ./internal/adapter/outbound/repository/...
go test ./internal/adapter/inbound/handler/...

# Run integration tests
go test ./test/...
```

## Database Migrations

```bash
# Using migration script (recommended)
./migrate.sh apply      # Apply migrations
./migrate.sh status     # Check migration status
./migrate.sh gen <name> # Generate new migration

# Direct Atlas commands
atlas migrate apply --env local
atlas migrate diff <migration_name> --env local
```

## API Documentation

The project uses Swagger for API documentation:

```bash
# Generate/update Swagger docs
swag init

# Access Swagger UI (when web service is running)
# http://localhost:8080/swagger/index.html
```

## Wire Dependency Injection

```bash
# Regenerate wire_gen.go after modifying wire.go
wire ./internal/di
```

## Code Quality

```bash
# Run linter
golangci-lint run

# Format code
go fmt ./...

# Vet code
go vet ./...
```