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
import { db } from './sqlite/db';
import * as sqliteSchema from './sqlite/schema';
import { checkHydration, addToOutbox } from './sqlite/manager';
import { startOutboxSync } from './sqlite/sync_manager';
import { eq } from 'drizzle-orm';

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
  private syncInProgress: boolean = false;

  constructor() {
    this.client = axios.create({
      baseURL: `${API_URL}/api/v1`,
      headers: {
        'Content-Type': 'application/json',
      },
    });

    // Load from localStorage (JWT tokens only - API key no longer persisted for security)
    if (typeof window !== 'undefined') {
      // API key is NOT loaded from localStorage for security reasons
      // It will only be available in memory during the current session
      this.jwtToken = localStorage.getItem('supabase_token');
      this.updateHeaders();
      this.setupInterceptors();

      // Initialize Local DB & Sync
      this.initLocalDB();
    }
  }

  private async initLocalDB() {
    if (this.syncInProgress) {
      console.log('Sync already in progress, skipping...');
      return;
    }
    this.syncInProgress = true;
    try {
      await checkHydration(this.client);
      startOutboxSync(this.client);
    } catch (error) {
      console.error('Failed to initialize local DB:', error);
    } finally {
      this.syncInProgress = false;
    }
  }

  private setupInterceptors() {
    // Response interceptor to handle 401 errors
    this.client.interceptors.response.use(
      (response) => response,
      async (error) => {
        // Only clear auth on 401 if it's NOT the snapshot endpoint
        // (If snapshot 401s, we should just fail hydration quietly)
        if (error.response?.status === 401) {
          const isSnapshot = error.config?.url?.includes('/sync/snapshot');
          if (!isSnapshot) {
            console.warn('Unauthorized request, clearing auth state');
            this.clearAuth();
          } else {
             console.warn('Unauthorized snapshot request - skipping hydration');
          }
        }
        return Promise.reject(error);
      }
    );
  }


  setAuth(apiKey: string | null, jwtToken: string) {
    this.apiKey = apiKey;
    this.jwtToken = jwtToken;

    if (typeof window !== 'undefined') {
      localStorage.setItem('supabase_token', jwtToken);
    }

    this.updateHeaders();
  }

  setSupabaseToken(jwtToken: string) {
    this.jwtToken = jwtToken;

    if (typeof window !== 'undefined') {
      localStorage.setItem('supabase_token', jwtToken);
    }

    this.updateHeaders();
  }

  getIdToken() {
    return this.jwtToken;
  }

  getApiKey() {
    // Return API key from memory (not from localStorage for security)
    return this.apiKey;
  }

  clearAuth() {
    this.apiKey = null;
    this.jwtToken = null;
    if (typeof window !== 'undefined') {
      localStorage.removeItem('supabase_token');
      localStorage.removeItem('auth-storage');
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

  async getCurrentUser(): Promise<User> {
    const response = await this.client.get<{ user: User }>('/me');
    return response.data.user;
  }

  async refreshUserRole(): Promise<{ user: User; role_updated: boolean; message: string }> {
    const response = await this.client.post<{ user: User; role_updated: boolean; message: string }>('/me/refresh-role');
    return response.data;
  }

  // Quests
  async getQuests(page = 1, limit = 20): Promise<PaginatedResponse<Quest>> {
    try {
      // Try local first
      const localQuests = await db.select().from(sqliteSchema.quests).offset((page - 1) * limit).limit(limit);
      if (localQuests.length > 0) {
        return {
          data: localQuests.map(q => q.data as Quest),
          pagination: {
            total: localQuests.length,
            page,
            limit
          }
        };
      }
    } catch (e) {
      console.warn('Local quest fetch failed, falling back to API', e);
    }

    const response = await this.client.get<PaginatedResponse<Quest>>('/quests', {
      params: { page, limit },
    });
    return response.data;
  }

  async getQuest(id: number | string): Promise<Quest> {
    try {
      const local = await db.select().from(sqliteSchema.quests)
        .where(eq(typeof id === 'number' ? sqliteSchema.quests.id : sqliteSchema.quests.external_id, id as any))
        .limit(1);
      if (local.length > 0) return local[0].data as Quest;
    } catch (e) { }

    const response = await this.client.get<Quest>(`/quests/${id}`);
    return response.data;
  }

  async createQuest(data: Partial<Quest>): Promise<Quest> {
    const response = await this.client.post<Quest>('/quests', data);
    const quest = response.data;
    return quest;
  }

  async updateQuest(id: number, data: Partial<Quest>): Promise<Quest> {
    const response = await this.client.put<Quest>(`/quests/${id}`, data);
    const quest = response.data;
    return quest;
  }

  async deleteQuest(id: number): Promise<void> {
    await this.client.delete(`/quests/${id}`);
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
    return item;
  }

  async updateItem(id: number, data: Partial<Item>): Promise<Item> {
    const response = await this.client.put<Item>(`/items/${id}`, data);
    const item = response.data;
    return item;
  }

  async deleteItem(id: number): Promise<void> {
    await this.client.delete(`/items/${id}`);
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
    return skillNode;
  }

  async updateSkillNode(id: number, data: Partial<SkillNode>): Promise<SkillNode> {
    const response = await this.client.put<SkillNode>(`/skill-nodes/${id}`, data);
    const skillNode = response.data;
    return skillNode;
  }

  async deleteSkillNode(id: number): Promise<void> {
    await this.client.delete(`/skill-nodes/${id}`);
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
    return module;
  }

  async updateHideoutModule(id: number, data: Partial<HideoutModule>): Promise<HideoutModule> {
    const response = await this.client.put<HideoutModule>(`/hideout-modules/${id}`, data);
    const module = response.data;
    return module;
  }

  async deleteHideoutModule(id: number): Promise<void> {
    await this.client.delete(`/hideout-modules/${id}`);
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
    return enemyType;
  }

  async updateEnemyType(id: number, data: Partial<EnemyType>): Promise<EnemyType> {
    const response = await this.client.put<EnemyType>(`/enemy-types/${id}`, data);
    const enemyType = response.data;
    return enemyType;
  }

  async deleteEnemyType(id: number): Promise<void> {
    await this.client.delete(`/enemy-types/${id}`);
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

  // GitHub Data (Bots, Maps, Traders, Projects) - Now from database
  async getBots(offset = 0, limit = 20): Promise<PaginatedResponse<any>> {
    const response = await this.client.get('/bots', {
      params: { offset, limit },
    });
    return response.data;
  }

  async getMaps(offset = 0, limit = 20): Promise<PaginatedResponse<any>> {
    const response = await this.client.get('/maps', {
      params: { offset, limit },
    });
    return response.data;
  }

  async getTraders(offset = 0, limit = 20): Promise<PaginatedResponse<any>> {
    const response = await this.client.get('/repo-traders', {
      params: { offset, limit },
    });
    return response.data;
  }

  async getProjects(offset = 0, limit = 20): Promise<PaginatedResponse<any>> {
    const response = await this.client.get('/projects', {
      params: { offset, limit },
    });
    return response.data;
  }

  async getAlert(id: number): Promise<Alert> {
    const response = await this.client.get<Alert>(`/alerts/${id}`);
    return response.data;
  }

  async createAlert(data: Partial<Alert>): Promise<Alert> {
    const response = await this.client.post<Alert>('/alerts', data);
    const alert = response.data;
    return alert;
  }

  async updateAlert(id: number, data: Partial<Alert>): Promise<Alert> {
    const response = await this.client.put<Alert>(`/alerts/${id}`, data);
    const alert = response.data;
    return alert;
  }

  async deleteAlert(id: number): Promise<void> {
    await this.client.delete(`/alerts/${id}`);
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

  async updateUserRole(userId: number, role: string): Promise<{ message: string; user: User }> {
    const response = await this.client.put<{ message: string; user: User }>(`/admin/users/${userId}/role`, {
      role,
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
    // 1. Update/Add to local outbox for offline sync
    await addToOutbox('quest_progress', questExternalId, 'upsert', { completed });

    // 2. Optimistic Update (Backend call)
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

