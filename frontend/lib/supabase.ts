import { createClient, SupabaseClient } from '@supabase/supabase-js';
import type {
  Quest,
  Item,
  SkillNode,
  HideoutModule,
  EnemyType,
  Alert,
} from '@/types';

// Runtime config cache
let runtimeConfig: { url: string; anonKey: string; enabled: boolean } | null = null;
let configLoadPromise: Promise<void> | null = null;

// Load config from API endpoint at runtime
const loadRuntimeConfig = async (): Promise<{ url: string; anonKey: string; enabled: boolean }> => {
  // If already loading, wait for that promise
  if (configLoadPromise) {
    await configLoadPromise;
    return runtimeConfig || { url: '', anonKey: '', enabled: false };
  }

  // Start loading config
  configLoadPromise = (async () => {
    try {
      const apiUrl = typeof window !== 'undefined' 
        ? window.location.origin 
        : (process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080');
      
      const response = await fetch(`${apiUrl}/api/v1/config`);
      if (response.ok) {
        const data = await response.json();
        runtimeConfig = {
          enabled: data.supabase?.enabled === true,
          url: data.supabase?.url || '',
          anonKey: data.supabase?.anonKey || '',
        };
      } else {
        // Fallback to build-time env vars if API fails
        runtimeConfig = {
          enabled: process.env.NEXT_PUBLIC_SUPABASE_ENABLED === 'true',
          url: process.env.NEXT_PUBLIC_SUPABASE_URL || '',
          anonKey: process.env.NEXT_PUBLIC_SUPABASE_ANON_KEY || '',
        };
      }
    } catch (error) {
      // Fallback to build-time env vars on error
      runtimeConfig = {
        enabled: process.env.NEXT_PUBLIC_SUPABASE_ENABLED === 'true',
        url: process.env.NEXT_PUBLIC_SUPABASE_URL || '',
        anonKey: process.env.NEXT_PUBLIC_SUPABASE_ANON_KEY || '',
      };
    }
  })();

  await configLoadPromise;
  return runtimeConfig || { url: '', anonKey: '', enabled: false };
};

// Get Supabase configuration from environment variables or runtime config
const getSupabaseConfig = async (): Promise<{ url: string; anonKey: string; enabled: boolean }> => {
  // In browser, try to load from API first
  if (typeof window !== 'undefined') {
    return await loadRuntimeConfig();
  }
  
  // Server-side or fallback: use build-time env vars
  return {
    url: process.env.NEXT_PUBLIC_SUPABASE_URL || '',
    anonKey: process.env.NEXT_PUBLIC_SUPABASE_ANON_KEY || '',
    enabled: process.env.NEXT_PUBLIC_SUPABASE_ENABLED === 'true',
  };
};

// Initialize Supabase client (singleton)
let supabaseClient: SupabaseClient | null = null;

export const getSupabaseClient = async (): Promise<SupabaseClient | null> => {
  const { url, anonKey, enabled } = await getSupabaseConfig();

  if (!enabled || !url || !anonKey) {
    return null;
  }

  if (!supabaseClient) {
    supabaseClient = createClient(url, anonKey);
  }

  return supabaseClient;
};

// Check if Supabase is enabled (async version)
export const isSupabaseEnabled = async (): Promise<boolean> => {
  const { enabled, url, anonKey } = await getSupabaseConfig();
  return enabled && !!url && !!anonKey;
};

// Synchronous version for compatibility (uses cached config or build-time env)
export const isSupabaseEnabledSync = (): boolean => {
  if (runtimeConfig) {
    return runtimeConfig.enabled && !!runtimeConfig.url && !!runtimeConfig.anonKey;
  }
  // Fallback to build-time env vars
  const url = process.env.NEXT_PUBLIC_SUPABASE_URL;
  const anonKey = process.env.NEXT_PUBLIC_SUPABASE_ANON_KEY;
  const enabled = process.env.NEXT_PUBLIC_SUPABASE_ENABLED === 'true';
  return enabled && !!url && !!anonKey;
};

// Supabase service for syncing data
class SupabaseService {
  private client: SupabaseClient | null = null;
  private clientPromise: Promise<SupabaseClient | null> | null = null;

  constructor() {
    // Initialize client asynchronously
    this.clientPromise = getSupabaseClient().then((client) => {
      this.client = client;
      return client;
    });
  }

  private async ensureClient(): Promise<SupabaseClient | null> {
    if (this.client) {
      return this.client;
    }
    if (this.clientPromise) {
      return await this.clientPromise;
    }
    this.clientPromise = getSupabaseClient();
    this.client = await this.clientPromise;
    return this.client;
  }

  // Helper to log errors silently (optional - can be removed in production)
  private logError(operation: string, error: any) {
    if (process.env.NODE_ENV === 'development') {
      console.error(`Supabase ${operation} error:`, error);
    }
  }

  // Quest operations
  async syncQuest(quest: Quest | Partial<Quest>, operation: 'insert' | 'update' | 'delete') {
    const client = await this.ensureClient();
    if (!client) return;

    try {
      if (operation === 'delete') {
        await client.from('quests').delete().eq('external_id', (quest as Quest).external_id);
      } else if (operation === 'insert') {
        await client.from('quests').insert({
          external_id: quest.external_id,
          name: quest.name,
          description: quest.description,
          trader: (quest as Quest).trader,
          xp: (quest as Quest).xp,
          objectives: (quest as Quest).objectives,
          reward_item_ids: (quest as Quest).reward_item_ids,
          data: quest.data || {},
        });
      } else if (operation === 'update') {
        await client
          .from('quests')
          .update({
            name: quest.name,
            description: quest.description,
            trader: (quest as Quest).trader,
            xp: (quest as Quest).xp,
            objectives: (quest as Quest).objectives,
            reward_item_ids: (quest as Quest).reward_item_ids,
            data: quest.data || {},
            updated_at: new Date().toISOString(),
          })
          .eq('external_id', quest.external_id);
      }
    } catch (error) {
      this.logError(`quest ${operation}`, error);
    }
  }

  // Item operations
  async syncItem(item: Item | Partial<Item>, operation: 'insert' | 'update' | 'delete') {
    const client = await this.ensureClient();
    if (!client) return;

    try {
      if (operation === 'delete') {
        await client.from('items').delete().eq('external_id', (item as Item).external_id);
      } else if (operation === 'insert') {
        await client.from('items').insert({
          external_id: item.external_id,
          name: item.name,
          description: item.description,
          type: item.type,
          image_url: item.image_url,
          image_filename: item.image_filename,
          data: item.data || {},
        });
      } else if (operation === 'update') {
        await client
          .from('items')
          .update({
            name: item.name,
            description: item.description,
            type: item.type,
            image_url: item.image_url,
            image_filename: item.image_filename,
            data: item.data || {},
            updated_at: new Date().toISOString(),
          })
          .eq('external_id', item.external_id);
      }
    } catch (error) {
      this.logError(`item ${operation}`, error);
    }
  }

  // Skill Node operations
  async syncSkillNode(skillNode: SkillNode | Partial<SkillNode>, operation: 'insert' | 'update' | 'delete') {
    const client = await this.ensureClient();
    if (!client) return;

    try {
      if (operation === 'delete') {
        await client.from('skill_nodes').delete().eq('external_id', (skillNode as SkillNode).external_id);
      } else if (operation === 'insert') {
        await client.from('skill_nodes').insert({
          external_id: skillNode.external_id,
          name: skillNode.name,
          description: skillNode.description,
          impacted_skill: skillNode.impacted_skill,
          category: skillNode.category,
          max_points: skillNode.max_points,
          icon_name: skillNode.icon_name,
          is_major: skillNode.is_major,
          position: skillNode.position,
          known_value: skillNode.known_value,
          prerequisite_node_ids: skillNode.prerequisite_node_ids,
          data: skillNode.data || {},
        });
      } else if (operation === 'update') {
        await client
          .from('skill_nodes')
          .update({
            name: skillNode.name,
            description: skillNode.description,
            impacted_skill: skillNode.impacted_skill,
            category: skillNode.category,
            max_points: skillNode.max_points,
            icon_name: skillNode.icon_name,
            is_major: skillNode.is_major,
            position: skillNode.position,
            known_value: skillNode.known_value,
            prerequisite_node_ids: skillNode.prerequisite_node_ids,
            data: skillNode.data || {},
            updated_at: new Date().toISOString(),
          })
          .eq('external_id', skillNode.external_id);
      }
    } catch (error) {
      this.logError(`skill_node ${operation}`, error);
    }
  }

  // Hideout Module operations
  async syncHideoutModule(module: HideoutModule | Partial<HideoutModule>, operation: 'insert' | 'update' | 'delete') {
    const client = await this.ensureClient();
    if (!client) return;

    try {
      if (operation === 'delete') {
        await client.from('hideout_modules').delete().eq('external_id', (module as HideoutModule).external_id);
      } else if (operation === 'insert') {
        await client.from('hideout_modules').insert({
          external_id: module.external_id,
          name: module.name,
          description: module.description,
          max_level: module.max_level,
          levels: module.levels,
          data: module.data || {},
        });
      } else if (operation === 'update') {
        await client
          .from('hideout_modules')
          .update({
            name: module.name,
            description: module.description,
            max_level: module.max_level,
            levels: module.levels,
            data: module.data || {},
            updated_at: new Date().toISOString(),
          })
          .eq('external_id', module.external_id);
      }
    } catch (error) {
      this.logError(`hideout_module ${operation}`, error);
    }
  }

  // Enemy Type operations
  async syncEnemyType(enemyType: EnemyType | Partial<EnemyType>, operation: 'insert' | 'update' | 'delete') {
    const client = await this.ensureClient();
    if (!client) return;

    try {
      if (operation === 'delete') {
        await client.from('enemy_types').delete().eq('external_id', (enemyType as EnemyType).external_id);
      } else if (operation === 'insert') {
        await client.from('enemy_types').insert({
          external_id: enemyType.external_id,
          name: enemyType.name,
          description: enemyType.description,
          type: enemyType.type,
          image_url: enemyType.image_url,
          image_filename: enemyType.image_filename,
          weakpoints: enemyType.weakpoints,
          data: enemyType.data || {},
        });
      } else if (operation === 'update') {
        await client
          .from('enemy_types')
          .update({
            name: enemyType.name,
            description: enemyType.description,
            type: enemyType.type,
            image_url: enemyType.image_url,
            image_filename: enemyType.image_filename,
            weakpoints: enemyType.weakpoints,
            data: enemyType.data || {},
            updated_at: new Date().toISOString(),
          })
          .eq('external_id', enemyType.external_id);
      }
    } catch (error) {
      this.logError(`enemy_type ${operation}`, error);
    }
  }

  // Alert operations
  async syncAlert(alert: Alert | Partial<Alert>, operation: 'insert' | 'update' | 'delete') {
    const client = await this.ensureClient();
    if (!client) return;

    try {
      const alertId = (alert as Alert).id;
      if (operation === 'delete') {
        // Delete by api_id since that's what we use to match API records
        await client.from('alerts').delete().eq('api_id', alertId);
      } else if (operation === 'insert') {
        await client.from('alerts').insert({
          api_id: alertId, // Store API's id for future lookups
          name: alert.name,
          description: alert.description,
          severity: alert.severity,
          is_active: alert.is_active,
          data: alert.data || {},
        });
      } else if (operation === 'update') {
        await client
          .from('alerts')
          .update({
            name: alert.name,
            description: alert.description,
            severity: alert.severity,
            is_active: alert.is_active,
            data: alert.data || {},
            updated_at: new Date().toISOString(),
          })
          .eq('api_id', alertId); // Match by api_id
      }
    } catch (error) {
      this.logError(`alert ${operation}`, error);
    }
  }

  // Read operations - fetch data from Supabase
  async getQuests(limit = 100): Promise<Quest[]> {
    const client = await this.ensureClient();
    if (!client) return [];
    try {
      const { data, error } = await client.from('quests').select('*').limit(limit).order('created_at', { ascending: false });
      if (error) throw error;
      return (data || []) as Quest[];
    } catch (error) {
      this.logError('getQuests', error);
      return [];
    }
  }

  async getItems(limit = 100): Promise<Item[]> {
    const client = await this.ensureClient();
    if (!client) return [];
    try {
      const { data, error } = await client.from('items').select('*').limit(limit).order('created_at', { ascending: false });
      if (error) throw error;
      return (data || []) as Item[];
    } catch (error) {
      this.logError('getItems', error);
      return [];
    }
  }

  async getSkillNodes(limit = 100): Promise<SkillNode[]> {
    const client = await this.ensureClient();
    if (!client) return [];
    try {
      const { data, error } = await client.from('skill_nodes').select('*').limit(limit).order('created_at', { ascending: false });
      if (error) throw error;
      return (data || []) as SkillNode[];
    } catch (error) {
      this.logError('getSkillNodes', error);
      return [];
    }
  }

  async getHideoutModules(limit = 100): Promise<HideoutModule[]> {
    const client = await this.ensureClient();
    if (!client) return [];
    try {
      const { data, error } = await client.from('hideout_modules').select('*').limit(limit).order('created_at', { ascending: false });
      if (error) throw error;
      return (data || []) as HideoutModule[];
    } catch (error) {
      this.logError('getHideoutModules', error);
      return [];
    }
  }

  async getEnemyTypes(limit = 100): Promise<EnemyType[]> {
    const client = await this.ensureClient();
    if (!client) return [];
    try {
      const { data, error } = await client.from('enemy_types').select('*').limit(limit).order('created_at', { ascending: false });
      if (error) throw error;
      return (data || []) as EnemyType[];
    } catch (error) {
      this.logError('getEnemyTypes', error);
      return [];
    }
  }

  async getAlerts(limit = 100): Promise<Alert[]> {
    const client = await this.ensureClient();
    if (!client) return [];
    try {
      const { data, error } = await client.from('alerts').select('*').limit(limit).order('created_at', { ascending: false });
      if (error) throw error;
      return (data || []) as Alert[];
    } catch (error) {
      this.logError('getAlerts', error);
      return [];
    }
  }

  // Get counts for each table
  async getCounts(): Promise<Record<string, number>> {
    const client = await this.ensureClient();
    if (!client) return {};
    try {
      const [quests, items, skillNodes, hideoutModules, enemyTypes, alerts] = await Promise.all([
        client.from('quests').select('*', { count: 'exact', head: true }),
        client.from('items').select('*', { count: 'exact', head: true }),
        client.from('skill_nodes').select('*', { count: 'exact', head: true }),
        client.from('hideout_modules').select('*', { count: 'exact', head: true }),
        client.from('enemy_types').select('*', { count: 'exact', head: true }),
        client.from('alerts').select('*', { count: 'exact', head: true }),
      ]);

      return {
        quests: quests.count || 0,
        items: items.count || 0,
        skillNodes: skillNodes.count || 0,
        hideoutModules: hideoutModules.count || 0,
        enemyTypes: enemyTypes.count || 0,
        alerts: alerts.count || 0,
      };
    } catch (error) {
      this.logError('getCounts', error);
      return {};
    }
  }
}

export const supabaseService = new SupabaseService();

