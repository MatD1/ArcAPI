# Arc Raiders REST API Documentation

## Base URL

```
http://localhost:8080/api/v1
```

## Authentication

All protected endpoints require both:
1. **API Key** in `X-API-Key` header
2. **JWT Token** in `Authorization: Bearer <token>` header

## Authentication Endpoints

### GitHub OAuth Login

Initiate GitHub OAuth flow.

**Endpoint:** `GET /auth/github/login`

**Query Parameters:**
- `state` (optional): State parameter for OAuth flow

**Response:** Redirects to GitHub OAuth page

**Example:**
```
GET /api/v1/auth/github/login?state=random-state
```

### GitHub OAuth Callback

Handle OAuth callback and return JWT token.

**Endpoint:** `GET /auth/github/callback`

**Query Parameters:**
- `code`: Authorization code from GitHub
- `state`: State parameter

**Response:**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user": {
    "id": 1,
    "github_id": "username",
    "email": "user@example.com",
    "username": "username",
    "role": "user"
  }
}
```

### Login with API Key

Authenticate with API key and receive JWT token.

**Endpoint:** `POST /auth/login`

**Request Body:**
```json
{
  "api_key": "your-api-key-here"
}
```

**Response:**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user": {
    "id": 1,
    "email": "user@example.com",
    "username": "username",
    "role": "user"
  }
}
```

## Data Endpoints

All data endpoints support pagination with `page` and `limit` query parameters.

### Missions

#### List Missions

**Endpoint:** `GET /missions`

**Query Parameters:**
- `page` (optional, default: 1): Page number
- `limit` (optional, default: 20, max: 100): Items per page

**Response:**
```json
{
  "data": [
    {
      "id": 1,
      "external_id": "mission-1",
      "name": "Mission Name",
      "description": "Mission description",
      "data": { ... },
      "synced_at": "2024-01-01T00:00:00Z",
      "created_at": "2024-01-01T00:00:00Z",
      "updated_at": "2024-01-01T00:00:00Z"
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 20,
    "total": 100
  }
}
```

#### Get Mission

**Endpoint:** `GET /missions/:id`

**Response:**
```json
{
  "id": 1,
  "external_id": "mission-1",
  "name": "Mission Name",
  "description": "Mission description",
  "data": { ... },
  "synced_at": "2024-01-01T00:00:00Z",
  "created_at": "2024-01-01T00:00:00Z",
  "updated_at": "2024-01-01T00:00:00Z"
}
```

#### Create Mission

**Endpoint:** `POST /missions`

**Request Body:**
```json
{
  "external_id": "mission-1",
  "name": "Mission Name",
  "description": "Mission description",
  "data": { ... }
}
```

**Response:** 201 Created with mission object

#### Update Mission

**Endpoint:** `PUT /missions/:id`

**Request Body:**
```json
{
  "name": "Updated Mission Name",
  "description": "Updated description",
  "data": { ... }
}
```

**Response:** 200 OK with updated mission object

#### Delete Mission

**Endpoint:** `DELETE /missions/:id`

**Response:** 204 No Content

### Items

#### List Items (Paginated)

**Endpoint:** `GET /items`

**Query Parameters:**
- `page` (optional, default: 1): Page number
- `limit` (optional, default: 20, max: 100): Items per page

**Response:**
```json
{
  "data": [...],
  "pagination": {
    "page": 1,
    "limit": 20,
    "total": 500
  }
}
```

#### List All Items (Unpaginated)

**Endpoint:** `GET /items?all=true`

**Query Parameters:**
- `all=true` (required): Returns all items without pagination

**Response:**
```json
{
  "data": [...],
  "total": 500
}
```

**⚠️ Note:** This endpoint returns all items in a single response. Use for initial data loading or when you need the complete dataset. For browsing, use the paginated endpoint.

#### Other Item Endpoints

- `GET /items/:id` - Get single item
- `GET /items/required` - Get all required items for quests/hideout
- `GET /items/blueprints` - Get all blueprint items
- `POST /items` - Create item (admin only)
- `PUT /items/:id` - Update item (admin only)
- `DELETE /items/:id` - Delete item (admin only)

**Additional field:** Items include `image_url` field.

### Skill Nodes

#### List Skill Nodes (Paginated)

**Endpoint:** `GET /skill-nodes`

**Query Parameters:**
- `page` (optional, default: 1): Page number
- `limit` (optional, default: 20, max: 100): Items per page

#### List All Skill Nodes (Unpaginated)

**Endpoint:** `GET /skill-nodes?all=true`

Returns all skill nodes in a single response.

#### Other Skill Node Endpoints

- `GET /skill-nodes/:id` - Get skill node
- `POST /skill-nodes` - Create skill node (admin only)
- `PUT /skill-nodes/:id` - Update skill node (admin only)
- `DELETE /skill-nodes/:id` - Delete skill node (admin only)

### Hideout Modules

#### List Hideout Modules (Paginated)

**Endpoint:** `GET /hideout-modules`

**Query Parameters:**
- `page` (optional, default: 1): Page number
- `limit` (optional, default: 20, max: 100): Items per page

#### List All Hideout Modules (Unpaginated)

**Endpoint:** `GET /hideout-modules?all=true`

Returns all hideout modules in a single response.

#### Other Hideout Module Endpoints

- `GET /hideout-modules/:id` - Get hideout module
- `POST /hideout-modules` - Create hideout module (admin only)
- `PUT /hideout-modules/:id` - Update hideout module (admin only)
- `DELETE /hideout-modules/:id` - Delete hideout module (admin only)

## Management Endpoints (Admin Only)

### Create API Key

**Endpoint:** `POST /admin/api-keys`

**Request Body:**
```json
{
  "name": "My API Key"
}
```

**Response:**
```json
{
  "api_key": "generated-api-key-here",
  "name": "My API Key",
  "warning": "Save this API key now. You won't be able to see it again."
}
```

### List API Keys

**Endpoint:** `GET /admin/api-keys`

**Response:**
```json
[
  {
    "id": 1,
    "user_id": 1,
    "name": "My API Key",
    "last_used_at": "2024-01-01T00:00:00Z",
    "revoked_at": null,
    "created_at": "2024-01-01T00:00:00Z"
  }
]
```

### Revoke API Key

**Endpoint:** `DELETE /admin/api-keys/:id`

**Response:**
```json
{
  "message": "API key revoked"
}
```

### Revoke JWT Token

**Endpoint:** `POST /admin/jwts/revoke`

**Request Body:**
```json
{
  "token": "jwt-token-to-revoke"
}
```

**Response:**
```json
{
  "message": "JWT token revoked"
}
```

### List Active JWT Tokens

**Endpoint:** `GET /admin/jwts`

**Response:**
```json
[
  {
    "id": 1,
    "user_id": 1,
    "expires_at": "2024-01-04T00:00:00Z",
    "revoked_at": null,
    "created_at": "2024-01-01T00:00:00Z"
  }
]
```

### Query Audit Logs

**Endpoint:** `GET /admin/logs`

**Query Parameters:**
- `page` (optional, default: 1): Page number
- `limit` (optional, default: 50, max: 100): Items per page
- `api_key_id` (optional): Filter by API key ID
- `jwt_token_id` (optional): Filter by JWT token ID
- `user_id` (optional): Filter by user ID
- `endpoint` (optional): Filter by endpoint
- `method` (optional): Filter by HTTP method
- `start_time` (optional): Filter by start time (ISO 8601)
- `end_time` (optional): Filter by end time (ISO 8601)

**Response:**
```json
{
  "data": [
    {
      "id": 1,
      "api_key_id": 1,
      "jwt_token_id": 1,
      "user_id": 1,
      "endpoint": "/api/v1/missions",
      "method": "GET",
      "status_code": 200,
      "request_body": null,
      "response_time_ms": 15,
      "ip_address": "127.0.0.1",
      "created_at": "2024-01-01T00:00:00Z"
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 50,
    "total": 1000
  }
}
```

## Error Responses

All errors follow this format:

```json
{
  "error": "Error message description"
}
```

**HTTP Status Codes:**
- `200 OK` - Success
- `201 Created` - Resource created
- `204 No Content` - Success (delete operations)
- `400 Bad Request` - Invalid request
- `401 Unauthorized` - Authentication required or invalid
- `403 Forbidden` - Insufficient permissions
- `404 Not Found` - Resource not found
- `500 Internal Server Error` - Server error

## Examples

### Complete Authentication Flow

```bash
# 1. Login with API key
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"api_key": "your-api-key"}'

# Response: {"token": "...", "user": {...}}

# 2. Use API key + JWT token for protected endpoints
curl http://localhost:8080/api/v1/missions \
  -H "X-API-Key: your-api-key" \
  -H "Authorization: Bearer your-jwt-token"
```

### Create and Use API Key

```bash
# 1. Create API key (requires admin JWT)
curl -X POST http://localhost:8080/api/v1/admin/api-keys \
  -H "X-API-Key: admin-api-key" \
  -H "Authorization: Bearer admin-jwt-token" \
  -H "Content-Type: application/json" \
  -d '{"name": "My New Key"}'

# Response: {"api_key": "new-key-here", ...}

# 2. Login with new API key
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"api_key": "new-key-here"}'
```

