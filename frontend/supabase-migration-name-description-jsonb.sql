-- Migration: Update name and description to JSONB for quests and hideout_modules
-- Run this in your Supabase SQL editor if you already have tables created
-- This preserves existing string data by converting it to JSONB format

-- Migrate quests table
-- Convert TEXT to JSONB: strings become JSON strings, preserving existing data
ALTER TABLE quests 
  ALTER COLUMN name TYPE JSONB USING to_jsonb(name),
  ALTER COLUMN description TYPE JSONB USING CASE 
    WHEN description IS NULL THEN NULL 
    ELSE to_jsonb(description)
  END;

-- Migrate hideout_modules table
ALTER TABLE hideout_modules 
  ALTER COLUMN name TYPE JSONB USING to_jsonb(name),
  ALTER COLUMN description TYPE JSONB USING CASE 
    WHEN description IS NULL THEN NULL 
    ELSE to_jsonb(description)
  END;

-- Note: 
-- - to_jsonb() converts TEXT values to JSONB strings (e.g., "text" becomes '"text"' in JSONB)
-- - This preserves all existing data
-- - After migration, you can store objects/arrays in these fields
-- - If you have existing JSON objects/arrays stored as TEXT, you may need to parse them first

