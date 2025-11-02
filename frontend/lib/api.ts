import axios, { AxiosInstance, AxiosError } from 'axios';
import type {
  User,
  APIKey,
  JWTToken,
  Mission,
  Item,
  SkillNode,
  HideoutModule,
  AuditLog,
  PaginatedResponse,
  LoginResponse,
} from '@/types';

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

  setAuth(apiKey: string, jwtToken: string) {
    this.apiKey = apiKey;
    this.jwtToken = jwtToken;
    if (typeof window !== 'undefined') {
      localStorage.setItem('api_key', apiKey);
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
    if (this.apiKey && this.jwtToken) {
      this.client.defaults.headers.common['X-API-Key'] = this.apiKey;
      this.client.defaults.headers.common['Authorization'] = `Bearer ${this.jwtToken}`;
    } else {
      delete this.client.defaults.headers.common['X-API-Key'];
      delete this.client.defaults.headers.common['Authorization'];
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

  // Missions
  async getMissions(page = 1, limit = 20): Promise<PaginatedResponse<Mission>> {
    const response = await this.client.get<PaginatedResponse<Mission>>('/missions', {
      params: { page, limit },
    });
    return response.data;
  }

  async getMission(id: number): Promise<Mission> {
    const response = await this.client.get<Mission>(`/missions/${id}`);
    return response.data;
  }

  async createMission(data: Partial<Mission>): Promise<Mission> {
    const response = await this.client.post<Mission>('/missions', data);
    return response.data;
  }

  async updateMission(id: number, data: Partial<Mission>): Promise<Mission> {
    const response = await this.client.put<Mission>(`/missions/${id}`, data);
    return response.data;
  }

  async deleteMission(id: number): Promise<void> {
    await this.client.delete(`/missions/${id}`);
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
    return response.data;
  }

  async updateItem(id: number, data: Partial<Item>): Promise<Item> {
    const response = await this.client.put<Item>(`/items/${id}`, data);
    return response.data;
  }

  async deleteItem(id: number): Promise<void> {
    await this.client.delete(`/items/${id}`);
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
    return response.data;
  }

  async updateSkillNode(id: number, data: Partial<SkillNode>): Promise<SkillNode> {
    const response = await this.client.put<SkillNode>(`/skill-nodes/${id}`, data);
    return response.data;
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
    return response.data;
  }

  async updateHideoutModule(id: number, data: Partial<HideoutModule>): Promise<HideoutModule> {
    const response = await this.client.put<HideoutModule>(`/hideout-modules/${id}`, data);
    return response.data;
  }

  async deleteHideoutModule(id: number): Promise<void> {
    await this.client.delete(`/hideout-modules/${id}`);
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

