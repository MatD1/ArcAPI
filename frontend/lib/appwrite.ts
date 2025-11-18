import { Client, Databases, Account, Graphql, ID, Query } from 'appwrite';
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

type AppwriteConfigShape = { endpoint: string; projectId: string; enabled: boolean; databaseId?: string; graphqlEnabled?: boolean };

const defaultConfig: AppwriteConfigShape = { endpoint: '', projectId: '', enabled: false, databaseId: '' };

const loadStaticAppwriteConfig = async (): Promise<AppwriteConfigShape> => {
  try {
    const response = await fetch('/appwrite-config.json', { cache: 'no-store' });
    if (response.ok) {
      const data = await response.json();
      return {
        enabled: data.enabled === true,
        endpoint: data.endpoint || '',
        projectId: data.projectId || '',
        databaseId: data.databaseId || '',
        graphqlEnabled: data.graphqlEnabled !== false, // Default to true
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
          databaseId: data.appwrite?.databaseId || '',
          graphqlEnabled: data.appwrite?.graphqlEnabled !== false, // Default to true if not specified
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
        databaseId: process.env.NEXT_PUBLIC_APPWRITE_DATABASE_ID || '',
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
  const graphqlEnabledEnv = process.env.NEXT_PUBLIC_APPWRITE_GRAPHQL_ENABLED;
  return {
    endpoint: process.env.NEXT_PUBLIC_APPWRITE_ENDPOINT || '',
    projectId: process.env.NEXT_PUBLIC_APPWRITE_PROJECT_ID || '',
    enabled: process.env.NEXT_PUBLIC_APPWRITE_ENABLED === 'true',
    databaseId: process.env.NEXT_PUBLIC_APPWRITE_DATABASE_ID || '',
    graphqlEnabled: graphqlEnabledEnv === undefined || graphqlEnabledEnv === 'true', // Default to true
  };
};

// Initialize Appwrite client (singleton)
let appwriteClient: Client | null = null;
let databases: Databases | null = null;
let account: Account | null = null;
let graphql: Graphql | null = null;

export const getAppwriteClient = async (): Promise<{ client: Client; databases: Databases; account: Account; graphql: Graphql } | null> => {
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
    graphql = new Graphql(appwriteClient);
  }

  return { client: appwriteClient, databases: databases!, account: account!, graphql: graphql! };
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
    // Clear session cache after logout
    clearAppwriteSessionCache();
  } catch (error) {
    // Ignore errors if not logged in
    console.warn('Appwrite sign out error:', error);
    // Still clear cache even if API call fails
    clearAppwriteSessionCache();
  }
};

// Cache for session to prevent excessive API calls
let sessionCache: { session: any | null; timestamp: number } | null = null;
const SESSION_CACHE_DURATION = 30000; // 30 seconds cache

export const getAppwriteSession = async (forceRefresh = false): Promise<any | null> => {
  // Return cached session if still valid and not forcing refresh
  if (!forceRefresh && sessionCache && Date.now() - sessionCache.timestamp < SESSION_CACHE_DURATION) {
    return sessionCache.session;
  }

  const appwrite = await getAppwriteClient();
  if (!appwrite) {
    sessionCache = { session: null, timestamp: Date.now() };
    return null;
  }
  try {
    const session = await appwrite.account.get();
    sessionCache = { session, timestamp: Date.now() };
    return session;
  } catch (error) {
    // Cache null result to avoid repeated failed requests
    sessionCache = { session: null, timestamp: Date.now() };
    return null;
  }
};

// Clear session cache (useful after logout or login)
export const clearAppwriteSessionCache = (): void => {
  sessionCache = null;
};

// OAuth methods for Appwrite
export const createOAuthSession = async (provider: 'github' | 'discord', successUrl: string, failureUrl: string): Promise<void> => {
  const appwrite = await getAppwriteClient();
  if (!appwrite) {
    throw new Error('Appwrite is not enabled');
  }
  
  try {
    // createOAuth2Session redirects automatically, but we can also get the URL if needed
    // For Appwrite SDK, this method will handle the redirect internally
    // Type assertion needed as Appwrite SDK expects OAuthProvider type but it's not exported
    await appwrite.account.createOAuth2Session(provider as any, successUrl, failureUrl);
  } catch (error: any) {
    console.error(`Appwrite OAuth ${provider} error:`, error);
    throw new Error(`Failed to initiate ${provider} OAuth: ${error.message || 'Unknown error'}`);
  }
};

export const loginWithGitHub = async (): Promise<void> => {
  const successUrl = typeof window !== 'undefined' 
    ? `${window.location.origin}/appwrite?oauth=success`
    : '/appwrite?oauth=success';
  const failureUrl = typeof window !== 'undefined'
    ? `${window.location.origin}/appwrite?oauth=failure`
    : '/appwrite?oauth=failure';
  
  await createOAuthSession('github', successUrl, failureUrl);
  // createOAuthSession will redirect automatically
};

export const loginWithDiscord = async (): Promise<void> => {
  const successUrl = typeof window !== 'undefined'
    ? `${window.location.origin}/appwrite?oauth=success`
    : '/appwrite?oauth=success';
  const failureUrl = typeof window !== 'undefined'
    ? `${window.location.origin}/appwrite?oauth=failure`
    : '/appwrite?oauth=failure';
  
  await createOAuthSession('discord', successUrl, failureUrl);
  // createOAuthSession will redirect automatically
};

// Appwrite service for syncing data
class AppwriteService {
  private databases: Databases | null = null;
  private graphql: Graphql | null = null;
  private databasesPromise: Promise<{ databases: Databases | null; graphql: Graphql | null } | null> | null = null;
  private useGraphQL: boolean = true; // Toggle to use GraphQL or REST API

  constructor() {
    // Initialize databases and GraphQL asynchronously
    this.databasesPromise = getAppwriteClient().then(async (appwrite) => {
      this.databases = appwrite?.databases || null;
      this.graphql = appwrite?.graphql || null;
      
      // Check if GraphQL is enabled via config
      const config = await getAppwriteConfig();
      this.useGraphQL = config.graphqlEnabled !== false; // Default to true
      
      return { databases: this.databases, graphql: this.graphql };
    });
  }

  private async ensureDatabases(): Promise<Databases | null> {
    if (this.databases) {
      return this.databases;
    }
    if (this.databasesPromise) {
      const result = await this.databasesPromise;
      return result?.databases || null;
    }
    this.databasesPromise = getAppwriteClient().then((appwrite) => {
      this.databases = appwrite?.databases || null;
      this.graphql = appwrite?.graphql || null;
      return { databases: this.databases, graphql: this.graphql };
    });
    const result = await this.databasesPromise;
    return result?.databases || null;
  }

  private async ensureGraphQL(): Promise<Graphql | null> {
    if (this.graphql) {
      return this.graphql;
    }
    if (this.databasesPromise) {
      const result = await this.databasesPromise;
      return result?.graphql || null;
    }
    this.databasesPromise = getAppwriteClient().then((appwrite) => {
      this.databases = appwrite?.databases || null;
      this.graphql = appwrite?.graphql || null;
      return { databases: this.databases, graphql: this.graphql };
    });
    const result = await this.databasesPromise;
    return result?.graphql || null;
  }

  // Helper to log errors silently
  private logError(operation: string, error: any) {
    console.error(`Appwrite ${operation} error:`, error);
    if (error?.message) {
      console.error(`Error message:`, error.message);
    }
  }

  // Helper to fetch all documents using pagination
  // Appwrite has a maximum limit of 100 documents per request
  private async fetchAllDocuments<T>(
    databaseId: string,
    collectionId: string,
    mapper: (doc: any) => T
  ): Promise<T[]> {
    const databases = await this.ensureDatabases();
    if (!databases) return [];

    const allDocuments: T[] = [];
    const limit = 100; // Appwrite maximum per request
    let offset = 0;
    let total = 0;
    let hasMore = true;

    if (process.env.NODE_ENV === 'development') {
      console.log(`Starting to fetch all documents from ${collectionId}...`);
    }

    while (hasMore) {
      try {
        // Build query - always specify limit explicitly to avoid default limits
        const queries = [
          Query.limit(limit),
          Query.offset(offset)
        ];
        
        const response = await databases.listDocuments(databaseId, collectionId, queries);

        // Get total from first response
        if (offset === 0) {
          total = response.total || 0;
          if (process.env.NODE_ENV === 'development') {
            console.log(`Total documents in ${collectionId}: ${total}`);
          }
        }

        const documents = response.documents || [];
        const mapped = documents.map(mapper);
        allDocuments.push(...mapped);

        if (process.env.NODE_ENV === 'development') {
          console.log(`Fetched ${documents.length} documents (offset: ${offset}, total so far: ${allDocuments.length})`);
        }

        // Check if we've fetched all documents
        if (total > 0) {
          // We know the total, check if we've fetched all
          if (allDocuments.length >= total) {
            hasMore = false;
          } else {
            offset += limit;
          }
        } else {
          // No total available, use the old method
          if (documents.length < limit) {
            hasMore = false;
          } else {
            offset += limit;
          }
        }

        // Safety check: if we're not making progress, stop
        if (documents.length === 0) {
          hasMore = false;
        }
      } catch (error) {
        this.logError(`fetchAllDocuments(${collectionId})`, error);
        hasMore = false; // Stop on error
      }
    }

    if (process.env.NODE_ENV === 'development') {
      console.log(`Finished fetching ${allDocuments.length} documents from ${collectionId}${total > 0 ? ` (expected: ${total})` : ''}`);
    }

    return allDocuments;
  }

  // Convert JSON value to string array for Appwrite compatibility
  private jsonToStringArray(value: any): string[] {
    if (value === undefined || value === null) {
      return [];
    }
    
    // If it's already a string array, return it
    if (Array.isArray(value) && value.every(item => typeof item === 'string')) {
      return value;
    }
    
    // If it's an array, convert each element to JSON string
    if (Array.isArray(value)) {
      return value.map(item => JSON.stringify(item));
    }
    
    // If it's an object, convert to single-element array with JSON string
    if (typeof value === 'object') {
      return [JSON.stringify(value)];
    }
    
    // For primitives, wrap in array as JSON string
    return [JSON.stringify(value)];
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
    const normalizedObjectives = this.jsonToStringArray(objectivesCandidate);

    const rewardItemsCandidate =
      questData?.rewardItemIds !== undefined
        ? questData.rewardItemIds
        : questData?.reward_item_ids !== undefined
          ? questData.reward_item_ids
          : this.unwrapNestedField(typedQuest?.reward_item_ids, 'reward_item_ids');
    const normalizedRewardItems = this.jsonToStringArray(rewardItemsCandidate);

    const dataValue =
      typedQuest?.data && typeof typedQuest.data === 'object' ? typedQuest.data : {};
    const normalizedData = this.jsonToStringArray(dataValue);

    return {
      external_id: externalId,
      name: normalizedName ?? fallbackName,
      description: normalizedDescription,
      trader: traderValue,
      xp: normalizedXP,
      objectives: normalizedObjectives,
      reward_item_ids: normalizedRewardItems,
      data: normalizedData,
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
    const normalizedPosition = this.jsonToStringArray(positionCandidate);

    const knownValueCandidate =
      nodeData?.knownValue !== undefined
        ? nodeData.knownValue
        : nodeData?.known_value !== undefined
          ? nodeData.known_value
          : this.unwrapNestedField(typedSkillNode?.known_value, 'known_value');
    const normalizedKnownValue = this.jsonToStringArray(knownValueCandidate);

    const prereqCandidate =
      nodeData?.prerequisiteNodeIds !== undefined
        ? nodeData.prerequisiteNodeIds
        : nodeData?.prerequisite_node_ids !== undefined
          ? nodeData.prerequisite_node_ids
          : this.unwrapNestedField(typedSkillNode?.prerequisite_node_ids, 'prerequisite_node_ids');
    const normalizedPrereqs = this.jsonToStringArray(prereqCandidate);

    const dataValue =
      typedSkillNode?.data && typeof typedSkillNode.data === 'object' ? typedSkillNode.data : {};
    const normalizedData = this.jsonToStringArray(dataValue);

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
      data: normalizedData,
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
    const normalizedLevels = this.jsonToStringArray(levelsCandidate);

    const dataValue =
      typedModule?.data && typeof typedModule.data === 'object' ? typedModule.data : {};
    const normalizedData = this.jsonToStringArray(dataValue);

    return {
      external_id: externalId,
      name: normalizedName ?? fallbackName,
      description: normalizedDescription,
      max_level: normalizedMaxLevel,
      levels: normalizedLevels,
      data: normalizedData,
    };
  }

  // Database ID - you'll need to set this in your Appwrite project
  // Can be set via runtime config from API or build-time env var
  private getDatabaseId(): string {
    // Try runtime config first (from API)
    // Note: This should be the actual Appwrite database ID (not the name)
    // The database ID is a unique identifier found in the Appwrite console
    if (runtimeConfig?.databaseId) {
      return runtimeConfig.databaseId;
    }
    // Fallback to build-time env var
    // IMPORTANT: This should be the database ID, not the database name
    // You can find the database ID in the Appwrite console under Database settings
    const dbId = process.env.NEXT_PUBLIC_APPWRITE_DATABASE_ID;
    if (!dbId) {
      console.warn('Appwrite database ID not configured. Please set NEXT_PUBLIC_APPWRITE_DATABASE_ID to the actual database ID from Appwrite console.');
    }
    return dbId || '';
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
    if (!databaseId) {
      console.warn('Appwrite database ID not configured, skipping quest sync');
      return;
    }
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
    if (!databaseId) {
      console.warn('Appwrite database ID not configured, skipping item sync');
      return;
    }
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
            data: this.jsonToStringArray(item.data || {}),
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
    if (!databaseId) {
      console.warn('Appwrite database ID not configured, skipping skill node sync');
      return;
    }
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
    if (!databaseId) {
      console.warn('Appwrite database ID not configured, skipping hideout module sync');
      return;
    }
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
    if (!databaseId) {
      console.warn('Appwrite database ID not configured, skipping enemy type sync');
      return;
    }
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
            weakpoints: this.jsonToStringArray(enemyType.weakpoints || {}),
            data: this.jsonToStringArray(enemyType.data || {}),
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
    if (!databaseId) {
      console.warn('Appwrite database ID not configured, skipping alert sync');
      return;
    }
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
          data: this.jsonToStringArray(alert.data || {}),
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

  // Helper to parse string array back to JSON
  private stringArrayToJson(value: any): any {
    if (!value) return null;
    // If it's already an array of strings, parse each element
    if (Array.isArray(value) && value.length > 0 && typeof value[0] === 'string') {
      try {
        // Try to parse each string as JSON
        return value.map(str => {
          try {
            return JSON.parse(str);
          } catch {
            return str; // Return as-is if not valid JSON
          }
        });
      } catch {
        return value;
      }
    }
    // If it's a single string, try to parse it
    if (typeof value === 'string') {
      try {
        return JSON.parse(value);
      } catch {
        return value;
      }
    }
    return value;
  }

  // Helper to convert Appwrite document to Quest
  private documentToQuest(doc: any): Quest {
    return {
      id: parseInt(doc.api_id || doc.id || '0', 10),
      external_id: doc.external_id || '',
      name: doc.name || '',
      description: doc.description || '',
      trader: doc.trader || '',
      xp: parseInt(doc.xp || '0', 10),
      objectives: this.stringArrayToJson(doc.objectives),
      reward_item_ids: this.stringArrayToJson(doc.reward_item_ids),
      data: this.stringArrayToJson(doc.data),
      synced_at: doc.synced_at ? new Date(doc.synced_at).toISOString() : new Date().toISOString(),
      created_at: doc.created_at ? new Date(doc.created_at).toISOString() : new Date().toISOString(),
      updated_at: doc.updated_at ? new Date(doc.updated_at).toISOString() : new Date().toISOString(),
    };
  }

  // Helper to convert Appwrite document to Item
  private documentToItem(doc: any): Item {
    return {
      id: parseInt(doc.api_id || doc.id || '0', 10),
      external_id: doc.external_id || '',
      name: doc.name || '',
      description: doc.description || '',
      type: doc.type || '',
      image_url: doc.image_url || '',
      image_filename: doc.image_filename || '',
      data: this.stringArrayToJson(doc.data),
      synced_at: doc.synced_at ? new Date(doc.synced_at).toISOString() : new Date().toISOString(),
      created_at: doc.created_at ? new Date(doc.created_at).toISOString() : new Date().toISOString(),
      updated_at: doc.updated_at ? new Date(doc.updated_at).toISOString() : new Date().toISOString(),
    };
  }

  // Helper to convert Appwrite document to SkillNode
  private documentToSkillNode(doc: any): SkillNode {
    return {
      id: parseInt(doc.api_id || doc.id || '0', 10),
      external_id: doc.external_id || '',
      name: doc.name || '',
      description: doc.description || '',
      impacted_skill: doc.impacted_skill || '',
      category: doc.category || '',
      max_points: parseInt(doc.max_points || '0', 10),
      icon_name: doc.icon_name || '',
      is_major: doc.is_major === true || doc.is_major === 'true',
      position: this.stringArrayToJson(doc.position),
      known_value: this.stringArrayToJson(doc.known_value),
      prerequisite_node_ids: this.stringArrayToJson(doc.prerequisite_node_ids),
      data: this.stringArrayToJson(doc.data),
      synced_at: doc.synced_at ? new Date(doc.synced_at).toISOString() : new Date().toISOString(),
      created_at: doc.created_at ? new Date(doc.created_at).toISOString() : new Date().toISOString(),
      updated_at: doc.updated_at ? new Date(doc.updated_at).toISOString() : new Date().toISOString(),
    };
  }

  // Helper to convert Appwrite document to HideoutModule
  private documentToHideoutModule(doc: any): HideoutModule {
    return {
      id: parseInt(doc.api_id || doc.id || '0', 10),
      external_id: doc.external_id || '',
      name: doc.name || '',
      description: doc.description || '',
      max_level: parseInt(doc.max_level || '0', 10),
      levels: this.stringArrayToJson(doc.levels),
      data: this.stringArrayToJson(doc.data),
      synced_at: doc.synced_at ? new Date(doc.synced_at).toISOString() : new Date().toISOString(),
      created_at: doc.created_at ? new Date(doc.created_at).toISOString() : new Date().toISOString(),
      updated_at: doc.updated_at ? new Date(doc.updated_at).toISOString() : new Date().toISOString(),
    };
  }

  // Helper to convert Appwrite document to EnemyType
  private documentToEnemyType(doc: any): EnemyType {
    return {
      id: parseInt(doc.api_id || doc.id || '0', 10),
      external_id: doc.external_id || '',
      name: doc.name || '',
      description: doc.description || '',
      type: doc.type || '',
      image_url: doc.image_url || '',
      image_filename: doc.image_filename || '',
      weakpoints: this.stringArrayToJson(doc.weakpoints),
      data: this.stringArrayToJson(doc.data),
      synced_at: doc.synced_at ? new Date(doc.synced_at).toISOString() : new Date().toISOString(),
      created_at: doc.created_at ? new Date(doc.created_at).toISOString() : new Date().toISOString(),
      updated_at: doc.updated_at ? new Date(doc.updated_at).toISOString() : new Date().toISOString(),
    };
  }

  // Helper to convert Appwrite document to Alert
  private documentToAlert(doc: any): Alert {
    return {
      id: parseInt(doc.api_id || doc.id || '0', 10),
      name: doc.name || '',
      description: doc.description || '',
      severity: doc.severity || 'info',
      is_active: doc.is_active === true || doc.is_active === 'true',
      data: this.stringArrayToJson(doc.data),
      created_at: doc.created_at ? new Date(doc.created_at).toISOString() : new Date().toISOString(),
      updated_at: doc.updated_at ? new Date(doc.updated_at).toISOString() : new Date().toISOString(),
    };
  }

  // Read operations - fetch data from Appwrite
  // Fetches all records using pagination (Appwrite limit is 100 per request)
  async getQuests(limit?: number): Promise<Quest[]> {
    const databases = await this.ensureDatabases();
    if (!databases) return [];
    try {
      const databaseId = this.getDatabaseId();
      if (!databaseId) {
        console.warn('Appwrite database ID not configured, returning empty quests');
        return [];
      }
      
      // If limit is specified, fetch only that many
      if (limit !== undefined && limit > 0) {
        const { documents } = await databases.listDocuments(databaseId, 'quests', [
          Query.limit(Math.min(limit, 100)), // Appwrite max is 100
          Query.orderDesc('created_at')
        ]);
        return (documents || []).map(doc => this.documentToQuest(doc));
      }
      
      // Otherwise, fetch all records using pagination
      return await this.fetchAllDocuments(databaseId, 'quests', (doc) => this.documentToQuest(doc));
    } catch (error) {
      this.logError('getQuests', error);
      return [];
    }
  }

  async getItems(limit?: number): Promise<Item[]> {
    // Try GraphQL first if enabled
    if (this.useGraphQL) {
      try {
        if (process.env.NODE_ENV === 'development') {
          console.log('Attempting to fetch items via GraphQL...');
        }
        const result = await this.getItemsGraphQL(limit);
        if (process.env.NODE_ENV === 'development') {
          console.log(`GraphQL returned ${result.length} items`);
        }
        return result;
      } catch (error) {
        console.warn('GraphQL query failed, falling back to REST API:', error);
        // Fall back to REST API
      }
    }

    // REST API fallback
    const databases = await this.ensureDatabases();
    if (!databases) return [];
    try {
      const databaseId = this.getDatabaseId();
      if (!databaseId) {
        console.warn('Appwrite database ID not configured, returning empty items');
        return [];
      }
      
      // If limit is specified, fetch only that many
      if (limit !== undefined && limit > 0) {
        const { documents } = await databases.listDocuments(databaseId, 'items', [
          Query.limit(Math.min(limit, 100)), // Appwrite max is 100
          Query.orderDesc('created_at')
        ]);
        return (documents || []).map(doc => this.documentToItem(doc));
      }
      
      // Otherwise, fetch all records using pagination
      return await this.fetchAllDocuments(databaseId, 'items', (doc) => this.documentToItem(doc));
    } catch (error) {
      this.logError('getItems', error);
      return [];
    }
  }

  // GraphQL-based method to fetch items
  private async getItemsGraphQL(limit?: number): Promise<Item[]> {
    const graphql = await this.ensureGraphQL();
    if (!graphql) {
      throw new Error('GraphQL not available');
    }

    const databaseId = this.getDatabaseId();
    if (!databaseId) {
      throw new Error('Database ID not configured');
    }

    // Build GraphQL query - Appwrite GraphQL uses different field names
    // Try to fetch all items in one query if possible, or paginate
    const query = `
      query GetItems($databaseId: String!, $collectionId: String!, $limit: Int!, $offset: Int) {
        databasesListDocuments(
          databaseId: $databaseId
          collectionId: $collectionId
          limit: $limit
          offset: $offset
        ) {
          total
          documents {
            _id
            _createdAt
            _updatedAt
            external_id
            name
            description
            type
            image_url
            image_filename
            data
            synced_at
            created_at
            updated_at
          }
        }
      }
    `;

    const allItems: Item[] = [];
    const pageLimit = 100; // Appwrite GraphQL still has limits per request
    let offset = 0;
    let total = 0;
    let hasMore = true;

    while (hasMore) {
      try {
        const remaining = limit ? Math.max(limit - allItems.length, 1) : pageLimit;
        const effectiveLimit = Math.min(pageLimit, remaining);

        const variables: any = {
          databaseId,
          collectionId: 'items',
          limit: effectiveLimit,
          offset,
        };

        if (process.env.NODE_ENV === 'development') {
          console.log(`GraphQL query: offset=${offset}, limit=${variables.limit}`);
        }

        const response = await graphql.query({
          query,
          variables,
        }) as any; // Appwrite GraphQL response type is not fully typed

        if (process.env.NODE_ENV === 'development') {
          console.log('GraphQL response:', response);
        }

        // Handle different possible response structures
        const result = response.data?.databasesListDocuments || 
                      response.databasesListDocuments ||
                      response.data;
        
        if (!result || !result.documents) {
          // Try alternative response structure
          const altResult = (response as any).databases?.listDocuments;
          if (!altResult) {
            throw new Error('Invalid GraphQL response structure');
          }
          const documents = altResult.documents || [];
          allItems.push(...documents.map((doc: any) => this.documentToItem(doc)));
          total = altResult.total || documents.length;
          hasMore = false;
          break;
        }

        if (offset === 0) {
          total = result.total || 0;
        }

        const documents = result.documents || [];
        allItems.push(...documents.map((doc: any) => this.documentToItem(doc)));

        // Check if we've fetched all documents
        if (total > 0 && allItems.length >= total) {
          hasMore = false;
        } else if (documents.length === 0) {
          hasMore = false;
        } else if (limit && allItems.length >= limit) {
          hasMore = false;
        } else {
          offset += documents.length;
        }
      } catch (error: any) {
        // If it's a schema error, the GraphQL API might not be available or have different structure
        if (error?.message?.includes('Unknown type') || error?.message?.includes('Cannot query field')) {
          throw new Error('GraphQL schema mismatch - falling back to REST API');
        }
        this.logError('getItemsGraphQL', error);
        throw error; // Re-throw to trigger REST API fallback
      }
    }

    if (process.env.NODE_ENV === 'development') {
      console.log(`GraphQL: Fetched ${allItems.length} items${total > 0 ? ` (total: ${total})` : ''}`);
    }

    return limit ? allItems.slice(0, limit) : allItems;
  }

  async getSkillNodes(limit?: number): Promise<SkillNode[]> {
    const databases = await this.ensureDatabases();
    if (!databases) return [];
    try {
      const databaseId = this.getDatabaseId();
      if (!databaseId) {
        console.warn('Appwrite database ID not configured, returning empty skill nodes');
        return [];
      }
      
      // If limit is specified, fetch only that many
      if (limit !== undefined && limit > 0) {
        const { documents } = await databases.listDocuments(databaseId, 'skill_nodes', [
          Query.limit(Math.min(limit, 100)), // Appwrite max is 100
          Query.orderDesc('created_at')
        ]);
        return (documents || []).map(doc => this.documentToSkillNode(doc));
      }
      
      // Otherwise, fetch all records using pagination
      return await this.fetchAllDocuments(databaseId, 'skill_nodes', (doc) => this.documentToSkillNode(doc));
    } catch (error) {
      this.logError('getSkillNodes', error);
      return [];
    }
  }

  async getHideoutModules(limit?: number): Promise<HideoutModule[]> {
    const databases = await this.ensureDatabases();
    if (!databases) return [];
    try {
      const databaseId = this.getDatabaseId();
      if (!databaseId) {
        console.warn('Appwrite database ID not configured, returning empty hideout modules');
        return [];
      }
      
      // If limit is specified, fetch only that many
      if (limit !== undefined && limit > 0) {
        const { documents } = await databases.listDocuments(databaseId, 'hideout_modules', [
          Query.limit(Math.min(limit, 100)), // Appwrite max is 100
          Query.orderDesc('created_at')
        ]);
        return (documents || []).map(doc => this.documentToHideoutModule(doc));
      }
      
      // Otherwise, fetch all records using pagination
      return await this.fetchAllDocuments(databaseId, 'hideout_modules', (doc) => this.documentToHideoutModule(doc));
    } catch (error) {
      this.logError('getHideoutModules', error);
      return [];
    }
  }

  async getEnemyTypes(limit?: number): Promise<EnemyType[]> {
    const databases = await this.ensureDatabases();
    if (!databases) return [];
    try {
      const databaseId = this.getDatabaseId();
      if (!databaseId) {
        console.warn('Appwrite database ID not configured, returning empty enemy types');
        return [];
      }
      
      // If limit is specified, fetch only that many
      if (limit !== undefined && limit > 0) {
        const { documents } = await databases.listDocuments(databaseId, 'enemy_types', [
          Query.limit(Math.min(limit, 100)), // Appwrite max is 100
          Query.orderDesc('created_at')
        ]);
        return (documents || []).map(doc => this.documentToEnemyType(doc));
      }
      
      // Otherwise, fetch all records using pagination
      return await this.fetchAllDocuments(databaseId, 'enemy_types', (doc) => this.documentToEnemyType(doc));
    } catch (error) {
      this.logError('getEnemyTypes', error);
      return [];
    }
  }

  async getAlerts(limit?: number): Promise<Alert[]> {
    const databases = await this.ensureDatabases();
    if (!databases) return [];
    try {
      const databaseId = this.getDatabaseId();
      if (!databaseId) {
        console.warn('Appwrite database ID not configured, returning empty alerts');
        return [];
      }
      
      // If limit is specified, fetch only that many
      if (limit !== undefined && limit > 0) {
        const { documents } = await databases.listDocuments(databaseId, 'alerts', [
          Query.limit(Math.min(limit, 100)), // Appwrite max is 100
          Query.orderDesc('created_at')
        ]);
        return (documents || []).map(doc => this.documentToAlert(doc));
      }
      
      // Otherwise, fetch all records using pagination
      return await this.fetchAllDocuments(databaseId, 'alerts', (doc) => this.documentToAlert(doc));
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
      if (!databaseId) {
        console.warn('Appwrite database ID not configured, returning empty counts');
        return {};
      }
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

