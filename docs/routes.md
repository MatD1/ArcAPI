# Arc Raiders API - Complete Routes Documentation

## Base URL

```
http://localhost:8080/api/v1
```

## Authentication

### Public Endpoints (No Authentication Required)

#### 1. GitHub OAuth Login
**Endpoint:** `GET /auth/github/login`

**Description:** Initiates GitHub OAuth flow for user authentication.

**Query Parameters:**
- `redirect` (optional): Deep link URL for mobile apps (e.g., `arcdb://auth/callback`)
- `client` (optional): Client type - `mobile` or `web` (defaults to `web`)

**Response:** Redirects to GitHub OAuth page

**Example:**
```
GET /api/v1/auth/github/login?client=web
GET /api/v1/auth/github/login?redirect=arcdb://auth/callback&client=mobile
```

#### 2. GitHub OAuth Callback
**Endpoint:** `GET /auth/github/callback`

**Description:** Handles OAuth callback from GitHub and redirects to frontend or mobile app.

**Query Parameters:**
- `code`: Authorization code from GitHub
- `state`: OAuth state parameter (contains redirect URL and client type)

**Response:** Redirects to frontend callback URL or mobile deep link

#### 3. Exchange Temporary Token
**Endpoint:** `GET /auth/exchange-token`

**Description:** Exchanges a temporary token for JWT and user data.

**Query Parameters:**
- `token`: Temporary token received from OAuth callback

**Response:**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user": {
    "id": 1,
    "username": "username",
    "email": "user@example.com",
    "role": "user"
  },
  "api_key": "optional-api-key-if-auto-created"
}
```

#### 4. API Key Login
**Endpoint:** `POST /auth/login`

**Description:** Authenticates with API key and returns JWT token.

**Headers:**
- `Content-Type: application/json`

**Request Body:**
```json
{
  "api_key": "your-api-key"
}
```

**Response:**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user": {
    "id": 1,
    "username": "username",
    "email": "user@example.com",
    "role": "user"
  }
}
```

#### 5. Mobile Callback Page
**Endpoint:** `GET /auth/mobile-callback`

**Description:** Web page that redirects mobile apps to deep links. Used internally by OAuth flow.

**Query Parameters:**
- `token`: Temporary authentication token
- `redirect`: Deep link URL (e.g., `arcdb://auth/callback`)

**Response:** HTML page with JavaScript redirect

---

### Read-Only Endpoints (Require JWT Token)

**Authentication:** `Authorization: Bearer <jwt-token>`

**Note:** Basic users must have `can_access_data` enabled by an admin to access these endpoints.

#### Quests

##### 6. List Quests
**Endpoint:** `GET /quests`

**Description:** Returns a paginated list of all quests.

**Query Parameters:**
- `page` (optional): Page number (default: 1)
- `limit` (optional): Items per page (default: 20, max: 100)

**Response:**
```json
{
  "data": [
    {
      "id": 1,
      "external_id": "quest_001",
      "name": "Quest Name",
      "description": "Quest description",
      "trader": "Trader Name",
      "objectives": {...},
      "reward_item_ids": {...},
      "xp": 100,
      "data": {...},
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

##### 7. Get Quest by ID
**Endpoint:** `GET /quests/:id`

**Description:** Returns a single quest by its ID.

**Path Parameters:**
- `id`: Quest ID

**Response:**
```json
{
  "id": 1,
  "external_id": "quest_001",
  "name": "Quest Name",
  ...
}
```

#### Items

##### 8. List Items
**Endpoint:** `GET /items`

**Description:** Returns a paginated list of all items.

**Query Parameters:**
- `page` (optional): Page number (default: 1)
- `limit` (optional): Items per page (default: 20, max: 100)

**Response:** Same format as quests list

##### 9. Get Item by ID
**Endpoint:** `GET /items/:id`

**Description:** Returns a single item by its ID.

**Path Parameters:**
- `id`: Item ID

##### 10. Get Required Items
**Endpoint:** `GET /items/required`

**Description:** Returns all items required for quests and hideout modules, with aggregated quantities and usage information.

**Response:**
```json
{
  "data": [
    {
      "item": {
        "id": 1,
        "external_id": "item_001",
        "name": "Item Name",
        ...
      },
      "total_quantity": 15,
      "usages": [
        {
          "source_type": "quest",
          "source_id": 1,
          "source_name": "Quest Name",
          "quantity": 5,
          "level": null
        },
        {
          "source_type": "hideout_module",
          "source_id": 2,
          "source_name": "Module Name",
          "quantity": 10,
          "level": 3
        }
      ]
    }
  ],
  "total": 50
}
```

#### Skill Nodes

##### 11. List Skill Nodes
**Endpoint:** `GET /skill-nodes`

**Description:** Returns a paginated list of all skill nodes.

**Query Parameters:**
- `page` (optional): Page number (default: 1)
- `limit` (optional): Items per page (default: 20, max: 100)

##### 12. Get Skill Node by ID
**Endpoint:** `GET /skill-nodes/:id`

**Description:** Returns a single skill node by its ID.

**Path Parameters:**
- `id`: Skill node ID

#### Hideout Modules

##### 13. List Hideout Modules
**Endpoint:** `GET /hideout-modules`

**Description:** Returns a paginated list of all hideout modules.

**Query Parameters:**
- `page` (optional): Page number (default: 1)
- `limit` (optional): Items per page (default: 20, max: 100)

##### 14. Get Hideout Module by ID
**Endpoint:** `GET /hideout-modules/:id`

**Description:** Returns a single hideout module by its ID.

**Path Parameters:**
- `id`: Hideout module ID

---

### Progress Endpoints (Require JWT Token - Basic Users Can Update Own Progress)

**Authentication:** `Authorization: Bearer <jwt-token>`

**Note:** All progress endpoints allow basic users to read and update their own progress. Admins can access any user's progress (future enhancement).

#### Quest Progress

##### 15. Get My Quest Progress
**Endpoint:** `GET /progress/quests`

**Description:** Returns all quest completion progress for the authenticated user.

**Response:**
```json
{
  "data": [
    {
      "id": 1,
      "user_id": 1,
      "quest_id": 5,
      "completed": true,
      "created_at": "2024-01-01T00:00:00Z",
      "updated_at": "2024-01-01T00:00:00Z",
      "quest": {
        "id": 5,
        "external_id": "quest_005",
        "name": "Quest Name",
        ...
      }
    }
  ]
}
```

##### 16. Update Quest Progress
**Endpoint:** `PUT /progress/quests/:quest_id`

**Description:** Updates quest completion status for the authenticated user.

**Path Parameters:**
- `quest_id`: Quest external_id (e.g., "ss1", not the internal database ID)

**Request Body:**
```json
{
  "completed": true
}
```

**Response:**
```json
{
  "id": 1,
  "user_id": 1,
  "quest_id": 5,
  "completed": true,
  "created_at": "2024-01-01T00:00:00Z",
  "updated_at": "2024-01-01T00:00:00Z"
}
```

**Example:**
```
PUT /api/v1/progress/quests/ss1
Content-Type: application/json
Authorization: Bearer <jwt-token>

{
  "completed": true
}
```

#### Hideout Module Progress

##### 17. Get My Hideout Module Progress
**Endpoint:** `GET /progress/hideout-modules`

**Description:** Returns all hideout module progress for the authenticated user.

**Response:**
```json
{
  "data": [
    {
      "id": 1,
      "user_id": 1,
      "hideout_module_id": 3,
      "unlocked": true,
      "level": 5,
      "created_at": "2024-01-01T00:00:00Z",
      "updated_at": "2024-01-01T00:00:00Z",
      "hideout_module": {
        "id": 3,
        "external_id": "module_003",
        "name": "Module Name",
        ...
      }
    }
  ]
}
```

##### 18. Update Hideout Module Progress
**Endpoint:** `PUT /progress/hideout-modules/:module_id`

**Description:** Updates hideout module progress (unlocked status and level) for the authenticated user.

**Path Parameters:**
- `module_id`: Hideout module external_id (e.g., "module_001", not the internal database ID)

**Request Body:**
```json
{
  "unlocked": true,
  "level": 5
}
```

**Response:**
```json
{
  "id": 1,
  "user_id": 1,
  "hideout_module_id": 3,
  "unlocked": true,
  "level": 5,
  "created_at": "2024-01-01T00:00:00Z",
  "updated_at": "2024-01-01T00:00:00Z"
}
```

**Example:**
```
PUT /api/v1/progress/hideout-modules/module_001
Content-Type: application/json
Authorization: Bearer <jwt-token>

{
  "unlocked": true,
  "level": 5
}
```

#### Skill Node Progress

##### 19. Get My Skill Node Progress
**Endpoint:** `GET /progress/skill-nodes`

**Description:** Returns all skill node progress for the authenticated user.

**Response:**
```json
{
  "data": [
    {
      "id": 1,
      "user_id": 1,
      "skill_node_id": 10,
      "unlocked": true,
      "created_at": "2024-01-01T00:00:00Z",
      "updated_at": "2024-01-01T00:00:00Z",
      "skill_node": {
        "id": 10,
        "external_id": "skill_010",
        "name": "Skill Node Name",
        ...
      }
    }
  ]
}
```

##### 20. Update Skill Node Progress
**Endpoint:** `PUT /progress/skill-nodes/:skill_node_id`

**Description:** Updates skill node unlock status for the authenticated user.

**Path Parameters:**
- `skill_node_id`: Skill node external_id (e.g., "skill_001", not the internal database ID)

**Request Body:**
```json
{
  "unlocked": true
}
```

**Response:**
```json
{
  "id": 1,
  "user_id": 1,
  "skill_node_id": 10,
  "unlocked": true,
  "created_at": "2024-01-01T00:00:00Z",
  "updated_at": "2024-01-01T00:00:00Z"
}
```

**Example:**
```
PUT /api/v1/progress/skill-nodes/skill_001
Content-Type: application/json
Authorization: Bearer <jwt-token>

{
  "unlocked": true
}
```

---

### Write Endpoints (Admin Only)

**Authentication:** `Authorization: Bearer <jwt-token>`

**Note:** Only users with `role: "admin"` can access these endpoints.

#### Quests

##### 21. Create Quest
**Endpoint:** `POST /quests`

**Description:** Creates a new quest.

**Request Body:**
```json
{
  "external_id": "quest_001",
  "name": "Quest Name",
  "description": "Quest description",
  "trader": "Trader Name",
  "objectives": {...},
  "reward_item_ids": {...},
  "xp": 100,
  "data": {...}
}
```

**Response:** Created quest object

##### 22. Update Quest
**Endpoint:** `PUT /quests/:id`

**Description:** Updates an existing quest.

**Path Parameters:**
- `id`: Quest ID

**Request Body:** Same as Create Quest

**Response:** Updated quest object

##### 23. Delete Quest
**Endpoint:** `DELETE /quests/:id`

**Description:** Deletes a quest.

**Path Parameters:**
- `id`: Quest ID

**Response:** `204 No Content`

#### Items

##### 24. Create Item
**Endpoint:** `POST /items`

**Description:** Creates a new item.

##### 25. Update Item
**Endpoint:** `PUT /items/:id`

**Description:** Updates an existing item.

##### 26. Delete Item
**Endpoint:** `DELETE /items/:id`

**Description:** Deletes an item.

#### Skill Nodes

##### 27. Create Skill Node
**Endpoint:** `POST /skill-nodes`

**Description:** Creates a new skill node.

##### 28. Update Skill Node
**Endpoint:** `PUT /skill-nodes/:id`

**Description:** Updates an existing skill node.

##### 29. Delete Skill Node
**Endpoint:** `DELETE /skill-nodes/:id`

**Description:** Deletes a skill node.

#### Hideout Modules

##### 30. Create Hideout Module
**Endpoint:** `POST /hideout-modules`

**Description:** Creates a new hideout module.

##### 31. Update Hideout Module
**Endpoint:** `PUT /hideout-modules/:id`

**Description:** Updates an existing hideout module.

##### 32. Delete Hideout Module
**Endpoint:** `DELETE /hideout-modules/:id`

**Description:** Deletes a hideout module.

---

### Admin Management Endpoints (Admin Only)

**Authentication:** `Authorization: Bearer <jwt-token>`

#### API Keys

##### 33. List API Keys
**Endpoint:** `GET /admin/api-keys`

**Description:** Returns all API keys in the system.

**Response:**
```json
[
  {
    "id": 1,
    "user_id": 1,
    "name": "My API Key",
    "key_hash": "...",
    "revoked_at": null,
    "last_used_at": "2024-01-01T00:00:00Z",
    "created_at": "2024-01-01T00:00:00Z",
    "user": {...}
  }
]
```

##### 34. Create API Key
**Endpoint:** `POST /admin/api-keys`

**Description:** Creates a new API key for a user.

**Request Body:**
```json
{
  "user_id": 1,
  "name": "My API Key"
}
```

**Response:**
```json
{
  "api_key": "generated-api-key-string",
  "name": "My API Key",
  "warning": "Save this key now, you won't be able to see it again."
}
```

##### 35. Revoke API Key
**Endpoint:** `DELETE /admin/api-keys/:id`

**Description:** Revokes an API key.

**Path Parameters:**
- `id`: API key ID

**Response:**
```json
{
  "message": "API key revoked successfully"
}
```

#### JWT Tokens

##### 36. List JWT Tokens
**Endpoint:** `GET /admin/jwts`

**Description:** Returns all active JWT tokens.

##### 37. Revoke JWT Token
**Endpoint:** `POST /admin/jwts/revoke`

**Description:** Revokes a JWT token.

**Request Body:**
```json
{
  "token": "jwt-token-string"
}
```

#### Audit Logs

##### 38. Query Audit Logs
**Endpoint:** `GET /admin/logs`

**Description:** Returns audit logs with filtering options.

**Query Parameters:**
- `page` (optional): Page number
- `limit` (optional): Items per page
- `api_key_id` (optional): Filter by API key ID
- `jwt_token_id` (optional): Filter by JWT token ID
- `user_id` (optional): Filter by user ID
- `endpoint` (optional): Filter by endpoint
- `method` (optional): Filter by HTTP method
- `start_time` (optional): Filter by start time (ISO 8601)
- `end_time` (optional): Filter by end time (ISO 8601)

**Response:** Paginated list of audit logs

#### Sync Management

##### 39. Force Sync
**Endpoint:** `POST /admin/sync/force`

**Description:** Triggers an immediate data sync from GitHub.

**Response:**
```json
{
  "message": "Sync triggered successfully",
  "status": "running"
}
```

##### 40. Get Sync Status
**Endpoint:** `GET /admin/sync/status`

**Description:** Returns the current sync status.

**Response:**
```json
{
  "is_running": false
}
```

#### User Management

##### 41. List Users
**Endpoint:** `GET /admin/users`

**Description:** Returns a paginated list of all users.

**Query Parameters:**
- `page` (optional): Page number (default: 1)
- `limit` (optional): Items per page (default: 50)

**Response:** Paginated list of users

##### 42. Get User Details
**Endpoint:** `GET /admin/users/:id`

**Description:** Returns detailed information about a user, including their API keys and JWT tokens.

**Path Parameters:**
- `id`: User ID

**Response:**
```json
{
  "user": {
    "id": 1,
    "username": "username",
    "email": "user@example.com",
    "role": "user",
    "can_access_data": true,
    "created_via_app": false,
    ...
  },
  "api_keys": [...],
  "jwt_tokens": [...]
}
```

##### 43. Update User Access
**Endpoint:** `PUT /admin/users/:id/access`

**Description:** Updates a user's data access permission.

**Path Parameters:**
- `id`: User ID

**Request Body:**
```json
{
  "can_access_data": true
}
```

**Response:**
```json
{
  "message": "User access updated successfully",
  "user": {
    "id": 1,
    "can_access_data": true,
    ...
  }
}
```

##### 44. Delete User
**Endpoint:** `DELETE /admin/users/:id`

**Description:** Deletes a user and all associated data.

**Path Parameters:**
- `id`: User ID

**Response:**
```json
{
  "message": "User deleted successfully"
}
```

---

## Access Control Summary

### Basic Users (role: "user")
- ✅ Can read all data (if `can_access_data` is enabled)
- ✅ Can read and update their own progress (quests, hideout modules, skill nodes)
- ❌ Cannot create, update, or delete game data (quests, items, skill nodes, hideout modules)
- ❌ Cannot access admin endpoints

### Admin Users (role: "admin")
- ✅ Can read all data
- ✅ Can create, update, and delete all game data
- ✅ Can read and update their own progress
- ✅ Can access all admin endpoints (API keys, JWT tokens, audit logs, sync, user management)

---

## Error Responses

All endpoints return standard HTTP status codes:

- `200 OK`: Success
- `201 Created`: Resource created successfully
- `204 No Content`: Success (no response body)
- `400 Bad Request`: Invalid request parameters or body
- `401 Unauthorized`: Missing or invalid authentication
- `403 Forbidden`: Insufficient permissions
- `404 Not Found`: Resource not found
- `500 Internal Server Error`: Server error

Error response format:
```json
{
  "error": "Error message description"
}
```

---

## Notes

1. **Pagination**: All list endpoints support pagination with `page` and `limit` query parameters.
2. **Progress Tracking**: Progress endpoints use upsert logic - if a progress record doesn't exist, it's created; if it exists, it's updated.
3. **Authentication**: JWT tokens expire after 72 hours by default (configurable via `JWT_EXPIRY_HOURS`).
4. **Data Sync**: Game data is automatically synced from GitHub every 15 minutes (configurable via `SYNC_CRON`).
5. **Backward Compatibility**: The API maintains backward compatibility with `/missions` endpoints (deprecated, use `/quests` instead).

