-- Supabase Schema for Arc Raiders API Frontend Integration
-- Run this SQL in your Supabase SQL editor to create the necessary tables

-- Quests table
CREATE TABLE IF NOT EXISTS quests (
    external_id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
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
    name TEXT NOT NULL,
    description TEXT,
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
-- Note: We store the API's id as api_id since Supabase will generate its own id
CREATE TABLE IF NOT EXISTS alerts (
    id SERIAL PRIMARY KEY,
    api_id INTEGER UNIQUE, -- Store the API's id for syncing
    name TEXT NOT NULL,
    description TEXT,
    severity TEXT DEFAULT 'info' CHECK (severity IN ('info', 'warning', 'error', 'critical')),
    is_active BOOLEAN DEFAULT TRUE,
    data JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Create indexes for better query performance
CREATE INDEX IF NOT EXISTS idx_quests_name ON quests(name);
CREATE INDEX IF NOT EXISTS idx_quests_trader ON quests(trader);
CREATE INDEX IF NOT EXISTS idx_items_name ON items(name);
CREATE INDEX IF NOT EXISTS idx_items_type ON items(type);
CREATE INDEX IF NOT EXISTS idx_skill_nodes_name ON skill_nodes(name);
CREATE INDEX IF NOT EXISTS idx_skill_nodes_category ON skill_nodes(category);
CREATE INDEX IF NOT EXISTS idx_hideout_modules_name ON hideout_modules(name);
CREATE INDEX IF NOT EXISTS idx_enemy_types_name ON enemy_types(name);
CREATE INDEX IF NOT EXISTS idx_enemy_types_type ON enemy_types(type);
CREATE INDEX IF NOT EXISTS idx_alerts_severity ON alerts(severity);
CREATE INDEX IF NOT EXISTS idx_alerts_is_active ON alerts(is_active);
CREATE INDEX IF NOT EXISTS idx_alerts_api_id ON alerts(api_id);

-- Create updated_at trigger function
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create triggers to automatically update updated_at
CREATE TRIGGER update_quests_updated_at BEFORE UPDATE ON quests
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_items_updated_at BEFORE UPDATE ON items
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_skill_nodes_updated_at BEFORE UPDATE ON skill_nodes
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_hideout_modules_updated_at BEFORE UPDATE ON hideout_modules
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_enemy_types_updated_at BEFORE UPDATE ON enemy_types
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_alerts_updated_at BEFORE UPDATE ON alerts
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Enable Row Level Security (RLS) - adjust policies as needed
ALTER TABLE quests ENABLE ROW LEVEL SECURITY;
ALTER TABLE items ENABLE ROW LEVEL SECURITY;
ALTER TABLE skill_nodes ENABLE ROW LEVEL SECURITY;
ALTER TABLE hideout_modules ENABLE ROW LEVEL SECURITY;
ALTER TABLE enemy_types ENABLE ROW LEVEL SECURITY;
ALTER TABLE alerts ENABLE ROW LEVEL SECURITY;

-- Create policies to allow anonymous inserts, updates, and deletes
-- WARNING: These policies allow anyone with the anon key to modify data
-- Adjust these policies based on your security requirements

-- Quests policies
CREATE POLICY "Allow anonymous insert on quests" ON quests
    FOR INSERT WITH CHECK (true);

CREATE POLICY "Allow anonymous update on quests" ON quests
    FOR UPDATE USING (true) WITH CHECK (true);

CREATE POLICY "Allow anonymous delete on quests" ON quests
    FOR DELETE USING (true);

CREATE POLICY "Allow anonymous select on quests" ON quests
    FOR SELECT USING (true);

-- Items policies
CREATE POLICY "Allow anonymous insert on items" ON items
    FOR INSERT WITH CHECK (true);

CREATE POLICY "Allow anonymous update on items" ON items
    FOR UPDATE USING (true) WITH CHECK (true);

CREATE POLICY "Allow anonymous delete on items" ON items
    FOR DELETE USING (true);

CREATE POLICY "Allow anonymous select on items" ON items
    FOR SELECT USING (true);

-- Skill Nodes policies
CREATE POLICY "Allow anonymous insert on skill_nodes" ON skill_nodes
    FOR INSERT WITH CHECK (true);

CREATE POLICY "Allow anonymous update on skill_nodes" ON skill_nodes
    FOR UPDATE USING (true) WITH CHECK (true);

CREATE POLICY "Allow anonymous delete on skill_nodes" ON skill_nodes
    FOR DELETE USING (true);

CREATE POLICY "Allow anonymous select on skill_nodes" ON skill_nodes
    FOR SELECT USING (true);

-- Hideout Modules policies
CREATE POLICY "Allow anonymous insert on hideout_modules" ON hideout_modules
    FOR INSERT WITH CHECK (true);

CREATE POLICY "Allow anonymous update on hideout_modules" ON hideout_modules
    FOR UPDATE USING (true) WITH CHECK (true);

CREATE POLICY "Allow anonymous delete on hideout_modules" ON hideout_modules
    FOR DELETE USING (true);

CREATE POLICY "Allow anonymous select on hideout_modules" ON hideout_modules
    FOR SELECT USING (true);

-- Enemy Types policies
CREATE POLICY "Allow anonymous insert on enemy_types" ON enemy_types
    FOR INSERT WITH CHECK (true);

CREATE POLICY "Allow anonymous update on enemy_types" ON enemy_types
    FOR UPDATE USING (true) WITH CHECK (true);

CREATE POLICY "Allow anonymous delete on enemy_types" ON enemy_types
    FOR DELETE USING (true);

CREATE POLICY "Allow anonymous select on enemy_types" ON enemy_types
    FOR SELECT USING (true);

-- Alerts policies
CREATE POLICY "Allow anonymous insert on alerts" ON alerts
    FOR INSERT WITH CHECK (true);

CREATE POLICY "Allow anonymous update on alerts" ON alerts
    FOR UPDATE USING (true) WITH CHECK (true);

CREATE POLICY "Allow anonymous delete on alerts" ON alerts
    FOR DELETE USING (true);

CREATE POLICY "Allow anonymous select on alerts" ON alerts
    FOR SELECT USING (true);

