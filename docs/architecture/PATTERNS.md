# Important Architectural Patterns

## Repository Pattern
All database operations go through repository interfaces defined in `internal/domain/ports/outbound/repository/`, with implementations organized by business domain in `internal/adapter/outbound/repository/`

## Use Case Pattern
Business logic is encapsulated in use cases that implement interfaces from `internal/domain/ports/inbound/` and orchestrate repositories and services

## Domain Model Encapsulation Pattern ✨ **RECENTLY IMPLEMENTED**
Standardized domain entity management with:
- **Time-Aware Constructors** - Separate constructors for current time vs explicit timestamps
- **Getter Method Access** - All field access through encapsulated methods
- **Entity Validation** - Built-in `IsValid()` methods integrated into use case flows
- **Setter Method State Changes** - Controlled state modification through entity methods
- **Backward Compatibility** - Deprecated public fields maintain API compatibility

## Unified Router Management Pattern ✨ **IMPLEMENTED**
Centralized router architecture with:
- **Router Manager** - Central coordinator for all route registration with `SetupRoutersWithMiddleware()` method
- **Component Routers** - Individual routers for specific functionality (API, Swagger, health)
- **Middleware Integration** - Unified middleware management (tracing, authentication, CORS)
- **Environment-aware Configuration** - Different settings for development vs production

## Universal Cache Integration Pattern ✨ **RECENTLY IMPLEMENTED**
Comprehensive Redis cache integration across all UseCase layers with:
- **Type-Safe Generic Cache Functions** - `QueryWithCache[T any]()` with compile-time type checking
- **Clean Architecture Compliance** - CacheManager interface injection in UseCase layer
- **Redis Pipeline Optimization** - Batch cache invalidation for high-throughput scenarios
- **Cache-Aside Pattern** - Standard cache-aside with TTL support and database fallback
- **Testing Infrastructure** - NilCacheManager for comprehensive test coverage

## Event-Driven Architecture
System uses events for inter-service communication via KDS and Redis queues for identity synchronization

## Domain Type Isolation Pattern ✨ **RECENTLY IMPLEMENTED**
Domain port interfaces must not reference any framework or infrastructure types. All cross-boundary types are defined in the domain layer:
- **`entity.Span`** - Domain interface replacing OTel `trace.Span` in `TracingService` port
- **`entity.SpanAttr`** + helper funcs (`StringAttr`, `IntAttr`, `Int64Attr`, `BoolAttr`) - Replacing OTel `attribute.KeyValue`
- **`entity.CacheSetEntry`** - Domain-typed cache entry replacing Redis-specific structs
- **`infrastructure.DistributedLockService`** - Domain port replacing direct `redsync` usage
- Infrastructure wrappers (`otelSpan`, `RedisLockService`) bridge domain types to concrete frameworks

## Sentinel Error Pattern ✨ **RECENTLY IMPLEMENTED**
All entity validation errors are defined as sentinel errors in `internal/domain/errmsg/errors.go`:
- Callers can use `errors.Is()` for precise error comparison
- Entities (`merchant.go`, `player.go`, `manager.go`, `tag.go`, `level.go`, `agent.go`) import `errmsg` and return named constants instead of inline `errors.New("string")`
- 18 sentinel errors covering all 6 entity types

## Error Handling
Custom error types defined in `internal/domain/errmsg/` for consistent error handling across the application

## Entity Constructor Patterns
**Time-Aware Constructors**: All entities now support both current time and explicit timestamp initialization
```go
// Current time constructors (for new entities)
entity.NewTag(merchantID, name, globalTagID)
entity.NewMerchant(globalMerchantID, name)
entity.NewManager(merchantID, globalManagerID, account, email)
entity.NewLevel(merchantID, name, globalPlayerLevelID, globalMerchantID)

// Explicit time constructors (for sync operations)
entity.NewTagWithTimes(merchantID, name, globalTagID, updatedAt)
entity.NewMerchantWithTimes(globalMerchantID, name, displayName, updatedAt)
entity.NewManagerWithTimes(merchantID, globalManagerID, account, email, createdAt, updatedAt)
entity.NewLevelWithTimes(merchantID, name, globalPlayerLevelID, globalMerchantID, createdAt, updatedAt)
```

## Cache Integration Architecture
```
UseCase Layer (Business Logic + Cache Integration)
    ↓ Dependencies Injection
CacheManager Interface (Domain Port)
    ↓ Implementation
Redis Manager (Infrastructure)
```

## Router Management System Migration (v2.0)
- **Unified RouterManager**: All routing logic centralized in `internal/adapter/inbound/router/router_manager.go`
- **Middleware Integration**: TracingMiddleware and other middlewares unified in RouterManager
- **Code Simplification**: Web service startup code simplified by using `SetupRoutersWithMiddleware()`
- **Backward Compatibility**: Original `RegisterRoutes()` method maintained for compatibility

## Handler and Middleware Organization
- **API Handlers** (`internal/adapter/inbound/handler/api/`) - HTTP request handling
- **Consumer Handlers** (`internal/adapter/inbound/handler/consumer/`) - KDS event processing
- **Worker Handlers** (`internal/adapter/inbound/handler/worker/`) - Background task handling
- **Advanced Middleware System**: All middleware moved to `internal/adapter/inbound/middleware/`
  - Environment-aware CORS configuration
  - API Key authentication with security validation
  - OpenTelemetry distributed tracing
  - Production-ready security controls