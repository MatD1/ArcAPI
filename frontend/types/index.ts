export interface User {
  id: number;
  github_id?: string;
  email: string;
  username: string;
  role: 'admin' | 'user';
  can_access_data: boolean;
  created_via_app: boolean;
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

export interface Quest {
  id: number;
  external_id: string;
  name: string;
  description?: string;
  trader?: string;
  objectives?: { objectives: string[] };
  reward_item_ids?: { reward_item_ids: Array<{ itemId: string; quantity: number }> };
  xp?: number;
  data?: Record<string, any>;
  synced_at: string;
  created_at: string;
  updated_at: string;
}

// Mission is deprecated, use Quest instead
export type Mission = Quest;

export interface Item {
  id: number;
  external_id: string;
  name: string;
  description?: string;
  type?: string;
  image_url?: string;
  image_filename?: string;
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
  impacted_skill?: string;
  known_value?: { known_value: any[] };
  category?: string;
  max_points?: number;
  icon_name?: string;
  is_major?: boolean;
  position?: { x: number; y: number };
  prerequisite_node_ids?: { prerequisite_node_ids: string[] };
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
  max_level?: number;
  levels?: {
    levels: Array<{
      level: number;
      requirementItemIds?: Array<{ itemId: string; quantity: number }>;
      prerequisites?: string[];
    }>;
  };
  data?: Record<string, any>;
  synced_at: string;
  created_at: string;
  updated_at: string;
}

export interface EnemyType {
  id: number;
  external_id: string;
  name: string;
  description?: string;
  type?: string;
  image_url?: string;
  image_filename?: string;
  weakpoints?: Record<string, any>;
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

export interface RequiredItemUsage {
  source_type: 'quest' | 'hideout_module';
  source_id: number;
  source_name: string;
  quantity: number;
  level?: number;
}

export interface RequiredItemResponse {
  item: Item;
  total_quantity: number;
  usages: RequiredItemUsage[];
}

export interface RequiredItemsResponse {
  data: RequiredItemResponse[];
  total: number;
}

