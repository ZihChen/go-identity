# Configuration Management

## Overview

The application uses Viper for configuration management. Configuration is loaded from environment variables and `.env` files. 

## Key Configuration Areas

### HTTP Server Configuration
- Configurable timeouts (ReadTimeout, WriteTimeout, IdleTimeout) for optimal performance

### Database Connection (MySQL)
- GORM with advanced connection pooling (MaxIdle, MaxOpen, MaxLifetime, MaxIdleTime)

### Redis Connection
- Comprehensive connection pool management with timeout controls and connection lifecycle management

### AWS Credentials and Kinesis Settings
- Event streaming for identity synchronization

### OpenTelemetry Tracing Endpoints
- Distributed tracing across services

### Service-specific Ports and Settings
- Multi-service architecture support

### Authentication Settings
- API middleware configuration for merchant isolation