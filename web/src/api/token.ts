import type { APITokenCreateRequest, APITokenCreateResponse, APITokenListResponse, UpdateTokenStatusRequest } from '../types/api_token';

const API_BASE_URL = '/api';

export async function listTokens(page: number = 1, pageSize: number = 10): Promise<APITokenListResponse> {
  const response = await fetch(`${API_BASE_URL}/admin/tokens?page=${page}&pageSize=${pageSize}`);
  if (!response.ok) {
    const error = await response.json();
    throw new Error(error.error || 'Failed to list tokens');
  }
  return response.json();
}

export async function createToken(data: APITokenCreateRequest): Promise<APITokenCreateResponse> {
  const response = await fetch(`${API_BASE_URL}/admin/tokens`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(data),
  });
  if (!response.ok) {
    const error = await response.json();
    throw new Error(error.error || 'Failed to create token');
  }
  return response.json();
}

export async function updateTokenStatus(id: number, data: UpdateTokenStatusRequest): Promise<void> {
  const response = await fetch(`${API_BASE_URL}/admin/token/status?id=${id}`, {
    method: 'PUT',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(data),
  });
  if (!response.ok) {
    const error = await response.json();
    throw new Error(error.error || 'Failed to update token status');
  }
}

export async function deleteToken(id: number): Promise<void> {
  const response = await fetch(`${API_BASE_URL}/admin/token?id=${id}`, {
    method: 'DELETE',
  });
  if (!response.ok) {
    const error = await response.json();
    throw new Error(error.error || 'Failed to delete token');
  }
}
