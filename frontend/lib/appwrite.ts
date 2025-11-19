import { Client, Databases, Account, ID, Query, Graphql } from 'appwrite';
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

  constructor() {
    // Attempt to initialize clients eagerly but ignore failures (lazy fallback below)
    getAppwriteClient()
      .then((appwrite) => {
        this.databases = appwrite?.databases || null;
        this.graphql = appwrite?.graphql || null;
      })
      .catch(() => {
        // Swallow errors and allow lazy initialization later
      });
  }

  private async ensureClients() {
    if (this.databases && this.graphql) {
      return;
    }
    const appwrite = await getAppwriteClient();
    if (!appwrite) {
      this.databases = null;
      this.graphql = null;
      return;
    }
    if (!this.databases) {
      this.databases = appwrite.databases;
    }
    if (!this.graphql) {
      this.graphql = appwrite.graphql;
    }
  }

  private async ensureDatabases(): Promise<Databases | null> {
    if (!this.databases) {
      await this.ensureClients();
    }
    return this.databases;
  }

  private async ensureGraphql(): Promise<Graphql | null> {
    if (!this.graphql) {
      await this.ensureClients();
    }
    return this.graphql;
  }

  // Helper to log errors silently
  private logError(operation: string, error: any) {
    console.error(`Appwrite ${operation} error:`, error);
    if (error?.message) {
      console.error(`Error message:`, error.message);
    }
  }

  private async listDocumentsViaGraphql(collectionId: string, limit = 100, orderBy?: string) {
    const graphql = await this.ensureGraphql();
    if (!graphql) {
      return { documents: [], total: 0 };
    }

    const databaseId = this.getDatabaseId();
    const queryStrings: string[] = [];
    if (limit > 0) {
      queryStrings.push(Query.limit(limit));
    }
    if (orderBy) {
      queryStrings.push(Query.orderDesc(orderBy));
    }

    const LIST_DOCUMENTS_QUERY = `
      query ListDocuments($databaseId: String!, $collectionId: String!, $queries: [String!]) {
        databasesListDocuments(databaseId: $databaseId, collectionId: $collectionId, queries: $queries) {
          total
          documents {
            $id
            $createdAt
            $updatedAt
            data
          }
        }
      }
    `;

    try {
      const response: any = await graphql.query({
        query: LIST_DOCUMENTS_QUERY,
        variables: {
          databaseId,
          collectionId,
          queries: queryStrings,
        },
      });
      const result = response?.data?.databasesListDocuments;
      return {
        documents: result?.documents ?? [],
        total: result?.total ?? 0,
      };
    } catch (error) {
      this.logError(`graphql listDocuments ${collectionId}`, error);
      return { documents: [], total: 0 };
    }
  }

  private getDocField(doc: any, key: string) {
    if (!doc) return undefined;
    if (doc[key] !== undefined) {
      return doc[key];
    }
    if (doc?.data && doc.data[key] !== undefined) {
      return doc.data[key];
    }
    return undefined;
  }

  async pingAppwrite(): Promise<{ success: boolean; message: string }> {
    const appwrite = await getAppwriteClient();
    if (!appwrite) {
      throw new Error('Appwrite client is not configured');
    }

    const endpoint =
      (appwrite.client as any)?.config?.endpoint ||
      runtimeConfig?.endpoint ||
      (await getAppwriteConfig()).endpoint;

    if (!endpoint) {
      throw new Error('Appwrite endpoint is not configured');
    }

    try {
      const url = new URL(`${endpoint.replace(/\/$/, '')}/ping`);
      const result = await appwrite.client.call('get', url);
      const responseMessage =
        typeof result?.message === 'string' ? result.message : 'pong';
      return {
        success: true,
        message: `Appwrite ping successful: ${responseMessage}`,
      };
    } catch (error: any) {
      this.logError('pingAppwrite', error);
      throw new Error(error?.message || 'Failed to ping Appwrite');
    }
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
    const externalId =
      (this.getDocField(doc, 'external_id') ??
        this.getDocField(doc, 'externalId') ??
        '') as string;
    const objectives = this.stringArrayToJson(this.getDocField(doc, 'objectives'));
    const rewardItems = this.stringArrayToJson(this.getDocField(doc, 'reward_item_ids') ?? this.getDocField(doc, 'rewardItemIds'));
    const dataValue = this.stringArrayToJson(this.getDocField(doc, 'data'));
    const createdAt =
      this.getDocField(doc, 'created_at') ||
      doc?.created_at ||
      doc?.$createdAt ||
      new Date().toISOString();
    const updatedAt =
      this.getDocField(doc, 'updated_at') ||
      doc?.updated_at ||
      doc?.$updatedAt ||
      new Date().toISOString();
    const syncedAt =
      this.getDocField(doc, 'synced_at') ||
      doc?.synced_at ||
      updatedAt;
    const xpValue = this.getDocField(doc, 'xp');

    return {
      id: parseInt(String(this.getDocField(doc, 'api_id') ?? doc?.api_id ?? doc?.id ?? doc?.$id ?? '0'), 10),
      external_id: externalId,
      name: (this.getDocField(doc, 'name') ?? '') as string,
      description: (this.getDocField(doc, 'description') ?? '') as string,
      trader: (this.getDocField(doc, 'trader') ?? '') as string,
      xp: xpValue === undefined || xpValue === null ? 0 : parseInt(String(xpValue), 10),
      objectives,
      reward_item_ids: rewardItems,
      data: dataValue,
      synced_at: new Date(syncedAt).toISOString(),
      created_at: new Date(createdAt).toISOString(),
      updated_at: new Date(updatedAt).toISOString(),
    };
  }

  // Helper to convert Appwrite document to Item
  private documentToItem(doc: any): Item {
    const createdAt =
      this.getDocField(doc, 'created_at') ||
      doc?.created_at ||
      doc?.$createdAt ||
      new Date().toISOString();
    const updatedAt =
      this.getDocField(doc, 'updated_at') ||
      doc?.updated_at ||
      doc?.$updatedAt ||
      new Date().toISOString();
    const syncedAt =
      this.getDocField(doc, 'synced_at') ||
      doc?.synced_at ||
      updatedAt;

    return {
      id: parseInt(String(this.getDocField(doc, 'api_id') ?? doc?.api_id ?? doc?.id ?? doc?.$id ?? '0'), 10),
      external_id: (this.getDocField(doc, 'external_id') ?? this.getDocField(doc, 'externalId') ?? '') as string,
      name: (this.getDocField(doc, 'name') ?? '') as string,
      description: (this.getDocField(doc, 'description') ?? '') as string,
      type: (this.getDocField(doc, 'type') ?? '') as string,
      image_url: (this.getDocField(doc, 'image_url') ?? this.getDocField(doc, 'imageURL') ?? '') as string,
      image_filename: (this.getDocField(doc, 'image_filename') ?? '') as string,
      data: this.stringArrayToJson(this.getDocField(doc, 'data')),
      synced_at: new Date(syncedAt).toISOString(),
      created_at: new Date(createdAt).toISOString(),
      updated_at: new Date(updatedAt).toISOString(),
    };
  }

  // Helper to convert Appwrite document to SkillNode
  private documentToSkillNode(doc: any): SkillNode {
    const createdAt =
      this.getDocField(doc, 'created_at') ||
      doc?.created_at ||
      doc?.$createdAt ||
      new Date().toISOString();
    const updatedAt =
      this.getDocField(doc, 'updated_at') ||
      doc?.updated_at ||
      doc?.$updatedAt ||
      new Date().toISOString();
    const syncedAt =
      this.getDocField(doc, 'synced_at') ||
      doc?.synced_at ||
      updatedAt;
    const maxPoints = this.getDocField(doc, 'max_points');

    return {
      id: parseInt(String(this.getDocField(doc, 'api_id') ?? doc?.api_id ?? doc?.id ?? doc?.$id ?? '0'), 10),
      external_id: (this.getDocField(doc, 'external_id') ?? this.getDocField(doc, 'externalId') ?? '') as string,
      name: (this.getDocField(doc, 'name') ?? '') as string,
      description: (this.getDocField(doc, 'description') ?? '') as string,
      impacted_skill: (this.getDocField(doc, 'impacted_skill') ?? this.getDocField(doc, 'impactedSkill') ?? '') as string,
      category: (this.getDocField(doc, 'category') ?? '') as string,
      max_points: maxPoints === undefined || maxPoints === null ? 0 : parseInt(String(maxPoints), 10),
      icon_name: (this.getDocField(doc, 'icon_name') ?? this.getDocField(doc, 'iconName') ?? '') as string,
      is_major: this.normalizeBooleanValue(this.getDocField(doc, 'is_major') ?? this.getDocField(doc, 'isMajor')) ?? false,
      position: this.stringArrayToJson(this.getDocField(doc, 'position')),
      known_value: this.stringArrayToJson(this.getDocField(doc, 'known_value') ?? this.getDocField(doc, 'knownValue')),
      prerequisite_node_ids: this.stringArrayToJson(
        this.getDocField(doc, 'prerequisite_node_ids') ?? this.getDocField(doc, 'prerequisiteNodeIds')
      ),
      data: this.stringArrayToJson(this.getDocField(doc, 'data')),
      synced_at: new Date(syncedAt).toISOString(),
      created_at: new Date(createdAt).toISOString(),
      updated_at: new Date(updatedAt).toISOString(),
    };
  }

  // Helper to convert Appwrite document to HideoutModule
  private documentToHideoutModule(doc: any): HideoutModule {
    const createdAt =
      this.getDocField(doc, 'created_at') ||
      doc?.created_at ||
      doc?.$createdAt ||
      new Date().toISOString();
    const updatedAt =
      this.getDocField(doc, 'updated_at') ||
      doc?.updated_at ||
      doc?.$updatedAt ||
      new Date().toISOString();
    const syncedAt =
      this.getDocField(doc, 'synced_at') ||
      doc?.synced_at ||
      updatedAt;
    const maxLevel = this.getDocField(doc, 'max_level');

    return {
      id: parseInt(String(this.getDocField(doc, 'api_id') ?? doc?.api_id ?? doc?.id ?? doc?.$id ?? '0'), 10),
      external_id: (this.getDocField(doc, 'external_id') ?? this.getDocField(doc, 'externalId') ?? '') as string,
      name: (this.getDocField(doc, 'name') ?? '') as string,
      description: (this.getDocField(doc, 'description') ?? '') as string,
      max_level: maxLevel === undefined || maxLevel === null ? 0 : parseInt(String(maxLevel), 10),
      levels: this.stringArrayToJson(this.getDocField(doc, 'levels')),
      data: this.stringArrayToJson(this.getDocField(doc, 'data')),
      synced_at: new Date(syncedAt).toISOString(),
      created_at: new Date(createdAt).toISOString(),
      updated_at: new Date(updatedAt).toISOString(),
    };
  }

  // Helper to convert Appwrite document to EnemyType
  private documentToEnemyType(doc: any): EnemyType {
    const createdAt =
      this.getDocField(doc, 'created_at') ||
      doc?.created_at ||
      doc?.$createdAt ||
      new Date().toISOString();
    const updatedAt =
      this.getDocField(doc, 'updated_at') ||
      doc?.updated_at ||
      doc?.$updatedAt ||
      new Date().toISOString();
    const syncedAt =
      this.getDocField(doc, 'synced_at') ||
      doc?.synced_at ||
      updatedAt;

    return {
      id: parseInt(String(this.getDocField(doc, 'api_id') ?? doc?.api_id ?? doc?.id ?? doc?.$id ?? '0'), 10),
      external_id: (this.getDocField(doc, 'external_id') ?? this.getDocField(doc, 'externalId') ?? '') as string,
      name: (this.getDocField(doc, 'name') ?? '') as string,
      description: (this.getDocField(doc, 'description') ?? '') as string,
      type: (this.getDocField(doc, 'type') ?? '') as string,
      image_url: (this.getDocField(doc, 'image_url') ?? '') as string,
      image_filename: (this.getDocField(doc, 'image_filename') ?? '') as string,
      weakpoints: this.stringArrayToJson(this.getDocField(doc, 'weakpoints')),
      data: this.stringArrayToJson(this.getDocField(doc, 'data')),
      synced_at: new Date(syncedAt).toISOString(),
      created_at: new Date(createdAt).toISOString(),
      updated_at: new Date(updatedAt).toISOString(),
    };
  }

  // Helper to convert Appwrite document to Alert
  private documentToAlert(doc: any): Alert {
    const createdAt =
      this.getDocField(doc, 'created_at') ||
      doc?.created_at ||
      doc?.$createdAt ||
      new Date().toISOString();
    const updatedAt =
      this.getDocField(doc, 'updated_at') ||
      doc?.updated_at ||
      doc?.$updatedAt ||
      new Date().toISOString();
    const severity = (this.getDocField(doc, 'severity') ?? 'info') as Alert['severity'];

    return {
      id: parseInt(String(this.getDocField(doc, 'api_id') ?? doc?.api_id ?? doc?.id ?? doc?.$id ?? '0'), 10),
      name: (this.getDocField(doc, 'name') ?? '') as string,
      description: (this.getDocField(doc, 'description') ?? '') as string,
      severity,
      is_active: this.normalizeBooleanValue(this.getDocField(doc, 'is_active')) ?? false,
      data: this.stringArrayToJson(this.getDocField(doc, 'data')),
      created_at: new Date(createdAt).toISOString(),
      updated_at: new Date(updatedAt).toISOString(),
    };
  }

  // Read operations - fetch data from Appwrite via GraphQL
  async getQuests(limit = 100): Promise<Quest[]> {
    const { documents } = await this.listDocumentsViaGraphql('quests', limit, 'created_at');
    return documents.map((doc: any) => this.documentToQuest(doc));
  }

  async getItems(limit = 100): Promise<Item[]> {
    const { documents } = await this.listDocumentsViaGraphql('items', limit, 'created_at');
    return documents.map((doc: any) => this.documentToItem(doc));
  }

  async getSkillNodes(limit = 100): Promise<SkillNode[]> {
    const { documents } = await this.listDocumentsViaGraphql('skill_nodes', limit, 'created_at');
    return documents.map((doc: any) => this.documentToSkillNode(doc));
  }

  async getHideoutModules(limit = 100): Promise<HideoutModule[]> {
    const { documents } = await this.listDocumentsViaGraphql('hideout_modules', limit, 'created_at');
    return documents.map((doc: any) => this.documentToHideoutModule(doc));
  }

  async getEnemyTypes(limit = 100): Promise<EnemyType[]> {
    const { documents } = await this.listDocumentsViaGraphql('enemy_types', limit, 'created_at');
    return documents.map((doc: any) => this.documentToEnemyType(doc));
  }

  async getAlerts(limit = 100): Promise<Alert[]> {
    const { documents } = await this.listDocumentsViaGraphql('alerts', limit, 'created_at');
    return documents.map((doc: any) => this.documentToAlert(doc));
  }

  // Get counts for each collection via GraphQL
  async getCounts(): Promise<Record<string, number>> {
    const collections: Array<[string, string]> = [
      ['quests', 'quests'],
      ['items', 'items'],
      ['skillNodes', 'skill_nodes'],
      ['hideoutModules', 'hideout_modules'],
      ['enemyTypes', 'enemy_types'],
      ['alerts', 'alerts'],
    ];

    try {
      const totals = await Promise.all(
        collections.map(async ([key, collectionId]) => {
          const { total } = await this.listDocumentsViaGraphql(collectionId, 1);
          return [key, total] as const;
        })
      );

      return totals.reduce<Record<string, number>>((acc, [key, total]) => {
        acc[key] = total;
        return acc;
      }, {});
    } catch (error) {
      this.logError('getCounts', error);
      return {};
    }
  }
}

export const appwriteService = new AppwriteService();

