import type { Pet, Category, PetFilter, PetsResponse } from '../types/pet';

const API_BASE_URL = '/api/v1';

export async function getPets(filter: PetFilter = {}): Promise<PetsResponse> {
  const params = new URLSearchParams();
  if (filter.type) params.append('type', filter.type);
  if (filter.status) params.append('status', filter.status);
  if (filter.search) params.append('search', filter.search);
  if (filter.minPrice !== undefined) params.append('minPrice', filter.minPrice.toString());
  if (filter.maxPrice !== undefined) params.append('maxPrice', filter.maxPrice.toString());
  if (filter.page !== undefined) params.append('page', filter.page.toString());
  if (filter.pageSize !== undefined) params.append('pageSize', filter.pageSize.toString());

  const response = await fetch(`${API_BASE_URL}/pets?${params}`);
  if (!response.ok) {
    throw new Error('Failed to fetch pets');
  }
  return response.json();
}

export async function getPetById(id: number): Promise<Pet> {
  const response = await fetch(`${API_BASE_URL}/pets/${id}`);
  if (!response.ok) {
    throw new Error('Failed to fetch pet');
  }
  return response.json();
}

export async function getCategories(): Promise<Category[]> {
  const response = await fetch(`${API_BASE_URL}/categories`);
  if (!response.ok) {
    throw new Error('Failed to fetch categories');
  }
  return response.json();
}
