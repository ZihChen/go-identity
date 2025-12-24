# Event Flow

## System Event Flow

1. Identity events arrive via AWS Kinesis Data Streams
2. Consumer service processes KDS events and enqueues tasks to Redis
3. Worker service processes Redis queue tasks asynchronously for identity synchronization
4. Web service provides HTTP API for direct identity management operations

## Agent Event Flow Implementation
```
External KDS Event → Consumer → Redis Queue → Worker → 
AgentUseCase.SyncAgentData → 
1. Database Upsert (with merchant ID resolution)
2. Publish IdentityAgentSyncEvent to KDS (for downstream services)
```