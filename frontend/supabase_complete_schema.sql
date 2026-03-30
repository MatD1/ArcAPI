-- Supabase Complete Schema for Arc Raiders API
-- Unified SQL for both Frontend and Backend integration

-- 1. Game Data Tables
-- -------------------

-- Quests table
CREATE TABLE IF NOT EXISTS quests (
    external_id TEXT PRIMARY KEY,
    name JSONB NOT NULL,
    description JSONB,
    trader TEXT,
    xp INTEGER,
    objectives JSONB,
    reward_item_ids JSONB,
    data JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Items table
CREATE TABLE IF NOT EXISTS items (
    external_id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    type TEXT,
    image_url TEXT,
    image_filename TEXT,
    data JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Skill Nodes table
CREATE TABLE IF NOT EXISTS skill_nodes (
    external_id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    impacted_skill TEXT,
    category TEXT,
    max_points INTEGER,
    icon_name TEXT,
    is_major BOOLEAN DEFAULT FALSE,
    position JSONB,
    known_value JSONB,
    prerequisite_node_ids JSONB,
    data JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Hideout Modules table
CREATE TABLE IF NOT EXISTS hideout_modules (
    external_id TEXT PRIMARY KEY,
    name JSONB NOT NULL,
    description JSONB,
    max_level INTEGER,
    levels JSONB,
    data JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Enemy Types table
CREATE TABLE IF NOT EXISTS enemy_types (
    external_id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    type TEXT,
    image_url TEXT,
    image_filename TEXT,
    weakpoints JSONB,
    data JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Alerts table
CREATE TABLE IF NOT EXISTS alerts (
    id SERIAL PRIMARY KEY,
    api_id INTEGER UNIQUE,
    name TEXT NOT NULL,
    description TEXT,
    severity TEXT DEFAULT 'info' CHECK (severity IN ('info', 'warning', 'error', 'critical')),
    is_active BOOLEAN DEFAULT TRUE,
    data JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- 2. User Profiles & Role Management
-- ----------------------------------

-- Profiles table (stores user roles for frontend admin enforcement)
CREATE TABLE IF NOT EXISTS public.profiles (
    id UUID PRIMARY KEY REFERENCES auth.users(id) ON DELETE CASCADE,
    email TEXT,
    role TEXT DEFAULT 'user' CHECK (role IN ('user', 'admin')),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Enable RLS on profiles
ALTER TABLE public.profiles ENABLE ROW LEVEL SECURITY;

-- Policies for profiles
CREATE POLICY "Users can view their own profile" ON public.profiles
    FOR SELECT USING (auth.uid() = id);

-- Trigger to automatically create a profile for new users
CREATE OR REPLACE FUNCTION public.handle_new_user()
RETURNS TRIGGER AS $$
BEGIN
  INSERT INTO public.profiles (id, email, role)
  VALUES (new.id, new.email, 'user');
  RETURN new;
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

CREATE OR REPLACE TRIGGER on_auth_user_created
  AFTER INSERT ON auth.users
  FOR EACH ROW EXECUTE FUNCTION public.handle_new_user();

-- 3. Utility Functions & Automation
-- ---------------------------------

-- Updated_at trigger function
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Apply triggers to all tables
DO $$
DECLARE 
    t TEXT;
BEGIN
    FOR t IN 
        SELECT table_name 
        FROM information_schema.tables 
        WHERE table_schema = 'public' 
        AND table_name IN ('quests', 'items', 'skill_nodes', 'hideout_modules', 'enemy_types', 'alerts', 'profiles')
    LOOP
        EXECUTE format('DROP TRIGGER IF EXISTS update_%I_updated_at ON %I', t, t);
        EXECUTE format('CREATE TRIGGER update_%I_updated_at BEFORE UPDATE ON %I FOR EACH ROW EXECUTE FUNCTION update_updated_at_column()', t, t);
    END LOOP;
END;
$$;

-- 4. Sample Admin Setup (Optional)
-- --------------------------------
-- To manually promote a user to admin in the Supabase SQL editor:
-- UPDATE public.profiles SET role = 'admin' WHERE email = 'your-email@example.com';
