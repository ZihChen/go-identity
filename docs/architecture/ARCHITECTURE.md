# System Architecture

## Project Overview

Fat Identity Cat is a Go-based microservice for managing merchant, player, and manager identities with synchronization capabilities through AWS Kinesis Data Streams (KDS). It serves as a central identity management system that handles identity data and provides APIs for other services to query and manage identity information.

## Key Components

The application consists of three main services that can be run independently:

1. **Web Service** (`cmd/web`) - HTTP API server using Gin framework, provides REST endpoints for identity management
2. **Consumer Service** (`cmd/consumer`) - Processes identity events from AWS Kinesis Data Streams
3. **Worker Service** (`cmd/worker`) - Background job processor for handling identity synchronization tasks

## Core Domain Entities

- **Merchant** - Business entities in the system with API keys and global identifiers
- **Player** - End users/customers with levels, tags, and activity tracking
- **Manager** - Administrative users for merchant management
- **Agent** - Proxy/agent entities with hierarchical ancestry tracking 🆕
- **Level** - Player level classification system
- **Tag** - Player categorization and labeling system
- **PlayerTag** - Many-to-many relationship between players and tags

## Clean Architecture Layers

The codebase follows hexagonal architecture with clear separation:

- `internal/domain/` - Core business logic, interfaces (ports)
  - `entity/` - Domain entities (Merchant, Player, Manager, Agent, Level, Tag, PlayerTag) + cross-boundary types
    - `Span` interface, `SpanAttr` + helper funcs — domain-owned tracing types (replaces OTel types in ports)
    - `CacheSetEntry` — domain-owned cache entry type (replaces Redis types in ports)
  - `dto/` - Data transfer objects for API communication
  - `ports/` - Interface definitions split into inbound and outbound
    - `inbound/` - Use Case interfaces
    - `outbound/` - Repository, Service, and Infrastructure interfaces
      - `infrastructure/` - `DistributedLockService` port (replaces direct redsync dependency)
  - `consts/` - Domain constants and enumerations
  - `errmsg/` - Sentinel error constants for all entity validation errors (18 errors across 6 entities)
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
    - `middleware/` - **Advanced HTTP middleware system**
      - `cors_middleware.go` - Environment-aware CORS with security controls
      - `auth_middleware.go` - API Key authentication with merchant isolation
      - `tracing_middleware.go` - OpenTelemetry integration for distributed tracing
  - `outbound/` - Outbound adapters (external dependencies)
    - `repository/` - Database operations organized by domain
      - `merchant/` - Merchant-related repositories
      - `player/` - Player-related repositories with tag management
      - `manager/` - Manager-related repositories
      - `agent/` - Agent-related repositories with ancestry tracking 🆕
      - `level/` - Level management repositories
      - `tag/` - Tag management repositories
- `internal/infrastructure/` - External dependencies
  - `database/mysql/` - MySQL with GORM
  - `cache/redis/` - Redis caching and queuing with Pipeline optimization
  - `cache/helper.go` - Type-safe generic cache functions with CacheManager integration 🆕
  - `kds/` - AWS Kinesis integration for event streaming
    - `backoff_strategy.go` - Adaptive backoff management for Consumer resilience
    - `consumer.go` - KDS event consumption with batch processing and worker pools
    - `enqueue.go` - Event enqueueing and processing logic
    - `kds.go` - Core KDS service with AWS SDK integration
    - `producer.go` - KDS event production for identity synchronization
  - `queue/` - Background job queue management
  - `tracing/` - OpenTelemetry distributed tracing
  - `config/` - Configuration management
  - `logger/` - Structured logging

## Dependency Injection

Uses Google Wire for compile-time dependency injection (`internal/di/wire.go`)

## Key Differences from Fat-Notification-Cat

While both services share similar architectural patterns, Fat Identity Cat focuses specifically on:

- **Identity Management**: Core entities are Merchant, Player, Manager with their relationships
- **Synchronization Focus**: Heavy emphasis on keeping identity data synchronized via KDS
- **Simpler Domain Model**: More straightforward entity relationships compared to notification campaigns
- **Activity Tracking**: Player activity tracking with `last_active_at` field updates
- **Tag and Level Management**: Player categorization system for identity classification