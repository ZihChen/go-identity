# Development Workflow

## Adding New Features
1. **Define domain entities** in `internal/domain/entity/` following the standardized pattern:
   - Private fields with public deprecated fields for backward compatibility
   - `New[Entity]()` constructor for current time initialization
   - `New[Entity]WithTimes()` constructor for explicit timestamp initialization  
   - Getter methods for all field access (`Get[FieldName]()`)
   - Setter methods for state changes (`Set[FieldName]()`, `Update[FieldName]()`)
   - `IsValid()` method for entity validation
   - `sync[Entity]Fields()` method to maintain backward compatibility
2. Create repository interfaces in `internal/domain/ports/outbound/repository/`
3. Implement repositories in `internal/adapter/outbound/repository/`
4. Create use case interfaces in `internal/domain/ports/inbound/`
5. **Implement use cases** following the domain model pattern with cache integration:
   - Use time-aware constructors for entity creation
   - Access fields through getter methods only
   - Modify state through setter methods
   - Add entity validation calls where appropriate
   - **Inject CacheManager interface for entity lookups with `cache.QueryWithCache[T any]()` generic function**
   - **Implement cache invalidation after successful entity updates using `cache.Del()`**
   - **Use Redis Pipeline for batch cache operations in high-throughput scenarios**
   - **Follow type-safe cache patterns with proper TTL management (5-minute default)**
6. Wire dependencies in `internal/di/`
7. Add HTTP handlers in `internal/adapter/inbound/handler/api/`
8. Register routes in appropriate router files
9. Update Swagger documentation

## Testing Strategy
- **Unit Tests**: Test individual components and business logic
- **Integration Tests**: Test database operations and external service integrations
- **API Tests**: Test HTTP endpoints and responses
- **Performance Tests**: Test scalability and response times