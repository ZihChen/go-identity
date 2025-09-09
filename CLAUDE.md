# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Fat Identity Cat is a Go-based microservice for managing merchant, player, and manager identities with synchronization capabilities through AWS Kinesis Data Streams (KDS). It serves as a central identity management system that handles identity data and provides APIs for other services to query and manage identity information.

## Architecture

### Key Components

The application consists of three main services that can be run independently:

1. **Web Service** (`cmd/web`) - HTTP API server using Gin framework, provides REST endpoints for identity management
2. **Consumer Service** (`cmd/consumer`) - Processes identity events from AWS Kinesis Data Streams
3. **Worker Service** (`cmd/worker`) - Background job processor for handling identity synchronization tasks

### Core Domain Entities

- **Merchant** - Business entities in the system with API keys and global identifiers
- **Player** - End users/customers with levels, tags, and activity tracking
- **Manager** - Administrative users for merchant management
- **Level** - Player level classification system
- **Tag** - Player categorization and labeling system
- **PlayerTag** - Many-to-many relationship between players and tags

### Clean Architecture Layers

The codebase follows hexagonal architecture with clear separation:

- `internal/domain/` - Core business logic, interfaces (ports)
  - `entity/` - Domain entities (Merchant, Player, Manager, Level, Tag, PlayerTag)
  - `dto/` - Data transfer objects for API communication
  - `ports/` - Interface definitions split into inbound and outbound
    - `inbound/` - Use Case interfaces
    - `outbound/` - Repository, Service, and Infrastructure interfaces
  - `consts/` - Domain constants and enumerations
  - `errmsg/` - Custom error message definitions
  - `event/` - Event structure definitions for KDS integration
- `internal/adapter/` - Implementation of domain interfaces
  - `inbound/` - Inbound adapters (external requests)
    - `handler/` - HTTP and Worker handlers
      - `api/` - HTTP API handlers for web service
      - `worker/` - Worker task handlers for background processing
    - `router/` - **Unified Router Management System**
      - `router_manager.go` - Central router coordinator with middleware management
      - `api_router.go` - API route registration
      - `swagger_router.go` - Swagger documentation routes
      - `health_router.go` - Health check routes
    - `middleware/` - HTTP middleware (tracing, authentication, etc.)
  - `outbound/` - Outbound adapters (external dependencies)
    - `repository/` - Database operations organized by domain
      - `merchant/` - Merchant-related repositories
      - `player/` - Player-related repositories with tag management
      - `manager/` - Manager-related repositories
      - `level/` - Level management repositories
      - `tag/` - Tag management repositories
- `internal/infrastructure/` - External dependencies
  - `database/mysql/` - MySQL with GORM
  - `cache/redis/` - Redis caching and queuing
  - `kds/` - AWS Kinesis integration for event streaming
  - `queue/` - Background job queue management
  - `tracing/` - OpenTelemetry distributed tracing
  - `config/` - Configuration management
  - `logger/` - Structured logging

### Dependency Injection

Uses Google Wire for compile-time dependency injection (`internal/di/wire.go`)

## Common Development Commands

### Build and Run

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

### Testing

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

### Database Migrations

```bash
# Using migration script (recommended)
./migrate.sh apply      # Apply migrations
./migrate.sh status     # Check migration status
./migrate.sh gen <name> # Generate new migration

# Direct Atlas commands
atlas migrate apply --env local
atlas migrate diff <migration_name> --env local
```

### API Documentation

The project uses Swagger for API documentation:

```bash
# Generate/update Swagger docs
swag init

# Access Swagger UI (when web service is running)
# http://localhost:8080/swagger/index.html
```

### Wire Dependency Injection

```bash
# Regenerate wire_gen.go after modifying wire.go
wire ./internal/di
```

### Code Quality

```bash
# Run linter
golangci-lint run

# Format code
go fmt ./...

# Vet code
go vet ./...
```

## Event Flow

1. Identity events arrive via AWS Kinesis Data Streams
2. Consumer service processes KDS events and enqueues tasks to Redis
3. Worker service processes Redis queue tasks asynchronously for identity synchronization
4. Web service provides HTTP API for direct identity management operations

## Configuration

The application uses Viper for configuration management. Configuration is loaded from environment variables and `.env` files. Key configuration areas include:

- **HTTP Server configuration** - Configurable timeouts (ReadTimeout, WriteTimeout, IdleTimeout) for optimal performance
- **Database connection (MySQL)** - GORM with advanced connection pooling (MaxIdle, MaxOpen, MaxLifetime, MaxIdleTime)
- **Redis connection** - Comprehensive connection pool management with timeout controls and connection lifecycle management
- **AWS credentials and Kinesis settings** - Event streaming for identity synchronization
- **OpenTelemetry tracing endpoints** - Distributed tracing across services
- **Service-specific ports and settings** - Multi-service architecture support
- **Authentication settings** - API middleware configuration for merchant isolation

## Important Patterns

### Repository Pattern
All database operations go through repository interfaces defined in `internal/domain/ports/outbound/repository/`, with implementations organized by business domain in `internal/adapter/outbound/repository/`

### Use Case Pattern
Business logic is encapsulated in use cases that implement interfaces from `internal/domain/ports/inbound/` and orchestrate repositories and services

### Unified Router Management Pattern ✨ **RECENTLY IMPLEMENTED**
Centralized router architecture with:
- **Router Manager** - Central coordinator for all route registration with `SetupRoutersWithMiddleware()` method
- **Component Routers** - Individual routers for specific functionality (API, Swagger, health)
- **Middleware Integration** - Unified middleware management (tracing, authentication, CORS)
- **Environment-aware Configuration** - Different settings for development vs production

### Event-Driven Architecture
System uses events for inter-service communication via KDS and Redis queues for identity synchronization

### Error Handling
Custom error types defined in `internal/domain/errmsg/` for consistent error handling across the application

## API Endpoints

### Merchant APIs
- `GET /api/v1/merchants/:id` - Get merchant by internal ID
- `GET /api/v1/merchants/global/:global_id` - Get merchant by global ID

### Player APIs
- `GET /api/v1/players/:id` - Get player by internal ID
- `GET /api/v1/players/global/:global_id` - Get player by global ID
- `PUT /api/v1/players/:id/active` - Update player last active timestamp

### Manager APIs
- `GET /api/v1/managers/:id` - Get manager by internal ID
- `GET /api/v1/managers/global/:global_id` - Get manager by global ID

### System APIs
- `GET /health` - Health check endpoint
- `GET /swagger/*` - Swagger API documentation

## Recent Architecture Updates

### Router Management System Migration (Latest)
- **Unified RouterManager**: All routing logic centralized in `internal/adapter/inbound/router/router_manager.go`
- **Middleware Integration**: TracingMiddleware and other middlewares unified in RouterManager
- **Code Simplification**: Web service startup code simplified by using `SetupRoutersWithMiddleware()`
- **Backward Compatibility**: Original `RegisterRoutes()` method maintained for compatibility

### Middleware Organization
- **Centralized Location**: All middleware moved to `internal/adapter/inbound/middleware/`
- **Unified Management**: Middleware configuration handled by RouterManager
- **Cleaner Architecture**: Separation of concerns between routing and business logic

### Handler Separation
- **API Handlers** (`internal/adapter/inbound/handler/api/`) - HTTP request handling
- **Worker Handlers** (`internal/adapter/inbound/handler/worker/`) - Background task handling
- **Clear Responsibilities**: Each handler type focuses on specific concerns

## Development Workflow

### Adding New Features
1. Define domain entities in `internal/domain/entity/`
2. Create repository interfaces in `internal/domain/ports/outbound/repository/`
3. Implement repositories in `internal/adapter/outbound/repository/`
4. Create use case interfaces in `internal/domain/ports/inbound/`
5. Implement use cases and wire dependencies in `internal/di/`
6. Add HTTP handlers in `internal/adapter/inbound/handler/api/`
7. Register routes in appropriate router files
8. Update Swagger documentation

### Testing Strategy
- **Unit Tests**: Test individual components and business logic
- **Integration Tests**: Test database operations and external service integrations
- **API Tests**: Test HTTP endpoints and responses
- **Performance Tests**: Test scalability and response times

## Key Differences from Fat-Notification-Cat

While both services share similar architectural patterns, Fat Identity Cat focuses specifically on:

- **Identity Management**: Core entities are Merchant, Player, Manager with their relationships
- **Synchronization Focus**: Heavy emphasis on keeping identity data synchronized via KDS
- **Simpler Domain Model**: More straightforward entity relationships compared to notification campaigns
- **Activity Tracking**: Player activity tracking with `last_active_at` field updates
- **Tag and Level Management**: Player categorization system for identity classification

## Current Status

**v2.0 Router Architecture Migration Completed**: Unified router management system implemented  
**v1.0 Core Identity Management**: Basic CRUD operations for all identity entities completed

### Recently Completed
- ✅ Unified Router Management System implementation
- ✅ Middleware centralization and organization
- ✅ Handler separation (API vs Worker)
- ✅ Clean architecture with hexagonal pattern
- ✅ Comprehensive API documentation with Swagger
- ✅ Event-driven architecture via KDS integration
- ✅ Background job processing with Redis queues

### Current Focus
- Comprehensive testing and validation
- Performance optimization
- Security enhancements
- Documentation updates
- Integration testing between services

The system is production-ready for identity management operations with robust synchronization capabilities.