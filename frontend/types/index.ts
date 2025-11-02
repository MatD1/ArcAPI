export interface User {
  id: number;
  github_id?: string;
  email: string;
  username: string;
  role: 'admin' | 'user';
  created_at: string;
  updated_at: string;
}

export interface APIKey {
  id: number;
  user_id: number;
  name: string;
  last_used_at?: string;
  revoked_at?: string;
  created_at: string;
}

export interface JWTToken {
  id: number;
  user_id: number;
  expires_at: string;
  revoked_at?: string;
  created_at: string;
}

export interface Mission {
  id: number;
  external_id: string;
  name: string;
  description?: string;
  data?: Record<string, any>;
  synced_at: string;
  created_at: string;
  updated_at: string;
}

export interface Item {
  id: number;
  external_id: string;
  name: string;
  description?: string;
  image_url?: string;
  data?: Record<string, any>;
  synced_at: string;
  created_at: string;
  updated_at: string;
}

export interface SkillNode {
  id: number;
  external_id: string;
  name: string;
  description?: string;
  data?: Record<string, any>;
  synced_at: string;
  created_at: string;
  updated_at: string;
}

export interface HideoutModule {
  id: number;
  external_id: string;
  name: string;
  description?: string;
  data?: Record<string, any>;
  synced_at: string;
  created_at: string;
  updated_at: string;
}

export interface AuditLog {
  id: number;
  api_key_id?: number;
  jwt_token_id?: number;
  user_id?: number;
  endpoint: string;
  method: string;
  status_code: number;
  request_body?: Record<string, any>;
  response_time_ms: number;
  ip_address?: string;
  created_at: string;
}

export interface PaginatedResponse<T> {
  data: T[];
  pagination: {
    page: number;
    limit: number;
    total: number;
  };
}

export interface LoginResponse {
  token: string;
  user: User;
}

