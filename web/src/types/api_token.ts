export interface APIToken {
  id: number;
  name: string;
  description?: string;
  token?: string; // 仅在创建时返回
  status: 'active' | 'disabled';
  createdAt: string;
  updatedAt: string;
  lastUsedAt?: string;
  expiresAt?: string;
  createdBy: number;
  permissions: string;
}

export interface APITokenListResponse {
  list: APIToken[];
  total: number;
}

export interface CreateAPITokenRequest {
  name: string;
  description?: string;
  expiresDays?: number;
  permissions?: string;
}

// Alias for API compatibility
export type APITokenCreateRequest = CreateAPITokenRequest;
export type APITokenCreateResponse = APIToken;

export interface UpdateAPITokenStatusRequest {
  status: 'active' | 'disabled';
}

// Alias for API compatibility
export type UpdateTokenStatusRequest = UpdateAPITokenStatusRequest;
