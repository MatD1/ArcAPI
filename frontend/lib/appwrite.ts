import { Client, Databases, Account, ID, Query } from 'appwrite';
import type {
  Quest,
  Item,
  SkillNode,
  HideoutModule,
  EnemyType,
  Alert,
} from '@/types';

// Runtime config cache
let runtimeConfig: AppwriteConfigShape | null = null;
let configLoadPromise: Promise<void> | null = null;

type AppwriteConfigShape = { endpoint: string; projectId: string; enabled: boolean; databaseId?: string };

const defaultConfig: AppwriteConfigShape = { endpoint: '', projectId: '', enabled: false, databaseId: 'arcapi' };

const loadStaticAppwriteConfig = async (): Promise<AppwriteConfigShape> => {
  try {
    const response = await fetch('/appwrite-config.json', { cache: 'no-store' });
    if (response.ok) {
      const data = await response.json();
      return {
        enabled: data.enabled === true,
        endpoint: data.endpoint || '',
        projectId: data.projectId || '',
        databaseId: data.databaseId || 'arcapi',
      };
    }
  } catch (error) {
    if (process.env.NODE_ENV === 'development') {
      console.warn('Failed to load appwrite-config.json fallback:', error);
    }
  }
  return defaultConfig;
};

// Load config from API endpoint at runtime
const loadRuntimeConfig = async (): Promise<AppwriteConfigShape> => {
  // If already loading, wait for that promise
  if (configLoadPromise) {
    await configLoadPromise;
    return runtimeConfig || defaultConfig;
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
          enabled: data.appwrite?.enabled === true,
          endpoint: data.appwrite?.endpoint || '',
          projectId: data.appwrite?.projectId || '',
          databaseId: data.appwrite?.databaseId || 'arcapi',
        };
      } else {
        runtimeConfig = await loadStaticAppwriteConfig();
      }
    } catch (error) {
      runtimeConfig = await loadStaticAppwriteConfig();
    }

    if (!runtimeConfig?.enabled && process.env.NEXT_PUBLIC_APPWRITE_ENABLED) {
      runtimeConfig = {
        enabled: process.env.NEXT_PUBLIC_APPWRITE_ENABLED === 'true',
        endpoint: process.env.NEXT_PUBLIC_APPWRITE_ENDPOINT || '',
        projectId: process.env.NEXT_PUBLIC_APPWRITE_PROJECT_ID || '',
        databaseId: process.env.NEXT_PUBLIC_APPWRITE_DATABASE_ID || 'arcapi',
      };
    }
  })();

  await configLoadPromise;
  return runtimeConfig || defaultConfig;
};

// Get Appwrite configuration from environment variables or runtime config
const getAppwriteConfig = async (): Promise<AppwriteConfigShape> => {
  // In browser, try to load from API first
  if (typeof window !== 'undefined') {
    return await loadRuntimeConfig();
  }
  
  // Server-side or fallback: use build-time env vars
  return {
    endpoint: process.env.NEXT_PUBLIC_APPWRITE_ENDPOINT || '',
    projectId: process.env.NEXT_PUBLIC_APPWRITE_PROJECT_ID || '',
    enabled: process.env.NEXT_PUBLIC_APPWRITE_ENABLED === 'true',
    databaseId: process.env.NEXT_PUBLIC_APPWRITE_DATABASE_ID || 'arcapi',
  };
};

// Initialize Appwrite client (singleton)
let appwriteClient: Client | null = null;
let databases: Databases | null = null;
let account: Account | null = null;

export const getAppwriteClient = async (): Promise<{ client: Client; databases: Databases; account: Account } | null> => {
  const { endpoint, projectId, enabled } = await getAppwriteConfig();

  if (!enabled || !endpoint || !projectId) {
    if (process.env.NODE_ENV === 'development') {
      console.log('Appwrite not enabled:', { enabled, endpoint: endpoint ? 'set' : 'not set', projectId: projectId ? 'set' : 'not set' });
    }
    return null;
  }

  if (!appwriteClient) {
    if (process.env.NODE_ENV === 'development') {
      console.log('Creating Appwrite client with endpoint:', endpoint);
    }
    appwriteClient = new Client()
      .setEndpoint(endpoint)
      .setProject(projectId);
    
    databases = new Databases(appwriteClient);
    account = new Account(appwriteClient);
  }

  return { client: appwriteClient, databases: databases!, account: account! };
};

// Check if Appwrite is enabled (async version)
export const isAppwriteEnabled = async (): Promise<boolean> => {
  const { enabled, endpoint, projectId } = await getAppwriteConfig();
  return enabled && !!endpoint && !!projectId;
};

  // Synchronous version for compatibility (uses cached config or build-time env)
  export const isAppwriteEnabledSync = (): boolean => {
    if (runtimeConfig) {
      return runtimeConfig.enabled && !!runtimeConfig.endpoint && !!runtimeConfig.projectId;
    }
    // Fallback to build-time env vars (only works if set at build time)
    // In production (Railway), this will be false until runtime config loads
    const endpoint = process.env.NEXT_PUBLIC_APPWRITE_ENDPOINT;
    const projectId = process.env.NEXT_PUBLIC_APPWRITE_PROJECT_ID;
    const enabled = process.env.NEXT_PUBLIC_APPWRITE_ENABLED === 'true';
    return enabled && !!endpoint && !!projectId;
  };

export const signOutOfAppwrite = async (): Promise<void> => {
  const appwrite = await getAppwriteClient();
  if (!appwrite) {
    return;
  }
  try {
    await appwrite.account.deleteSession('current');
  } catch (error) {
    // Ignore errors if not logged in
    console.warn('Appwrite sign out error:', error);
  }
};

export const getAppwriteSession = async (): Promise<any | null> => {
  const appwrite = await getAppwriteClient();
  if (!appwrite) {
    return null;
  }
  try {
    const session = await appwrite.account.get();
    return session;
  } catch (error) {
    return null;
  }
};

// Appwrite service for syncing data
class AppwriteService {
  private databases: Databases | null = null;
  private databasesPromise: Promise<Databases | null> | null = null;

  constructor() {
    // Initialize databases asynchronously
    this.databasesPromise = getAppwriteClient().then((appwrite) => {
      this.databases = appwrite?.databases || null;
      return this.databases;
    });
  }

  private async ensureDatabases(): Promise<Databases | null> {
    if (this.databases) {
      return this.databases;
    }
    if (this.databasesPromise) {
      return await this.databasesPromise;
    }
    this.databasesPromise = getAppwriteClient();
    const appwrite = await this.databasesPromise;
    this.databases = appwrite?.databases || null;
    return this.databases;
  }

  // Helper to log errors silently
  private logError(operation: string, error: any) {
    console.error(`Appwrite ${operation} error:`, error);
    if (error?.message) {
      console.error(`Error message:`, error.message);
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

  // Database ID - you'll need to set this in your Appwrite project
  // Can be set via runtime config from API or build-time env var
  private getDatabaseId(): string {
    // Try runtime config first (from API)
    if (runtimeConfig?.databaseId) {
      return runtimeConfig.databaseId;
    }
    // Fallback to build-time env var
    return process.env.NEXT_PUBLIC_APPWRITE_DATABASE_ID || 'arcapi';
  }

  // Quest operations
  async syncQuest(quest: Quest | Partial<Quest>, operation: 'insert' | 'update' | 'delete') {
    const databases = await this.ensureDatabases();
    if (!databases) return;
    if (!quest?.external_id) {
      console.warn('Appwrite quest sync skipped: missing external_id');
      return;
    }

    const databaseId = this.getDatabaseId();
    const collectionId = 'quests';

    try {
      if (operation === 'delete') {
        // Find document by external_id and delete
        const docs = await databases.listDocuments(databaseId, collectionId, [
          Query.equal('external_id', quest.external_id)
        ]);
        if (docs.documents.length > 0) {
          await databases.deleteDocument(databaseId, collectionId, docs.documents[0].$id);
        }
      } else {
        const payload = this.buildQuestPayload(quest);
        // Check if document exists
        const docs = await databases.listDocuments(databaseId, collectionId, [
          Query.equal('external_id', quest.external_id)
        ]);
        
        if (docs.documents.length > 0) {
          // Update existing
          await databases.updateDocument(databaseId, collectionId, docs.documents[0].$id, payload);
        } else {
          // Create new
          await databases.createDocument(databaseId, collectionId, ID.unique(), payload);
        }
      }
    } catch (error) {
      this.logError(`quest ${operation}`, error);
    }
  }

  // Item operations
  async syncItem(item: Item | Partial<Item>, operation: 'insert' | 'update' | 'delete') {
    const databases = await this.ensureDatabases();
    if (!databases) return;
    if (!item?.external_id) {
      console.warn('Appwrite item sync skipped: missing external_id');
      return;
    }

    const databaseId = this.getDatabaseId();
    const collectionId = 'items';

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
        const docs = await databases.listDocuments(databaseId, collectionId, [
          Query.equal('external_id', item.external_id)
        ]);
        if (docs.documents.length > 0) {
          await databases.deleteDocument(databaseId, collectionId, docs.documents[0].$id);
        }
      } else if (payload) {
        const docs = await databases.listDocuments(databaseId, collectionId, [
          Query.equal('external_id', item.external_id)
        ]);
        
        if (docs.documents.length > 0) {
          await databases.updateDocument(databaseId, collectionId, docs.documents[0].$id, payload);
        } else {
          await databases.createDocument(databaseId, collectionId, ID.unique(), payload);
        }
      }
    } catch (error) {
      this.logError(`item ${operation}`, error);
    }
  }

  // Skill Node operations
  async syncSkillNode(skillNode: SkillNode | Partial<SkillNode>, operation: 'insert' | 'update' | 'delete') {
    const databases = await this.ensureDatabases();
    if (!databases) return;
    if (!skillNode?.external_id) {
      console.warn('Appwrite skill node sync skipped: missing external_id');
      return;
    }
    const payload = this.buildSkillNodePayload(skillNode);

    const databaseId = this.getDatabaseId();
    const collectionId = 'skill_nodes';

    try {
      if (operation === 'delete') {
        const docs = await databases.listDocuments(databaseId, collectionId, [
          Query.equal('external_id', skillNode.external_id)
        ]);
        if (docs.documents.length > 0) {
          await databases.deleteDocument(databaseId, collectionId, docs.documents[0].$id);
        }
      } else {
        const docs = await databases.listDocuments(databaseId, collectionId, [
          Query.equal('external_id', skillNode.external_id)
        ]);
        
        if (docs.documents.length > 0) {
          await databases.updateDocument(databaseId, collectionId, docs.documents[0].$id, payload);
        } else {
          await databases.createDocument(databaseId, collectionId, ID.unique(), payload);
        }
      }
    } catch (error) {
      this.logError(`skill_node ${operation}`, error);
    }
  }

  // Hideout Module operations
  async syncHideoutModule(module: HideoutModule | Partial<HideoutModule>, operation: 'insert' | 'update' | 'delete') {
    const databases = await this.ensureDatabases();
    if (!databases) return;
    if (!module?.external_id) {
      console.warn('Appwrite hideout module sync skipped: missing external_id');
      return;
    }
    const payload = this.buildHideoutModulePayload(module);

    const databaseId = this.getDatabaseId();
    const collectionId = 'hideout_modules';

    try {
      if (operation === 'delete') {
        const docs = await databases.listDocuments(databaseId, collectionId, [
          Query.equal('external_id', module.external_id)
        ]);
        if (docs.documents.length > 0) {
          await databases.deleteDocument(databaseId, collectionId, docs.documents[0].$id);
        }
      } else {
        const docs = await databases.listDocuments(databaseId, collectionId, [
          Query.equal('external_id', module.external_id)
        ]);
        
        if (docs.documents.length > 0) {
          await databases.updateDocument(databaseId, collectionId, docs.documents[0].$id, payload);
        } else {
          await databases.createDocument(databaseId, collectionId, ID.unique(), payload);
        }
      }
    } catch (error) {
      this.logError(`hideout_module ${operation}`, error);
    }
  }

  // Enemy Type operations
  async syncEnemyType(enemyType: EnemyType | Partial<EnemyType>, operation: 'insert' | 'update' | 'delete') {
    const databases = await this.ensureDatabases();
    if (!databases) return;
    if (!enemyType?.external_id) {
      console.warn('Appwrite enemy type sync skipped: missing external_id');
      return;
    }

    const databaseId = this.getDatabaseId();
    const collectionId = 'enemy_types';

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
        const docs = await databases.listDocuments(databaseId, collectionId, [
          Query.equal('external_id', enemyType.external_id)
        ]);
        if (docs.documents.length > 0) {
          await databases.deleteDocument(databaseId, collectionId, docs.documents[0].$id);
        }
      } else if (payload) {
        const docs = await databases.listDocuments(databaseId, collectionId, [
          Query.equal('external_id', enemyType.external_id)
        ]);
        
        if (docs.documents.length > 0) {
          await databases.updateDocument(databaseId, collectionId, docs.documents[0].$id, payload);
        } else {
          await databases.createDocument(databaseId, collectionId, ID.unique(), payload);
        }
      }
    } catch (error) {
      this.logError(`enemy_type ${operation}`, error);
    }
  }

  // Alert operations
  async syncAlert(alert: Alert | Partial<Alert>, operation: 'insert' | 'update' | 'delete') {
    const databases = await this.ensureDatabases();
    if (!databases) return;

    const databaseId = this.getDatabaseId();
    const collectionId = 'alerts';

    try {
      const alertId = (alert as Alert).id;
      if (operation === 'delete') {
        // Delete by api_id since that's what we use to match API records
        const docs = await databases.listDocuments(databaseId, collectionId, [
          Query.equal('api_id', String(alertId))
        ]);
        if (docs.documents.length > 0) {
          await databases.deleteDocument(databaseId, collectionId, docs.documents[0].$id);
        }
      } else {
        const docs = await databases.listDocuments(databaseId, collectionId, [
          Query.equal('api_id', String(alertId))
        ]);
        
        const payload = {
          api_id: alertId, // Store API's id for future lookups
          name: alert.name,
          description: alert.description,
          severity: alert.severity,
          is_active: alert.is_active,
          data: alert.data || {},
        };
        
        if (docs.documents.length > 0) {
          await databases.updateDocument(databaseId, collectionId, docs.documents[0].$id, payload);
        } else {
          await databases.createDocument(databaseId, collectionId, ID.unique(), payload);
        }
      }
    } catch (error) {
      this.logError(`alert ${operation}`, error);
    }
  }

  // Read operations - fetch data from Appwrite
  async getQuests(limit = 100): Promise<Quest[]> {
    const databases = await this.ensureDatabases();
    if (!databases) return [];
    try {
      const databaseId = this.getDatabaseId();
      const { documents } = await databases.listDocuments(databaseId, 'quests', [
        Query.limit(limit),
        Query.orderDesc('created_at')
      ]);
      return (documents || []) as Quest[];
    } catch (error) {
      this.logError('getQuests', error);
      return [];
    }
  }

  async getItems(limit = 100): Promise<Item[]> {
    const databases = await this.ensureDatabases();
    if (!databases) return [];
    try {
      const databaseId = this.getDatabaseId();
      const { documents } = await databases.listDocuments(databaseId, 'items', [
        Query.limit(limit),
        Query.orderDesc('created_at')
      ]);
      return (documents || []) as Item[];
    } catch (error) {
      this.logError('getItems', error);
      return [];
    }
  }

  async getSkillNodes(limit = 100): Promise<SkillNode[]> {
    const databases = await this.ensureDatabases();
    if (!databases) return [];
    try {
      const databaseId = this.getDatabaseId();
      const { documents } = await databases.listDocuments(databaseId, 'skill_nodes', [
        Query.limit(limit),
        Query.orderDesc('created_at')
      ]);
      return (documents || []) as SkillNode[];
    } catch (error) {
      this.logError('getSkillNodes', error);
      return [];
    }
  }

  async getHideoutModules(limit = 100): Promise<HideoutModule[]> {
    const databases = await this.ensureDatabases();
    if (!databases) return [];
    try {
      const databaseId = this.getDatabaseId();
      const { documents } = await databases.listDocuments(databaseId, 'hideout_modules', [
        Query.limit(limit),
        Query.orderDesc('created_at')
      ]);
      return (documents || []) as HideoutModule[];
    } catch (error) {
      this.logError('getHideoutModules', error);
      return [];
    }
  }

  async getEnemyTypes(limit = 100): Promise<EnemyType[]> {
    const databases = await this.ensureDatabases();
    if (!databases) return [];
    try {
      const databaseId = this.getDatabaseId();
      const { documents } = await databases.listDocuments(databaseId, 'enemy_types', [
        Query.limit(limit),
        Query.orderDesc('created_at')
      ]);
      return (documents || []) as EnemyType[];
    } catch (error) {
      this.logError('getEnemyTypes', error);
      return [];
    }
  }

  async getAlerts(limit = 100): Promise<Alert[]> {
    const databases = await this.ensureDatabases();
    if (!databases) return [];
    try {
      const databaseId = this.getDatabaseId();
      const { documents } = await databases.listDocuments(databaseId, 'alerts', [
        Query.limit(limit),
        Query.orderDesc('created_at')
      ]);
      return (documents || []) as Alert[];
    } catch (error) {
      this.logError('getAlerts', error);
      return [];
    }
  }

  // Get counts for each collection
  async getCounts(): Promise<Record<string, number>> {
    const databases = await this.ensureDatabases();
    if (!databases) return {};
    try {
      const databaseId = this.getDatabaseId();
      const [quests, items, skillNodes, hideoutModules, enemyTypes, alerts] = await Promise.all([
        databases.listDocuments(databaseId, 'quests', [Query.limit(1)]),
        databases.listDocuments(databaseId, 'items', [Query.limit(1)]),
        databases.listDocuments(databaseId, 'skill_nodes', [Query.limit(1)]),
        databases.listDocuments(databaseId, 'hideout_modules', [Query.limit(1)]),
        databases.listDocuments(databaseId, 'enemy_types', [Query.limit(1)]),
        databases.listDocuments(databaseId, 'alerts', [Query.limit(1)]),
      ]);

      return {
        quests: quests.total || 0,
        items: items.total || 0,
        skillNodes: skillNodes.total || 0,
        hideoutModules: hideoutModules.total || 0,
        enemyTypes: enemyTypes.total || 0,
        alerts: alerts.total || 0,
      };
    } catch (error) {
      this.logError('getCounts', error);
      return {};
    }
  }
}

export const appwriteService = new AppwriteService();

