import axios, { AxiosInstance, AxiosError } from 'axios';
import type {
  User,
  APIKey,
  JWTToken,
  Quest,
  Mission,
  Item,
  SkillNode,
  HideoutModule,
  EnemyType,
  Alert,
  AuditLog,
  PaginatedResponse,
  LoginResponse,
  RequiredItemsResponse,
  UserQuestProgress,
  UserHideoutModuleProgress,
  UserSkillNodeProgress,
  UserBlueprintProgress,
  AllUserProgress,
} from '@/types';
import { appwriteService, isAppwriteEnabled, isAppwriteEnabledSync } from './appwrite';

// Use relative URL when embedded, or explicit URL if provided
const getAPIURL = () => {
  if (process.env.NEXT_PUBLIC_API_URL) {
    return process.env.NEXT_PUBLIC_API_URL;
  }
  // When embedded in Go app, use relative URL
  if (typeof window !== 'undefined') {
    return window.location.origin;
  }
  return 'http://localhost:8080';
};

const API_URL = getAPIURL();

class APIClient {
  private client: AxiosInstance;
  private apiKey: string | null = null;
  private jwtToken: string | null = null;

  constructor() {
    this.client = axios.create({
      baseURL: `${API_URL}/api/v1`,
      headers: {
        'Content-Type': 'application/json',
      },
    });

    // Load from localStorage
    if (typeof window !== 'undefined') {
      this.apiKey = localStorage.getItem('api_key');
      this.jwtToken = localStorage.getItem('jwt_token');
      this.updateHeaders();
    }
  }

  setAuth(apiKey: string | null, jwtToken: string) {
    this.apiKey = apiKey;
    this.jwtToken = jwtToken;
    if (typeof window !== 'undefined') {
      if (apiKey) {
        localStorage.setItem('api_key', apiKey);
      } else {
        localStorage.removeItem('api_key');
      }
      localStorage.setItem('jwt_token', jwtToken);
    }
    this.updateHeaders();
  }

  setJWT(jwtToken: string) {
    this.jwtToken = jwtToken;
    if (typeof window !== 'undefined') {
      localStorage.setItem('jwt_token', jwtToken);
    }
    this.updateHeaders();
  }

  clearAuth() {
    this.apiKey = null;
    this.jwtToken = null;
    if (typeof window !== 'undefined') {
      localStorage.removeItem('api_key');
      localStorage.removeItem('jwt_token');
    }
    this.updateHeaders();
  }

  private updateHeaders() {
    // Always set JWT if available (needed for read operations)
    if (this.jwtToken) {
      this.client.defaults.headers.common['Authorization'] = `Bearer ${this.jwtToken}`;
    } else {
      delete this.client.defaults.headers.common['Authorization'];
    }

    // Only set API key if available (needed for write operations by non-admin users)
    if (this.apiKey) {
      this.client.defaults.headers.common['X-API-Key'] = this.apiKey;
    } else {
      delete this.client.defaults.headers.common['X-API-Key'];
    }
  }

  // Auth
  async login(apiKey: string): Promise<LoginResponse> {
    const response = await this.client.post<LoginResponse>('/auth/login', { api_key: apiKey });
    if (response.data.token) {
      this.setAuth(apiKey, response.data.token);
    }
    return response.data;
  }

  // Quests
  async getQuests(page = 1, limit = 20): Promise<PaginatedResponse<Quest>> {
    const response = await this.client.get<PaginatedResponse<Quest>>('/quests', {
      params: { page, limit },
    });
    return response.data;
  }

  async getQuest(id: number): Promise<Quest> {
    const response = await this.client.get<Quest>(`/quests/${id}`);
    return response.data;
  }

  async createQuest(data: Partial<Quest>): Promise<Quest> {
    const response = await this.client.post<Quest>('/quests', data);
    const quest = response.data;
    // Sync to Appwrite if enabled
    if (isAppwriteEnabledSync() && quest.external_id) {
      await appwriteService.syncQuest(quest, 'insert').catch(() => {
        // Silently fail - Appwrite sync is optional
      });
    }
    return quest;
  }

  async updateQuest(id: number, data: Partial<Quest>): Promise<Quest> {
    const response = await this.client.put<Quest>(`/quests/${id}`, data);
    const quest = response.data;
    // Sync to Appwrite if enabled
    if (isAppwriteEnabledSync() && quest.external_id) {
      await appwriteService.syncQuest(quest, 'update').catch(() => {
        // Silently fail - Appwrite sync is optional
      });
    }
    return quest;
  }

  async deleteQuest(id: number): Promise<void> {
    // Get quest before deleting to sync to Appwrite
    let quest: Quest | null = null;
    if (isAppwriteEnabledSync()) {
      try {
        quest = await this.getQuest(id);
      } catch {
        // If we can't get the quest, skip Appwrite sync
      }
    }
    await this.client.delete(`/quests/${id}`);
    // Sync to Appwrite if enabled
    if (isAppwriteEnabledSync() && quest && quest.external_id) {
      await appwriteService.syncQuest(quest, 'delete').catch(() => {
        // Silently fail - Appwrite sync is optional
      });
    }
  }

  // Missions (deprecated - use quests instead)
  async getMissions(page = 1, limit = 20): Promise<PaginatedResponse<Mission>> {
    return this.getQuests(page, limit);
  }

  async getMission(id: number): Promise<Mission> {
    return this.getQuest(id);
  }

  async createMission(data: Partial<Mission>): Promise<Mission> {
    return this.createQuest(data);
  }

  async updateMission(id: number, data: Partial<Mission>): Promise<Mission> {
    return this.updateQuest(id, data);
  }

  async deleteMission(id: number): Promise<void> {
    return this.deleteQuest(id);
  }

  // Items
  async getItems(page = 1, limit = 20): Promise<PaginatedResponse<Item>> {
    const response = await this.client.get<PaginatedResponse<Item>>('/items', {
      params: { page, limit },
    });
    return response.data;
  }

  async getItem(id: number): Promise<Item> {
    const response = await this.client.get<Item>(`/items/${id}`);
    return response.data;
  }

  async createItem(data: Partial<Item>): Promise<Item> {
    const response = await this.client.post<Item>('/items', data);
    const item = response.data;
    // Sync to Appwrite if enabled
    if (isAppwriteEnabledSync() && item.external_id) {
      await appwriteService.syncItem(item, 'insert').catch(() => {
        // Silently fail - Appwrite sync is optional
      });
    }
    return item;
  }

  async updateItem(id: number, data: Partial<Item>): Promise<Item> {
    const response = await this.client.put<Item>(`/items/${id}`, data);
    const item = response.data;
    // Sync to Appwrite if enabled
    if (isAppwriteEnabledSync() && item.external_id) {
      await appwriteService.syncItem(item, 'update').catch(() => {
        // Silently fail - Appwrite sync is optional
      });
    }
    return item;
  }

  async deleteItem(id: number): Promise<void> {
    // Get item before deleting to sync to Appwrite
    let item: Item | null = null;
    if (isAppwriteEnabledSync()) {
      try {
        item = await this.getItem(id);
      } catch {
        // If we can't get the item, skip Appwrite sync
      }
    }
    await this.client.delete(`/items/${id}`);
    // Sync to Appwrite if enabled
    if (isAppwriteEnabledSync() && item && item.external_id) {
      await appwriteService.syncItem(item, 'delete').catch(() => {
        // Silently fail - Appwrite sync is optional
      });
    }
  }

  async getRequiredItems(): Promise<RequiredItemsResponse> {
    const response = await this.client.get<RequiredItemsResponse>('/items/required');
    return response.data;
  }

  // Skill Nodes
  async getSkillNodes(page = 1, limit = 20): Promise<PaginatedResponse<SkillNode>> {
    const response = await this.client.get<PaginatedResponse<SkillNode>>('/skill-nodes', {
      params: { page, limit },
    });
    return response.data;
  }

  async getSkillNode(id: number): Promise<SkillNode> {
    const response = await this.client.get<SkillNode>(`/skill-nodes/${id}`);
    return response.data;
  }

  async createSkillNode(data: Partial<SkillNode>): Promise<SkillNode> {
    const response = await this.client.post<SkillNode>('/skill-nodes', data);
    const skillNode = response.data;
    // Sync to Appwrite if enabled
    if (isAppwriteEnabledSync() && skillNode.external_id) {
      await appwriteService.syncSkillNode(skillNode, 'insert').catch(() => {
        // Silently fail - Appwrite sync is optional
      });
    }
    return skillNode;
  }

  async updateSkillNode(id: number, data: Partial<SkillNode>): Promise<SkillNode> {
    const response = await this.client.put<SkillNode>(`/skill-nodes/${id}`, data);
    const skillNode = response.data;
    // Sync to Appwrite if enabled
    if (isAppwriteEnabledSync() && skillNode.external_id) {
      await appwriteService.syncSkillNode(skillNode, 'update').catch(() => {
        // Silently fail - Appwrite sync is optional
      });
    }
    return skillNode;
  }

  async deleteSkillNode(id: number): Promise<void> {
    // Get skill node before deleting to sync to Appwrite
    let skillNode: SkillNode | null = null;
    if (isAppwriteEnabledSync()) {
      try {
        skillNode = await this.getSkillNode(id);
      } catch {
        // If we can't get the skill node, skip Appwrite sync
      }
    }
    await this.client.delete(`/skill-nodes/${id}`);
    // Sync to Appwrite if enabled
    if (isAppwriteEnabledSync() && skillNode && skillNode.external_id) {
      await appwriteService.syncSkillNode(skillNode, 'delete').catch(() => {
        // Silently fail - Appwrite sync is optional
      });
    }
  }

  // Hideout Modules
  async getHideoutModules(page = 1, limit = 20): Promise<PaginatedResponse<HideoutModule>> {
    const response = await this.client.get<PaginatedResponse<HideoutModule>>('/hideout-modules', {
      params: { page, limit },
    });
    return response.data;
  }

  async getHideoutModule(id: number): Promise<HideoutModule> {
    const response = await this.client.get<HideoutModule>(`/hideout-modules/${id}`);
    return response.data;
  }

  async createHideoutModule(data: Partial<HideoutModule>): Promise<HideoutModule> {
    const response = await this.client.post<HideoutModule>('/hideout-modules', data);
    const module = response.data;
    // Sync to Appwrite if enabled
    if (isAppwriteEnabledSync() && module.external_id) {
      await appwriteService.syncHideoutModule(module, 'insert').catch(() => {
        // Silently fail - Appwrite sync is optional
      });
    }
    return module;
  }

  async updateHideoutModule(id: number, data: Partial<HideoutModule>): Promise<HideoutModule> {
    const response = await this.client.put<HideoutModule>(`/hideout-modules/${id}`, data);
    const module = response.data;
    // Sync to Appwrite if enabled
    if (isAppwriteEnabledSync() && module.external_id) {
      await appwriteService.syncHideoutModule(module, 'update').catch(() => {
        // Silently fail - Appwrite sync is optional
      });
    }
    return module;
  }

  async deleteHideoutModule(id: number): Promise<void> {
    // Get hideout module before deleting to sync to Appwrite
    let module: HideoutModule | null = null;
    if (isAppwriteEnabledSync()) {
      try {
        module = await this.getHideoutModule(id);
      } catch {
        // If we can't get the module, skip Appwrite sync
      }
    }
    await this.client.delete(`/hideout-modules/${id}`);
    // Sync to Appwrite if enabled
    if (isAppwriteEnabledSync() && module && module.external_id) {
      await appwriteService.syncHideoutModule(module, 'delete').catch(() => {
        // Silently fail - Appwrite sync is optional
      });
    }
  }

  // Enemy Types
  async getEnemyTypes(page = 1, limit = 20): Promise<PaginatedResponse<EnemyType>> {
    const response = await this.client.get<PaginatedResponse<EnemyType>>('/enemy-types', {
      params: { page, limit },
    });
    return response.data;
  }

  async getEnemyType(id: number): Promise<EnemyType> {
    const response = await this.client.get<EnemyType>(`/enemy-types/${id}`);
    return response.data;
  }

  async createEnemyType(data: Partial<EnemyType>): Promise<EnemyType> {
    const response = await this.client.post<EnemyType>('/enemy-types', data);
    const enemyType = response.data;
    // Sync to Appwrite if enabled
    if (isAppwriteEnabledSync() && enemyType.external_id) {
      await appwriteService.syncEnemyType(enemyType, 'insert').catch(() => {
        // Silently fail - Appwrite sync is optional
      });
    }
    return enemyType;
  }

  async updateEnemyType(id: number, data: Partial<EnemyType>): Promise<EnemyType> {
    const response = await this.client.put<EnemyType>(`/enemy-types/${id}`, data);
    const enemyType = response.data;
    // Sync to Appwrite if enabled
    if (isAppwriteEnabledSync() && enemyType.external_id) {
      await appwriteService.syncEnemyType(enemyType, 'update').catch(() => {
        // Silently fail - Appwrite sync is optional
      });
    }
    return enemyType;
  }

  async deleteEnemyType(id: number): Promise<void> {
    // Get enemy type before deleting to sync to Appwrite
    let enemyType: EnemyType | null = null;
    if (isAppwriteEnabledSync()) {
      try {
        enemyType = await this.getEnemyType(id);
      } catch {
        // If we can't get the enemy type, skip Appwrite sync
      }
    }
    await this.client.delete(`/enemy-types/${id}`);
    // Sync to Appwrite if enabled
    if (isAppwriteEnabledSync() && enemyType && enemyType.external_id) {
      await appwriteService.syncEnemyType(enemyType, 'delete').catch(() => {
        // Silently fail - Appwrite sync is optional
      });
    }
  }

  // Alerts
  async getAlerts(page = 1, limit = 20): Promise<PaginatedResponse<Alert>> {
    const response = await this.client.get<PaginatedResponse<Alert>>('/alerts', {
      params: { page, limit },
    });
    return response.data;
  }

  async getActiveAlerts(): Promise<{ data: Alert[] }> {
    const response = await this.client.get<{ data: Alert[] }>('/alerts/active');
    return response.data;
  }

  // GitHub Data (Bots, Maps, Traders, Projects)
  async getBots(): Promise<any> {
    const response = await this.client.get('/data/bots');
    return response.data;
  }

  async getMaps(): Promise<any> {
    const response = await this.client.get('/data/maps');
    return response.data;
  }

  async getTraders(): Promise<any> {
    const response = await this.client.get('/data/traders');
    return response.data;
  }

  async getProjects(): Promise<any> {
    const response = await this.client.get('/data/projects');
    return response.data;
  }

  async getAlert(id: number): Promise<Alert> {
    const response = await this.client.get<Alert>(`/alerts/${id}`);
    return response.data;
  }

  async createAlert(data: Partial<Alert>): Promise<Alert> {
    const response = await this.client.post<Alert>('/alerts', data);
    const alert = response.data;
    // Sync to Appwrite if enabled
    if (isAppwriteEnabledSync()) {
      await appwriteService.syncAlert(alert, 'insert').catch(() => {
        // Silently fail - Appwrite sync is optional
      });
    }
    return alert;
  }

  async updateAlert(id: number, data: Partial<Alert>): Promise<Alert> {
    const response = await this.client.put<Alert>(`/alerts/${id}`, data);
    const alert = response.data;
    // Sync to Appwrite if enabled
    if (isAppwriteEnabledSync()) {
      await appwriteService.syncAlert(alert, 'update').catch(() => {
        // Silently fail - Appwrite sync is optional
      });
    }
    return alert;
  }

  async deleteAlert(id: number): Promise<void> {
    // Get alert before deleting to sync to Appwrite
    let alert: Alert | null = null;
    if (isAppwriteEnabledSync()) {
      try {
        alert = await this.getAlert(id);
      } catch {
        // If we can't get the alert, skip Appwrite sync
      }
    }
    await this.client.delete(`/alerts/${id}`);
    // Sync to Appwrite if enabled
    if (isAppwriteEnabledSync() && alert) {
      await appwriteService.syncAlert(alert, 'delete').catch(() => {
        // Silently fail - Appwrite sync is optional
      });
    }
  }

  // Management API (Admin only)
  async createAPIKey(name: string): Promise<{ api_key: string; name: string; warning: string }> {
    const response = await this.client.post<{ api_key: string; name: string; warning: string }>(
      '/admin/api-keys',
      { name }
    );
    return response.data;
  }

  async getAPIKeys(): Promise<APIKey[]> {
    const response = await this.client.get<APIKey[]>('/admin/api-keys');
    return response.data;
  }

  async revokeAPIKey(id: number): Promise<void> {
    await this.client.delete(`/admin/api-keys/${id}`);
  }

  async getJWTTokens(): Promise<JWTToken[]> {
    const response = await this.client.get<JWTToken[]>('/admin/jwts');
    return response.data;
  }

  async revokeJWT(token: string): Promise<void> {
    await this.client.post('/admin/jwts/revoke', { token });
  }

  async getLogs(params?: {
    page?: number;
    limit?: number;
    api_key_id?: number;
    jwt_token_id?: number;
    user_id?: number;
    endpoint?: string;
    method?: string;
    start_time?: string;
    end_time?: string;
  }): Promise<PaginatedResponse<AuditLog>> {
    const response = await this.client.get<PaginatedResponse<AuditLog>>('/admin/logs', { params });
    return response.data;
  }

  // Sync
  async forceSync(): Promise<{ message: string; status: string }> {
    const response = await this.client.post<{ message: string; status: string }>('/admin/sync/force');
    return response.data;
  }

  async getSyncStatus(): Promise<{ is_running: boolean }> {
    const response = await this.client.get<{ is_running: boolean }>('/admin/sync/status');
    return response.data;
  }

  // Users
  async getUsers(page = 1, limit = 50): Promise<PaginatedResponse<User>> {
    const response = await this.client.get<PaginatedResponse<User>>('/admin/users', {
      params: { page, limit },
    });
    return response.data;
  }

  async getUser(id: number): Promise<{ user: User; api_keys: APIKey[]; jwt_tokens: JWTToken[] }> {
    const response = await this.client.get<{ user: User; api_keys: APIKey[]; jwt_tokens: JWTToken[] }>(`/admin/users/${id}`);
    return response.data;
  }

  async updateUserAccess(userId: number, canAccessData: boolean): Promise<{ message: string; user: User }> {
    const response = await this.client.put<{ message: string; user: User }>(`/admin/users/${userId}/access`, {
      can_access_data: canAccessData,
    });
    return response.data;
  }

  async deleteUser(userId: number): Promise<{ message: string }> {
    const response = await this.client.delete<{ message: string }>(`/admin/users/${userId}`);
    return response.data;
  }

  // Blueprints
  async getBlueprints(): Promise<{ data: Item[] }> {
    const response = await this.client.get<{ data: Item[] }>('/items/blueprints');
    return response.data;
  }

  // Progress Tracking - User Endpoints
  async getMyQuestProgress(): Promise<{ data: UserQuestProgress[] }> {
    const response = await this.client.get<{ data: UserQuestProgress[] }>('/progress/quests');
    return response.data;
  }

  async updateMyQuestProgress(questExternalId: string, completed: boolean): Promise<UserQuestProgress> {
    const response = await this.client.put<UserQuestProgress>(`/progress/quests/${questExternalId}`, {
      completed,
    });
    return response.data;
  }

  async getMyHideoutProgress(): Promise<{ data: UserHideoutModuleProgress[] }> {
    const response = await this.client.get<{ data: UserHideoutModuleProgress[] }>('/progress/hideout-modules');
    return response.data;
  }

  async updateMyHideoutProgress(moduleExternalId: string, unlocked: boolean, level: number): Promise<UserHideoutModuleProgress> {
    const response = await this.client.put<UserHideoutModuleProgress>(`/progress/hideout-modules/${moduleExternalId}`, {
      unlocked,
      level,
    });
    return response.data;
  }

  async getMySkillNodeProgress(): Promise<{ data: UserSkillNodeProgress[] }> {
    const response = await this.client.get<{ data: UserSkillNodeProgress[] }>('/progress/skill-nodes');
    return response.data;
  }

  async updateMySkillNodeProgress(skillNodeExternalId: string, unlocked: boolean, level: number): Promise<UserSkillNodeProgress> {
    const response = await this.client.put<UserSkillNodeProgress>(`/progress/skill-nodes/${skillNodeExternalId}`, {
      unlocked,
      level,
    });
    return response.data;
  }

  async getMyBlueprintProgress(): Promise<{ data: UserBlueprintProgress[] }> {
    const response = await this.client.get<{ data: UserBlueprintProgress[] }>('/progress/blueprints');
    return response.data;
  }

  async updateMyBlueprintProgress(itemExternalId: string, consumed: boolean): Promise<UserBlueprintProgress> {
    const response = await this.client.put<UserBlueprintProgress>(`/progress/blueprints/${itemExternalId}`, {
      consumed,
    });
    return response.data;
  }

  // Progress Tracking - Admin Endpoints
  async getAllUserProgress(userId: number): Promise<AllUserProgress> {
    const response = await this.client.get<AllUserProgress>(`/admin/users/${userId}/progress`);
    return response.data;
  }

  async getUserQuestProgress(userId: number): Promise<{ data: UserQuestProgress[]; user_id: number }> {
    const response = await this.client.get<{ data: UserQuestProgress[]; user_id: number }>(`/admin/users/${userId}/progress/quests`);
    return response.data;
  }

  async updateUserQuestProgress(userId: number, questExternalId: string, completed: boolean): Promise<UserQuestProgress> {
    const response = await this.client.put<UserQuestProgress>(`/admin/users/${userId}/progress/quests/${questExternalId}`, {
      completed,
    });
    return response.data;
  }

  async getUserHideoutProgress(userId: number): Promise<{ data: UserHideoutModuleProgress[]; user_id: number }> {
    const response = await this.client.get<{ data: UserHideoutModuleProgress[]; user_id: number }>(`/admin/users/${userId}/progress/hideout-modules`);
    return response.data;
  }

  async updateUserHideoutProgress(userId: number, moduleExternalId: string, unlocked: boolean, level: number): Promise<UserHideoutModuleProgress> {
    const response = await this.client.put<UserHideoutModuleProgress>(`/admin/users/${userId}/progress/hideout-modules/${moduleExternalId}`, {
      unlocked,
      level,
    });
    return response.data;
  }

  async getUserSkillNodeProgress(userId: number): Promise<{ data: UserSkillNodeProgress[]; user_id: number }> {
    const response = await this.client.get<{ data: UserSkillNodeProgress[]; user_id: number }>(`/admin/users/${userId}/progress/skill-nodes`);
    return response.data;
  }

  async updateUserSkillNodeProgress(userId: number, skillNodeExternalId: string, unlocked: boolean, level: number): Promise<UserSkillNodeProgress> {
    const response = await this.client.put<UserSkillNodeProgress>(`/admin/users/${userId}/progress/skill-nodes/${skillNodeExternalId}`, {
      unlocked,
      level,
    });
    return response.data;
  }

  async getUserBlueprintProgress(userId: number): Promise<{ data: UserBlueprintProgress[]; user_id: number }> {
    const response = await this.client.get<{ data: UserBlueprintProgress[]; user_id: number }>(`/admin/users/${userId}/progress/blueprints`);
    return response.data;
  }

  async updateUserBlueprintProgress(userId: number, itemExternalId: string, consumed: boolean): Promise<UserBlueprintProgress> {
    const response = await this.client.put<UserBlueprintProgress>(`/admin/users/${userId}/progress/blueprints/${itemExternalId}`, {
      consumed,
    });
    return response.data;
  }

  // Appwrite Management
  async getAppwriteQuests(limit = 100): Promise<Quest[]> {
    return appwriteService.getQuests(limit);
  }

  async getAppwriteItems(limit = 100): Promise<Item[]> {
    return appwriteService.getItems(limit);
  }

  async getAppwriteSkillNodes(limit = 100): Promise<SkillNode[]> {
    return appwriteService.getSkillNodes(limit);
  }

  async getAppwriteHideoutModules(limit = 100): Promise<HideoutModule[]> {
    return appwriteService.getHideoutModules(limit);
  }

  async getAppwriteEnemyTypes(limit = 100): Promise<EnemyType[]> {
    return appwriteService.getEnemyTypes(limit);
  }

  async getAppwriteAlerts(limit = 100): Promise<Alert[]> {
    return appwriteService.getAlerts(limit);
  }

  async getAppwriteCounts(): Promise<Record<string, number>> {
    return appwriteService.getCounts();
  }

  // Force sync all data from API to Appwrite
  async forceSyncToAppwrite(): Promise<{ synced: number; errors: number; details: Record<string, { synced: number; errors: number }> }> {
    if (!isAppwriteEnabledSync()) {
      throw new Error('Appwrite is not enabled');
    }

    const result = {
      synced: 0,
      errors: 0,
      details: {} as Record<string, { synced: number; errors: number }>,
    };

    // Helper to sync a batch
    const syncBatch = async <T extends { external_id?: string; id?: number }>(
      items: T[],
      syncFn: (item: T, op: 'insert' | 'update') => Promise<void>,
      entityName: string
    ) => {
      let synced = 0;
      let errors = 0;

      for (const item of items) {
        try {
          // Try update first, if it fails, try insert
          try {
            await syncFn(item, 'update');
          } catch {
            await syncFn(item, 'insert');
          }
          synced++;
        } catch (error) {
          errors++;
          if (process.env.NODE_ENV === 'development') {
            console.error(`Error syncing ${entityName}:`, error);
          }
        }
      }

      result.details[entityName] = { synced, errors };
      result.synced += synced;
      result.errors += errors;
    };

    try {
      // Sync all entities - fetch all pages
      const fetchAll = async <T>(getFn: (page: number, limit: number) => Promise<PaginatedResponse<T>>): Promise<T[]> => {
        const all: T[] = [];
        let page = 1;
        let hasMore = true;
        while (hasMore) {
          const response = await getFn(page, 100);
          all.push(...response.data);
          hasMore = response.data.length === 100 && page * 100 < response.pagination.total;
          page++;
        }
        return all;
      };

      const [quests, items, skillNodes, hideoutModules, enemyTypes, alerts] = await Promise.all([
        fetchAll<Quest>((p, l) => this.getQuests(p, l)),
        fetchAll<Item>((p, l) => this.getItems(p, l)),
        fetchAll<SkillNode>((p, l) => this.getSkillNodes(p, l)),
        fetchAll<HideoutModule>((p, l) => this.getHideoutModules(p, l)),
        fetchAll<EnemyType>((p, l) => this.getEnemyTypes(p, l)),
        fetchAll<Alert>((p, l) => this.getAlerts(p, l)),
      ]);

      await Promise.all([
        syncBatch(quests, (q, op) => appwriteService.syncQuest(q, op), 'quests'),
        syncBatch(items, (i, op) => appwriteService.syncItem(i, op), 'items'),
        syncBatch(skillNodes, (s, op) => appwriteService.syncSkillNode(s, op), 'skillNodes'),
        syncBatch(hideoutModules, (h, op) => appwriteService.syncHideoutModule(h, op), 'hideoutModules'),
        syncBatch(enemyTypes, (e, op) => appwriteService.syncEnemyType(e, op), 'enemyTypes'),
        syncBatch(alerts, (a, op) => appwriteService.syncAlert(a, op), 'alerts'),
      ]);
    } catch (error) {
      throw error;
    }

    return result;
  }

  // Sync individual categories
  async syncQuestsToAppwrite(): Promise<{ synced: number; errors: number }> {
    if (!isAppwriteEnabledSync()) {
      throw new Error('Appwrite is not enabled');
    }

    const fetchAll = async <T>(getFn: (page: number, limit: number) => Promise<PaginatedResponse<T>>): Promise<T[]> => {
      const all: T[] = [];
      let page = 1;
      let hasMore = true;
      while (hasMore) {
        const response = await getFn(page, 100);
        all.push(...response.data);
        hasMore = response.data.length === 100 && page * 100 < response.pagination.total;
        page++;
      }
      return all;
    };

    const quests = await fetchAll<Quest>((p, l) => this.getQuests(p, l));
    let synced = 0;
    let errors = 0;

    for (const quest of quests) {
      try {
        try {
          await appwriteService.syncQuest(quest, 'update');
        } catch {
          await appwriteService.syncQuest(quest, 'insert');
        }
        synced++;
      } catch (error) {
        errors++;
      }
    }

    return { synced, errors };
  }

  async syncItemsToAppwrite(): Promise<{ synced: number; errors: number }> {
    if (!isAppwriteEnabledSync()) {
      throw new Error('Appwrite is not enabled');
    }

    const fetchAll = async <T>(getFn: (page: number, limit: number) => Promise<PaginatedResponse<T>>): Promise<T[]> => {
      const all: T[] = [];
      let page = 1;
      let hasMore = true;
      while (hasMore) {
        const response = await getFn(page, 100);
        all.push(...response.data);
        hasMore = response.data.length === 100 && page * 100 < response.pagination.total;
        page++;
      }
      return all;
    };

    const items = await fetchAll<Item>((p, l) => this.getItems(p, l));
    let synced = 0;
    let errors = 0;

    for (const item of items) {
      try {
        try {
          await appwriteService.syncItem(item, 'update');
        } catch {
          await appwriteService.syncItem(item, 'insert');
        }
        synced++;
      } catch (error) {
        errors++;
      }
    }

    return { synced, errors };
  }

  async syncSkillNodesToAppwrite(): Promise<{ synced: number; errors: number }> {
    if (!isAppwriteEnabledSync()) {
      throw new Error('Appwrite is not enabled');
    }

    const fetchAll = async <T>(getFn: (page: number, limit: number) => Promise<PaginatedResponse<T>>): Promise<T[]> => {
      const all: T[] = [];
      let page = 1;
      let hasMore = true;
      while (hasMore) {
        const response = await getFn(page, 100);
        all.push(...response.data);
        hasMore = response.data.length === 100 && page * 100 < response.pagination.total;
        page++;
      }
      return all;
    };

    const skillNodes = await fetchAll<SkillNode>((p, l) => this.getSkillNodes(p, l));
    let synced = 0;
    let errors = 0;

    for (const skillNode of skillNodes) {
      try {
        try {
          await appwriteService.syncSkillNode(skillNode, 'update');
        } catch {
          await appwriteService.syncSkillNode(skillNode, 'insert');
        }
        synced++;
      } catch (error) {
        errors++;
      }
    }

    return { synced, errors };
  }

  async syncHideoutModulesToAppwrite(): Promise<{ synced: number; errors: number }> {
    if (!isAppwriteEnabledSync()) {
      throw new Error('Appwrite is not enabled');
    }

    const fetchAll = async <T>(getFn: (page: number, limit: number) => Promise<PaginatedResponse<T>>): Promise<T[]> => {
      const all: T[] = [];
      let page = 1;
      let hasMore = true;
      while (hasMore) {
        const response = await getFn(page, 100);
        all.push(...response.data);
        hasMore = response.data.length === 100 && page * 100 < response.pagination.total;
        page++;
      }
      return all;
    };

    const hideoutModules = await fetchAll<HideoutModule>((p, l) => this.getHideoutModules(p, l));
    let synced = 0;
    let errors = 0;

    for (const module of hideoutModules) {
      try {
        try {
          await appwriteService.syncHideoutModule(module, 'update');
        } catch {
          await appwriteService.syncHideoutModule(module, 'insert');
        }
        synced++;
      } catch (error) {
        errors++;
      }
    }

    return { synced, errors };
  }

  async syncEnemyTypesToAppwrite(): Promise<{ synced: number; errors: number }> {
    if (!isAppwriteEnabledSync()) {
      throw new Error('Appwrite is not enabled');
    }

    const fetchAll = async <T>(getFn: (page: number, limit: number) => Promise<PaginatedResponse<T>>): Promise<T[]> => {
      const all: T[] = [];
      let page = 1;
      let hasMore = true;
      while (hasMore) {
        const response = await getFn(page, 100);
        all.push(...response.data);
        hasMore = response.data.length === 100 && page * 100 < response.pagination.total;
        page++;
      }
      return all;
    };

    const enemyTypes = await fetchAll<EnemyType>((p, l) => this.getEnemyTypes(p, l));
    let synced = 0;
    let errors = 0;

    for (const enemyType of enemyTypes) {
      try {
        try {
          await appwriteService.syncEnemyType(enemyType, 'update');
        } catch {
          await appwriteService.syncEnemyType(enemyType, 'insert');
        }
        synced++;
      } catch (error) {
        errors++;
      }
    }

    return { synced, errors };
  }

  async syncAlertsToAppwrite(): Promise<{ synced: number; errors: number }> {
    if (!isAppwriteEnabledSync()) {
      throw new Error('Appwrite is not enabled');
    }

    const fetchAll = async <T>(getFn: (page: number, limit: number) => Promise<PaginatedResponse<T>>): Promise<T[]> => {
      const all: T[] = [];
      let page = 1;
      let hasMore = true;
      while (hasMore) {
        const response = await getFn(page, 100);
        all.push(...response.data);
        hasMore = response.data.length === 100 && page * 100 < response.pagination.total;
        page++;
      }
      return all;
    };

    const alerts = await fetchAll<Alert>((p, l) => this.getAlerts(p, l));
    let synced = 0;
    let errors = 0;

    for (const alert of alerts) {
      try {
        try {
          await appwriteService.syncAlert(alert, 'update');
        } catch {
          await appwriteService.syncAlert(alert, 'insert');
        }
        synced++;
      } catch (error) {
        errors++;
      }
    }

    return { synced, errors };
  }

  // Data Export (CSV) - Admin only
  async exportData(
    type:
      | 'quests'
      | 'items'
      | 'skillNodes'
      | 'hideoutModules'
      | 'enemyTypes'
      | 'alerts'
      | 'bots'
      | 'maps'
      | 'repoTraders'
      | 'projects',
  ): Promise<string> {
    const endpointMap: Record<string, string> = {
      quests: '/admin/export/quests',
      items: '/admin/export/items',
      skillNodes: '/admin/export/skill-nodes',
      hideoutModules: '/admin/export/hideout-modules',
      enemyTypes: '/admin/export/enemy-types',
      alerts: '/admin/export/alerts',
      bots: '/admin/export/bots',
      maps: '/admin/export/maps',
      repoTraders: '/admin/export/traders',
      projects: '/admin/export/projects',
    };

    const endpoint = endpointMap[type];
    if (!endpoint) {
      throw new Error(`Invalid export type: ${type}`);
    }

    const response = await this.client.get(endpoint, {
      responseType: 'blob',
    });

    // Create a blob URL for the CSV data
    const blob = new Blob([response.data], { type: 'text/csv' });
    return URL.createObjectURL(blob);
  }
}

export const apiClient = new APIClient();

// Error helper
export function getErrorMessage(error: unknown): string {
  if (axios.isAxiosError(error)) {
    const axiosError = error as AxiosError<{ error?: string }>;
    return axiosError.response?.data?.error || axiosError.message || 'An error occurred';
  }
  if (error instanceof Error) {
    return error.message;
  }
  return 'An unknown error occurred';
}

