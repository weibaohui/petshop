export interface APIToken {
  id: number;
  name: string;
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
  expiresDays?: number;
  permissions?: string;
}

export interface UpdateAPITokenStatusRequest {
  status: 'active' | 'disabled';
}
