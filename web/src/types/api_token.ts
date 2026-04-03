export interface APIToken {
  id: number;
  name: string;
  description: string;
  status: 'active' | 'disabled';
  lastUsedAt: string | null;
  expiresAt: string | null;
  createdAt: string;
  updatedAt: string;
}

export interface APITokenCreateRequest {
  name: string;
  description?: string;
  expiresDays?: number;
}

export interface APITokenCreateResponse extends APIToken {
  token: string;
}

export interface APITokenListResponse {
  data: APIToken[];
  pagination: {
    page: number;
    pageSize: number;
    total: number;
    totalPages: number;
  };
}

export interface UpdateTokenStatusRequest {
  status: 'active' | 'disabled';
}
