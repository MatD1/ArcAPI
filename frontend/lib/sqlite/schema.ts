import { sqliteTable, text, integer, blob } from 'drizzle-orm/sqlite-core';

// Static Collections (Hydrated from Snapshot)

export const quests = sqliteTable('quests', {
  id: integer('id').primaryKey({ autoIncrement: true }),
  external_id: text('external_id').notNull().unique(),
  name: text('name').notNull(),
  description: text('description'),
  trader: text('trader'),
  objectives: blob('objectives', { mode: 'json' }), // JSONB representation
  reward_item_ids: blob('reward_item_ids', { mode: 'json' }),
  xp: integer('xp'),
  data: blob('data', { mode: 'json' }), // Full raw JSON
  synced_at: integer('synced_at', { mode: 'timestamp' }).notNull(),
});

export const items = sqliteTable('items', {
  id: integer('id').primaryKey({ autoIncrement: true }),
  external_id: text('external_id').notNull().unique(),
  name: text('name').notNull(),
  description: text('description'),
  type: text('type'),
  image_url: text('image_url'),
  image_filename: text('image_filename'),
  data: blob('data', { mode: 'json' }),
  synced_at: integer('synced_at', { mode: 'timestamp' }).notNull(),
});

export const skillNodes = sqliteTable('skill_nodes', {
  id: integer('id').primaryKey({ autoIncrement: true }),
  external_id: text('external_id').notNull().unique(),
  name: text('name').notNull(),
  description: text('description'),
  impacted_skill: text('impacted_skill'),
  category: text('category'),
  max_points: integer('max_points'),
  icon_name: text('icon_name'),
  is_major: integer('is_major', { mode: 'boolean' }),
  data: blob('data', { mode: 'json' }),
  synced_at: integer('synced_at', { mode: 'timestamp' }).notNull(),
});

export const hideoutModules = sqliteTable('hideout_modules', {
  id: integer('id').primaryKey({ autoIncrement: true }),
  external_id: text('external_id').notNull().unique(),
  name: text('name').notNull(),
  description: text('description'),
  max_level: integer('max_level'),
  levels: blob('levels', { mode: 'json' }),
  data: blob('data', { mode: 'json' }),
  synced_at: integer('synced_at', { mode: 'timestamp' }).notNull(),
});

export const bots = sqliteTable('bots', {
  id: integer('id').primaryKey({ autoIncrement: true }),
  external_id: text('external_id').notNull().unique(),
  name: text('name').notNull(),
  description: text('description'),
  data: blob('data', { mode: 'json' }),
  synced_at: integer('synced_at', { mode: 'timestamp' }).notNull(),
});

export const maps = sqliteTable('maps', {
  id: integer('id').primaryKey({ autoIncrement: true }),
  external_id: text('external_id').notNull().unique(),
  name: text('name').notNull(),
  description: text('description'),
  data: blob('data', { mode: 'json' }),
  synced_at: integer('synced_at', { mode: 'timestamp' }).notNull(),
});

export const traders = sqliteTable('traders', {
  id: integer('id').primaryKey({ autoIncrement: true }),
  external_id: text('external_id').notNull().unique(),
  name: text('name').notNull(),
  description: text('description'),
  data: blob('data', { mode: 'json' }),
  synced_at: integer('synced_at', { mode: 'timestamp' }).notNull(),
});

export const projects = sqliteTable('projects', {
  id: integer('id').primaryKey({ autoIncrement: true }),
  external_id: text('external_id').notNull().unique(),
  name: text('name').notNull(),
  description: text('description'),
  data: blob('data', { mode: 'json' }),
  synced_at: integer('synced_at', { mode: 'timestamp' }).notNull(),
});

// Offline Outbox (For Progress Sync)

export const outbox = sqliteTable('outbox', {
  id: integer('id').primaryKey({ autoIncrement: true }),
  type: text('type', { enum: ['quest_progress', 'hideout_module_progress', 'skill_node_progress', 'blueprint_progress'] }).notNull(),
  target_id: text('target_id').notNull(), // External ID of the entity
  action: text('action', { enum: ['upsert', 'delete'] }).notNull(),
  payload: blob('payload', { mode: 'json' }).notNull(), // The actual progress data
  created_at: integer('created_at', { mode: 'timestamp' }).notNull(),
  retry_count: integer('retry_count').default(0),
});
