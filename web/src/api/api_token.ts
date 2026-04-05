import type { APIToken, APITokenListResponse, CreateAPITokenRequest, UpdateAPITokenStatusRequest } from '../types/api_token';

const API_BASE_URL = '/api/admin';

// 获取存储的JWT token
function getAuthHeaders(): HeadersInit {
  const token = localStorage.getItem('jwt_token');
  return {
    'Content-Type': 'application/json',
    'Authorization': token ? `Bearer ${token}` : '',
  };
}

export async function listAPITokens(page: number = 1, limit: number = 10): Promise<APITokenListResponse> {
  const response = await fetch(`${API_BASE_URL}/tokens?page=${page}&limit=${limit}`, {
    headers: getAuthHeaders(),
  });
  if (!response.ok) {
    if (response.status === 401) {
      throw new Error('未授权，请先登录');
    }
    throw new Error('获取API Token列表失败');
  }
  return response.json();
}

export async function createAPIToken(data: CreateAPITokenRequest): Promise<APIToken> {
  const response = await fetch(`${API_BASE_URL}/tokens`, {
    method: 'POST',
    headers: getAuthHeaders(),
    body: JSON.stringify(data),
  });
  if (!response.ok) {
    if (response.status === 401) {
      throw new Error('未授权，请先登录');
    }
    throw new Error('创建API Token失败');
  }
  return response.json();
}

export async function updateAPITokenStatus(id: number, data: UpdateAPITokenStatusRequest): Promise<void> {
  const response = await fetch(`${API_BASE_URL}/token?id=${id}`, {
    method: 'PUT',
    headers: getAuthHeaders(),
    body: JSON.stringify(data),
  });
  if (!response.ok) {
    if (response.status === 401) {
      throw new Error('未授权，请先登录');
    }
    throw new Error('更新API Token状态失败');
  }
}

export async function deleteAPIToken(id: number): Promise<void> {
  const response = await fetch(`${API_BASE_URL}/token?id=${id}`, {
    method: 'DELETE',
    headers: getAuthHeaders(),
  });
  if (!response.ok) {
    if (response.status === 401) {
      throw new Error('未授权，请先登录');
    }
    throw new Error('删除API Token失败');
  }
}
