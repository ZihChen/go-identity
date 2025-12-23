# API Reference

## Merchant APIs
- `GET /api/v1/merchants/:id` - Get merchant by internal ID
- `GET /api/v1/merchants/global/:global_id` - Get merchant by global ID

## Player APIs
- `GET /api/v1/players/:id` - Get player by internal ID
- `GET /api/v1/players/global/:global_id` - Get player by global ID
- `PUT /api/v1/players/:id/active` - Update player last active timestamp

## Manager APIs
- `GET /api/v1/managers/:id` - Get manager by internal ID
- `GET /api/v1/managers/global/:global_id` - Get manager by global ID

## Agent APIs 🆕
- `GET /api/v1/agents/:id` - Get agent by internal ID
- `GET /api/v1/agents/global/:global_id` - Get agent by global ID
- `GET /api/v1/agents/merchant/:merchant_id` - Get agents by merchant ID

## Testing APIs 🆕
- `POST /api/v1/test/kds` - Send test KDS event to Consumer Stream

## System APIs
- `GET /health` - Health check endpoint
- `GET /swagger/*` - Swagger API documentation