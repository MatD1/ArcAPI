# Arc Raiders REST API

A fast, secure REST API built in Go that synchronizes and serves game data from the `MatD1/arcraiders-data-fork` repository.

## Features

- **Data Synchronization**: Automatically syncs JSON data from GitHub repository on a configurable schedule
- **CRUD Operations**: Full CRUD API for Missions, Items, Skill Nodes, and Hideout Modules
- **Dual Authentication**: Requires both API keys and JWT tokens for all protected endpoints
- **OAuth Integration**: Optional GitHub OAuth for user authentication
- **Comprehensive Logging**: All requests are logged with API key/JWT information for auditing
- **Management API**: Admin endpoints for managing API keys, JWT tokens, and querying logs
- **Web Dashboard**: Built-in Next.js frontend accessible at `/dashboard` for managing all data
- **High Performance**: Uses Go concurrency features and optional Redis caching
- **PostgreSQL**: Persistent storage with proper indexing
- **Docker Support**: Complete Docker Compose setup for local development

## Prerequisites

- Go 1.24 or higher
- Node.js 18+ and npm (for frontend)
- PostgreSQL 12 or higher
- Redis (optional, for caching)
- Docker and Docker Compose (optional)

## Installation

### Local Development

1. Clone the repository:
```bash
git clone <repository-url>
cd ArcAPI
```

2. Copy the example environment file:
```bash
cp .env.example .env
```

3. Update `.env` with your configuration:
   - Set `DB_PASSWORD` for PostgreSQL
   - Set `JWT_SECRET` (use a secure random string)
   - Set `GITHUB_CLIENT_ID` and `GITHUB_CLIENT_SECRET` if using OAuth

4. Install dependencies:
```bash
go mod download
cd frontend && npm install && cd ..
```

5. Build the frontend:
```bash
make build-frontend
# Or manually:
cd frontend && npm run build && cd ..
```

6. Start PostgreSQL and Redis (if using Docker Compose):
```bash
docker-compose up -d postgres redis
```

7. Run the application:
```bash
go run cmd/server/main.go
# Or make run
```

The API will be available at `http://localhost:8080`
The Dashboard will be available at `http://localhost:8080/dashboard`

### Docker Compose

1. Copy `.env.example` to `.env` and configure it:
```bash
cp .env.example .env
```

2. Build and start all services:
```bash
docker-compose build
docker-compose up -d
```

The Dockerfile will automatically build the frontend during the image build process.

## Configuration

All configuration is done via environment variables. See `.env.example` for all available options.

### Key Configuration Options

- `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`: PostgreSQL connection details
- `REDIS_ADDR`, `REDIS_PASSWORD`: Redis connection (optional)
- `JWT_SECRET`: Secret key for JWT signing (required)
- `JWT_EXPIRY_HOURS`: JWT token expiration time (default: 72)
- `GITHUB_CLIENT_ID`, `GITHUB_CLIENT_SECRET`: GitHub OAuth credentials
- `OAUTH_ENABLED`: Enable/disable OAuth (default: true)
- `SYNC_CRON`: Cron expression for sync schedule (default: `*/15 * * * *` = every 15 minutes)
- `PORT`: Server port (default: 8080, Railway uses PORT env var)
- `LOG_LEVEL`: Logging level (debug, info, warn, error)

## Web Dashboard

Access the integrated web dashboard at `/dashboard` after starting the server.

Features:
- **Login**: Use your API key to authenticate
- **Dashboard**: Overview of all entities
- **CRUD Operations**: Full Create, Read, Update, Delete interfaces for:
  - Missions
  - Items
  - Skill Nodes
  - Hideout Modules
- **Management**:
  - API Key management (create, revoke, view)
  - JWT Token management
  - Audit Logs viewer with filtering

The dashboard is built with Next.js and automatically detects the API URL.

## API Endpoints

See `docs/api.md` for complete API documentation.

### Authentication Endpoints

- `GET /api/v1/auth/github/login` - Initiate GitHub OAuth flow
- `GET /api/v1/auth/github/callback` - OAuth callback handler
- `POST /api/v1/auth/login` - Login with API key and get JWT token

### Data Endpoints (Require API Key + JWT)

All data endpoints require both an API key (header: `X-API-Key`) and a JWT token (header: `Authorization: Bearer <token>`).

#### Missions
- `GET /api/v1/missions` - List missions (paginated)
- `GET /api/v1/missions/:id` - Get mission by ID
- `POST /api/v1/missions` - Create mission
- `PUT /api/v1/missions/:id` - Update mission
- `DELETE /api/v1/missions/:id` - Delete mission

Similar endpoints for `/items`, `/skill-nodes`, `/hideout-modules`.

### Management Endpoints (Admin Only)

- `POST /api/v1/admin/api-keys` - Create API key
- `GET /api/v1/admin/api-keys` - List API keys
- `DELETE /api/v1/admin/api-keys/:id` - Revoke API key
- `POST /api/v1/admin/jwts/revoke` - Revoke JWT token
- `GET /api/v1/admin/jwts` - List active JWT tokens
- `GET /api/v1/admin/logs` - Query audit logs with filters

### Health Check

- `GET /health` - Health check endpoint

## Authentication Flow

1. **Via OAuth (if enabled)**:
   - Visit `/api/v1/auth/github/login` to initiate OAuth
   - After GitHub authentication, you'll receive a JWT token
   - Create an API key via the management API (requires admin role initially)

2. **Via API Key**:
   - Login with API key via `POST /api/v1/auth/login`
   - Receive JWT token in response
   - Use both API key and JWT token for subsequent requests

3. **Via Dashboard**:
   - Visit `/dashboard` in your browser
   - Login with your API key
   - Access all features through the web interface

### Making Authenticated Requests

All protected endpoints require:
- **API Key**: Send in `X-API-Key` header
- **JWT Token**: Send in `Authorization: Bearer <token>` header

Example:
```bash
curl -H "X-API-Key: your-api-key" \
     -H "Authorization: Bearer your-jwt-token" \
     http://localhost:8080/api/v1/missions
```

## Data Synchronization

The sync service automatically fetches JSON data from the `MatD1/arcraiders-data-fork` repository on a schedule defined by `SYNC_CRON`. By default, it runs every 15 minutes.

The sync service:
- Fetches JSON files for missions, items, skill nodes, and hideout modules
- Parses and maps data to the database
- Uses upsert logic (updates existing records or creates new ones)
- Runs concurrently for better performance

## Project Structure

```
ArcAPI/
├── cmd/server/          # Application entry point
├── frontend/            # Next.js frontend (served at /dashboard)
├── internal/
│   ├── config/         # Configuration management
│   ├── models/         # Database models
│   ├── handlers/       # HTTP handlers
│   ├── middleware/     # Middleware (auth, logging)
│   ├── services/       # Business logic
│   └── repository/     # Data access layer
├── migrations/         # SQL migration files
├── tests/             # Test files
├── docs/              # API documentation
├── docker-compose.yml  # Docker Compose setup
└── Dockerfile          # Application container
```

## Development

### Building

```bash
# Build everything (includes frontend)
make build

# Build just frontend
make build-frontend
```

### Running Tests

```bash
go test ./...
```

### Database Migrations

GORM automatically handles migrations on startup. See `migrations/` for SQL reference.

## Security Considerations

1. **API Keys**: Stored as bcrypt hashes, cannot be retrieved once created
2. **JWT Tokens**: Signed with HS256, include expiration
3. **Passwords**: All secrets should use strong, random values
4. **HTTPS**: Use HTTPS in production
5. **Rate Limiting**: Consider adding rate limiting for production

## License

[Add your license here]

## Contributing

[Add contributing guidelines here]
