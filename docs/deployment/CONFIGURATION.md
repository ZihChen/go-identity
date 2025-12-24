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
- JWT token generation for player authentication

## JWT Configuration

The application supports JWT token generation for player authentication. Configure the following environment variables:

### Required Environment Variables

```env
# JWT Configuration
JWT_SECRET=your-jwt-secret-key-here
JWT_ISSUER=your-jwt-issuer-identifier
```

### Configuration Details

| Variable | Description | Default Value | Required |
|----------|-------------|---------------|----------|
| `JWT_SECRET` | Secret key used for signing JWT tokens | None | **Yes** |
| `JWT_ISSUER` | Issuer identifier for JWT tokens | None | **Yes** |

⚠️ **Important**: Both `JWT_SECRET` and `JWT_ISSUER` are required. The application will fail to start if either is missing.

### JWT Token Structure

Generated tokens include the following claims:
- `iss`: Issuer (from JWT_ISSUER config)
- `sub`: Subject (always "player")
- `account`: Player account name
- `player_global_id`: Player's global identifier
- `exp`: Expiration time (1 hour from issue time)

### Example Configuration

```env
JWT_SECRET=REDACTED_JWT_SECRET
JWT_ISSUER=REDACTED_JWT_ISSUER
```

### Security Notes

- Keep `JWT_SECRET` secure and use a strong, random value in production
- The secret should be at least 32 characters long for security
- Both `JWT_SECRET` and `JWT_ISSUER` must be configured - the application will fail to start if either is missing
- Use environment variables or secure configuration management for these values