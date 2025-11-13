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
    if (process.env.NODE_ENV === 'development') {
      console.log('Supabase not enabled:', { enabled, url: url ? 'set' : 'not set', anonKey: anonKey ? 'set' : 'not set' });
    }
    return null;
  }

  if (!supabaseClient) {
    if (process.env.NODE_ENV === 'development') {
      console.log('Creating Supabase client with URL:', url);
    }
    supabaseClient = createClient(url, anonKey, {
      auth: {
        persistSession: false, // Don't persist auth sessions
      },
    });
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
    // Always log errors in development, and in production for debugging
    console.error(`Supabase ${operation} error:`, error);
    if (error?.message) {
      console.error(`Error message:`, error.message);
    }
    if (error?.details) {
      console.error(`Error details:`, error.details);
    }
  }

  private normalizeJsonValue(value: any) {
    if (value === undefined || value === null) {
      return null;
    }
    if (typeof value === 'string' && value.length === 0) {
      return null;
    }
    return value;
  }

  private normalizeNumberValue(value: any) {
    if (value === undefined || value === null || value === '') {
      return null;
    }
    if (typeof value === 'number' && Number.isFinite(value)) {
      return value;
    }
    if (typeof value === 'string') {
      const parsed = Number(value);
      return Number.isFinite(parsed) ? parsed : null;
    }
    return null;
  }

  private normalizeBooleanValue(value: any) {
    if (value === undefined || value === null || value === '') {
      return null;
    }
    if (typeof value === 'boolean') {
      return value;
    }
    if (typeof value === 'number') {
      if (value === 1) return true;
      if (value === 0) return false;
    }
    if (typeof value === 'string') {
      const normalized = value.trim().toLowerCase();
      if (normalized === 'true') return true;
      if (normalized === 'false') return false;
      if (normalized === '1') return true;
      if (normalized === '0') return false;
    }
    return null;
  }

  private unwrapNestedField(value: any, key: string) {
    if (value === undefined || value === null) {
      return null;
    }
    if (Array.isArray(value)) {
      return value;
    }
    if (typeof value === 'object' && key in value) {
      return value[key];
    }
    return value;
  }

  private buildQuestPayload(quest: Quest | Partial<Quest>): Record<string, any> {
    const externalId = quest.external_id as string;
    const typedQuest = quest as Quest;
    const questData = ((typedQuest?.data ?? {}) as Record<string, any>) || {};

    const nameCandidate =
      questData?.name !== undefined ? questData.name : typedQuest?.name;
    const normalizedName = this.normalizeJsonValue(nameCandidate);
    const fallbackName = externalId || typedQuest?.name || 'Unknown Quest';

    const descriptionCandidate =
      questData?.description !== undefined ? questData.description : typedQuest?.description;
    const normalizedDescription = this.normalizeJsonValue(descriptionCandidate);

    const traderValue = typedQuest?.trader ?? questData?.trader ?? null;
    const xpCandidate =
      questData?.xp ??
      questData?.XP ??
      questData?.experience ??
      typedQuest?.xp;
    const normalizedXP =
      xpCandidate === undefined || xpCandidate === null ? null : String(xpCandidate);

    const objectivesCandidate =
      questData?.objectives !== undefined
        ? questData.objectives
        : this.unwrapNestedField(typedQuest?.objectives, 'objectives');
    const normalizedObjectives = this.normalizeJsonValue(objectivesCandidate);

    const rewardItemsCandidate =
      questData?.rewardItemIds !== undefined
        ? questData.rewardItemIds
        : questData?.reward_item_ids !== undefined
          ? questData.reward_item_ids
          : this.unwrapNestedField(typedQuest?.reward_item_ids, 'reward_item_ids');
    const normalizedRewardItems = this.normalizeJsonValue(rewardItemsCandidate);

    const dataValue =
      typedQuest?.data && typeof typedQuest.data === 'object' ? typedQuest.data : {};

    return {
      external_id: externalId,
      name: normalizedName ?? fallbackName,
      description: normalizedDescription,
      trader: traderValue,
      xp: normalizedXP,
      objectives: normalizedObjectives,
      reward_item_ids: normalizedRewardItems,
      data: dataValue,
    };
  }

  private buildSkillNodePayload(skillNode: SkillNode | Partial<SkillNode>): Record<string, any> {
    const externalId = skillNode.external_id as string;
    const typedSkillNode = skillNode as SkillNode;
    const nodeData = ((typedSkillNode?.data ?? {}) as Record<string, any>) || {};

    const nameCandidate = nodeData?.name ?? typedSkillNode?.name;
    const normalizedName = this.normalizeJsonValue(nameCandidate);
    const fallbackName = externalId || typedSkillNode?.name || 'Unknown Skill Node';

    const descriptionCandidate = nodeData?.description ?? typedSkillNode?.description;
    const normalizedDescription = this.normalizeJsonValue(descriptionCandidate);

    const impactedSkill =
      typedSkillNode?.impacted_skill ??
      nodeData?.impacted_skill ??
      nodeData?.impactedSkill ??
      null;

    const categoryValue = typedSkillNode?.category ?? nodeData?.category ?? null;

    const maxPointsCandidate =
      typedSkillNode?.max_points ??
      nodeData?.max_points ??
      nodeData?.maxPoints;
    const normalizedMaxPoints = this.normalizeNumberValue(maxPointsCandidate);

    const iconNameValue =
      typedSkillNode?.icon_name ??
      nodeData?.icon_name ??
      nodeData?.iconName ??
      null;

    const isMajorCandidate =
      typedSkillNode?.is_major ??
      nodeData?.is_major ??
      nodeData?.isMajor;
    const normalizedIsMajor = this.normalizeBooleanValue(isMajorCandidate);

    const positionCandidate =
      nodeData?.position !== undefined ? nodeData.position : typedSkillNode?.position;
    const normalizedPosition = this.normalizeJsonValue(positionCandidate);

    const knownValueCandidate =
      nodeData?.knownValue !== undefined
        ? nodeData.knownValue
        : nodeData?.known_value !== undefined
          ? nodeData.known_value
          : this.unwrapNestedField(typedSkillNode?.known_value, 'known_value');
    const normalizedKnownValue = this.normalizeJsonValue(knownValueCandidate);

    const prereqCandidate =
      nodeData?.prerequisiteNodeIds !== undefined
        ? nodeData.prerequisiteNodeIds
        : nodeData?.prerequisite_node_ids !== undefined
          ? nodeData.prerequisite_node_ids
          : this.unwrapNestedField(typedSkillNode?.prerequisite_node_ids, 'prerequisite_node_ids');
    const normalizedPrereqs = this.normalizeJsonValue(prereqCandidate);

    const dataValue =
      typedSkillNode?.data && typeof typedSkillNode.data === 'object' ? typedSkillNode.data : {};

    return {
      external_id: externalId,
      name: normalizedName ?? fallbackName,
      description: normalizedDescription,
      impacted_skill: impactedSkill,
      category: categoryValue,
      max_points: normalizedMaxPoints,
      icon_name: iconNameValue,
      is_major: normalizedIsMajor ?? false,
      position: normalizedPosition,
      known_value: normalizedKnownValue,
      prerequisite_node_ids: normalizedPrereqs,
      data: dataValue,
    };
  }

  private buildHideoutModulePayload(module: HideoutModule | Partial<HideoutModule>): Record<string, any> {
    const externalId = module.external_id as string;
    const typedModule = module as HideoutModule;
    const moduleData = ((typedModule?.data ?? {}) as Record<string, any>) || {};

    const nameCandidate = moduleData?.name ?? typedModule?.name;
    const normalizedName = this.normalizeJsonValue(nameCandidate);
    const fallbackName = externalId || typedModule?.name || 'Unknown Hideout Module';

    const descriptionCandidate = moduleData?.description ?? typedModule?.description;
    const normalizedDescription = this.normalizeJsonValue(descriptionCandidate);

    const maxLevelCandidate =
      typedModule?.max_level ??
      moduleData?.max_level ??
      moduleData?.maxLevel;
    const normalizedMaxLevel = this.normalizeNumberValue(maxLevelCandidate);

    const levelsCandidate =
      moduleData?.levels !== undefined
        ? moduleData.levels
        : typedModule?.levels !== undefined
          ? typedModule.levels
          : null;
    const normalizedLevels = this.normalizeJsonValue(levelsCandidate);

    const dataValue =
      typedModule?.data && typeof typedModule.data === 'object' ? typedModule.data : {};

    return {
      external_id: externalId,
      name: normalizedName ?? fallbackName,
      description: normalizedDescription,
      max_level: normalizedMaxLevel,
      levels: normalizedLevels,
      data: dataValue,
    };
  }

  // Quest operations
  async syncQuest(quest: Quest | Partial<Quest>, operation: 'insert' | 'update' | 'delete') {
    const client = await this.ensureClient();
    if (!client) return;
    if (!quest?.external_id) {
      console.warn('Supabase quest sync skipped: missing external_id');
      return;
    }

    const payload = operation === 'delete' ? null : this.buildQuestPayload(quest);

    try {
      if (operation === 'delete') {
        await client.from('quests').delete().eq('external_id', quest.external_id);
      } else if (payload) {
        await client
          .from('quests')
          .upsert(payload, { onConflict: 'external_id' });
      }
    } catch (error) {
      this.logError(`quest ${operation}`, error);
    }
  }

  // Item operations
  async syncItem(item: Item | Partial<Item>, operation: 'insert' | 'update' | 'delete') {
    const client = await this.ensureClient();
    if (!client) return;
    if (!item?.external_id) {
      console.warn('Supabase item sync skipped: missing external_id');
      return;
    }

    const payload =
      operation === 'delete'
        ? null
        : {
            external_id: item.external_id,
            name: item.name,
            description: item.description,
            type: item.type,
            image_url: item.image_url,
            image_filename: item.image_filename,
            data: item.data || {},
          };

    try {
      if (operation === 'delete') {
        await client.from('items').delete().eq('external_id', item.external_id);
      } else if (payload) {
        await client
          .from('items')
          .upsert(payload, { onConflict: 'external_id' });
      }
    } catch (error) {
      this.logError(`item ${operation}`, error);
    }
  }

  // Skill Node operations
  async syncSkillNode(skillNode: SkillNode | Partial<SkillNode>, operation: 'insert' | 'update' | 'delete') {
    const client = await this.ensureClient();
    if (!client) return;
    if (!skillNode?.external_id) {
      console.warn('Supabase skill node sync skipped: missing external_id');
      return;
    }
    const payload = this.buildSkillNodePayload(skillNode);

    try {
      if (operation === 'delete') {
        await client.from('skill_nodes').delete().eq('external_id', skillNode.external_id);
      } else {
        await client
          .from('skill_nodes')
          .upsert(payload, { onConflict: 'external_id' });
      }
    } catch (error) {
      this.logError(`skill_node ${operation}`, error);
    }
  }

  // Hideout Module operations
  async syncHideoutModule(module: HideoutModule | Partial<HideoutModule>, operation: 'insert' | 'update' | 'delete') {
    const client = await this.ensureClient();
    if (!client) return;
    if (!module?.external_id) {
      console.warn('Supabase hideout module sync skipped: missing external_id');
      return;
    }
    const payload = this.buildHideoutModulePayload(module);

    try {
      if (operation === 'delete') {
        await client.from('hideout_modules').delete().eq('external_id', module.external_id);
      } else {
        await client
          .from('hideout_modules')
          .upsert(payload, { onConflict: 'external_id' });
      }
    } catch (error) {
      this.logError(`hideout_module ${operation}`, error);
    }
  }

  // Enemy Type operations
  async syncEnemyType(enemyType: EnemyType | Partial<EnemyType>, operation: 'insert' | 'update' | 'delete') {
    const client = await this.ensureClient();
    if (!client) return;
    if (!enemyType?.external_id) {
      console.warn('Supabase enemy type sync skipped: missing external_id');
      return;
    }

    const payload =
      operation === 'delete'
        ? null
        : {
            external_id: enemyType.external_id,
            name: enemyType.name,
            description: enemyType.description,
            type: enemyType.type,
            image_url: enemyType.image_url,
            image_filename: enemyType.image_filename,
            weakpoints: enemyType.weakpoints,
            data: enemyType.data || {},
          };

    try {
      if (operation === 'delete') {
        await client.from('enemy_types').delete().eq('external_id', enemyType.external_id);
      } else if (payload) {
        await client
          .from('enemy_types')
          .upsert(payload, { onConflict: 'external_id' });
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
      } else {
        await client
          .from('alerts')
          .upsert(
            {
              api_id: alertId, // Store API's id for future lookups
              name: alert.name,
              description: alert.description,
              severity: alert.severity,
              is_active: alert.is_active,
              data: alert.data || {},
            },
            { onConflict: 'api_id' }
          );
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

