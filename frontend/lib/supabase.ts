import { createClient, SupabaseClient } from '@supabase/supabase-js';
import type {
  Quest,
  Item,
  SkillNode,
  HideoutModule,
  EnemyType,
  Alert,
} from '@/types';

// Get Supabase configuration from environment variables
const getSupabaseConfig = () => {
  const url = process.env.NEXT_PUBLIC_SUPABASE_URL;
  const anonKey = process.env.NEXT_PUBLIC_SUPABASE_ANON_KEY;
  const enabled = process.env.NEXT_PUBLIC_SUPABASE_ENABLED === 'true';

  return { url, anonKey, enabled };
};

// Initialize Supabase client (singleton)
let supabaseClient: SupabaseClient | null = null;

export const getSupabaseClient = (): SupabaseClient | null => {
  const { url, anonKey, enabled } = getSupabaseConfig();

  if (!enabled || !url || !anonKey) {
    return null;
  }

  if (!supabaseClient) {
    supabaseClient = createClient(url, anonKey);
  }

  return supabaseClient;
};

// Check if Supabase is enabled
export const isSupabaseEnabled = (): boolean => {
  const { enabled, url, anonKey } = getSupabaseConfig();
  return enabled === 'true' && !!url && !!anonKey;
};

// Supabase service for syncing data
class SupabaseService {
  private client: SupabaseClient | null;

  constructor() {
    this.client = getSupabaseClient();
  }

  // Helper to log errors silently (optional - can be removed in production)
  private logError(operation: string, error: any) {
    if (process.env.NODE_ENV === 'development') {
      console.error(`Supabase ${operation} error:`, error);
    }
  }

  // Quest operations
  async syncQuest(quest: Quest | Partial<Quest>, operation: 'insert' | 'update' | 'delete') {
    if (!this.client) return;

    try {
      if (operation === 'delete') {
        await this.client.from('quests').delete().eq('external_id', (quest as Quest).external_id);
      } else if (operation === 'insert') {
        await this.client.from('quests').insert({
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
        await this.client
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
    if (!this.client) return;

    try {
      if (operation === 'delete') {
        await this.client.from('items').delete().eq('external_id', (item as Item).external_id);
      } else if (operation === 'insert') {
        await this.client.from('items').insert({
          external_id: item.external_id,
          name: item.name,
          description: item.description,
          type: item.type,
          image_url: item.image_url,
          image_filename: item.image_filename,
          data: item.data || {},
        });
      } else if (operation === 'update') {
        await this.client
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
    if (!this.client) return;

    try {
      if (operation === 'delete') {
        await this.client.from('skill_nodes').delete().eq('external_id', (skillNode as SkillNode).external_id);
      } else if (operation === 'insert') {
        await this.client.from('skill_nodes').insert({
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
        await this.client
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
    if (!this.client) return;

    try {
      if (operation === 'delete') {
        await this.client.from('hideout_modules').delete().eq('external_id', (module as HideoutModule).external_id);
      } else if (operation === 'insert') {
        await this.client.from('hideout_modules').insert({
          external_id: module.external_id,
          name: module.name,
          description: module.description,
          max_level: module.max_level,
          levels: module.levels,
          data: module.data || {},
        });
      } else if (operation === 'update') {
        await this.client
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
    if (!this.client) return;

    try {
      if (operation === 'delete') {
        await this.client.from('enemy_types').delete().eq('external_id', (enemyType as EnemyType).external_id);
      } else if (operation === 'insert') {
        await this.client.from('enemy_types').insert({
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
        await this.client
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
    if (!this.client) return;

    try {
      const alertId = (alert as Alert).id;
      if (operation === 'delete') {
        // Delete by api_id since that's what we use to match API records
        await this.client.from('alerts').delete().eq('api_id', alertId);
      } else if (operation === 'insert') {
        await this.client.from('alerts').insert({
          api_id: alertId, // Store API's id for future lookups
          name: alert.name,
          description: alert.description,
          severity: alert.severity,
          is_active: alert.is_active,
          data: alert.data || {},
        });
      } else if (operation === 'update') {
        await this.client
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
}

export const supabaseService = new SupabaseService();

