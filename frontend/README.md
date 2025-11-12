# Arc Raiders API - Frontend

Next.js frontend for managing the Arc Raiders API. This frontend is integrated into the Go API server and accessible at `/dashboard`.

## Integration

The frontend is built as a static export and served directly by the Go API server. This means:
- Frontend and API are served from the same domain/port
- No CORS issues
- Single deployment
- Access the dashboard at `http://your-api-url/dashboard`

## Features

- **Authentication**: Login with API key
- **Dashboard**: Overview of all entities
- **CRUD Operations**: Full Create, Read, Update, Delete for:
  - Missions
  - Items
  - Skill Nodes
  - Hideout Modules
- **Management**:
  - API Key management (create, revoke, view)
  - JWT Token management
  - Audit Logs viewer with filtering
- **Supabase Integration** (Optional): Automatically syncs data to Supabase when creating, updating, or deleting entities

## Setup

1. Install dependencies:
```bash
cd frontend
npm install
```

2. Copy environment file:
```bash
cp .env.local.example .env.local
```

3. Update `.env.local` with your API URL and optional Supabase configuration:
```
NEXT_PUBLIC_API_URL=http://localhost:8080
# Or for production:
# NEXT_PUBLIC_API_URL=https://your-app.railway.app

# Optional: Supabase integration (set to 'true' to enable)
NEXT_PUBLIC_SUPABASE_ENABLED=false
NEXT_PUBLIC_SUPABASE_URL=https://your-project.supabase.co
NEXT_PUBLIC_SUPABASE_ANON_KEY=your-anon-key
```

4. Run development server:
```bash
npm run dev
```

5. Open [http://localhost:3000](http://localhost:3000)

## Usage

1. Login with your API key (you'll need to create one via the management API first, or have an admin create it for you)
2. Navigate through the dashboard to manage entities
3. Use the CRUD pages to create, edit, and delete records
4. View audit logs to monitor API usage
5. Manage API keys and JWT tokens (admin only)

## Project Structure

```
frontend/
├── app/                    # Next.js app router pages
│   ├── login/             # Login page
│   ├── dashboard/          # Dashboard
│   ├── missions/           # Missions CRUD
│   ├── items/              # Items CRUD
│   ├── skill-nodes/        # Skill Nodes CRUD
│   ├── hideout-modules/    # Hideout Modules CRUD
│   ├── api-keys/          # API Key management
│   ├── jwt-tokens/        # JWT Token management
│   └── logs/               # Audit logs viewer
├── components/             # React components
│   ├── layout/            # Layout components
│   ├── crud/              # CRUD components
│   └── ui/                # UI components
├── lib/                   # Utilities
│   ├── api.ts             # API client
│   └── utils.ts           # Helper functions
├── store/                 # State management (Zustand)
├── types/                 # TypeScript types
└── public/                # Static assets
```

## Building for Production

```bash
npm run build
npm start
```

## Deploying

The frontend can be deployed to:
- **Vercel** (recommended for Next.js)
- **Railway**
- **Any Node.js hosting**

Make sure to set the `NEXT_PUBLIC_API_URL` environment variable to your API URL.

## Supabase Integration

The frontend can optionally sync data to a Supabase database. When enabled, all create, update, and delete operations will automatically sync to your Supabase database.

### Setup

1. Create a Supabase project at [supabase.com](https://supabase.com)
2. Get your project URL and anon key from the Supabase dashboard
3. Create tables in Supabase by running the SQL schema file:
   - Open the Supabase SQL editor
   - Run the SQL from `supabase-schema.sql` file in the frontend directory
   - This will create all necessary tables, indexes, triggers, and Row Level Security policies

4. Set environment variables in `.env.local`:
   ```
   NEXT_PUBLIC_SUPABASE_ENABLED=true
   NEXT_PUBLIC_SUPABASE_URL=https://your-project.supabase.co
   NEXT_PUBLIC_SUPABASE_ANON_KEY=your-anon-key
   ```

5. Restart your development server

### Notes

- Supabase sync is optional and will not affect the main API if it fails
- Data is synced after successful API operations
- The sync uses `external_id` as the primary key for most entities (quests, items, skill_nodes, hideout_modules, enemy_types)
- Alerts use `api_id` to store the API's id for matching (since Supabase generates its own id)
- Supabase sync errors are logged in development mode but fail silently in production
- Make sure your Supabase Row Level Security (RLS) policies allow the operations you need (the schema includes permissive policies for testing - adjust for production)

