// Pet types
export interface VaccinationRecord {
  name: string;
  date: string;
  completed: boolean;
}

export interface Pet {
  id: number;
  name: string;
  type: string;
  breed: string;
  photoUrls: string[];
  status: 'available' | 'pending' | 'sold';
  age: number;
  ageDisplay: string;
  price: number;
  description: string;
  healthStatus: string;
  vaccinationRecords: VaccinationRecord[];
  createdAt: string;
  voteCount?: number;
}

export interface Category {
  id: number;
  name: string;
}

export interface PetFilter {
  type?: string;
  status?: string;
  search?: string;
  minPrice?: number;
  maxPrice?: number;
  page?: number;
  pageSize?: number;
}

export interface PageInfo {
  page: number;
  page_size: number;
  total: number;
  total_pages: number;
}

export interface PetsResponse {
  data: Pet[];
  page: PageInfo;
}

export const StatusMap: Record<string, { text: string; color: string }> = {
  available: { text: '待售', color: 'green' },
  pending: { text: '已预订', color: 'orange' },
  sold: { text: '已售', color: 'red' },
};

export interface PetAllStar {
  id: number;
  petId: number;
  pet: Pet;
  voteCount: number;
  electedAt: string;
  period: string;
}

export interface VoteResponse {
  message: string;
  voteCount: number;
  hasVoted: boolean;
}

export interface VoteStatus {
  petId: number;
  voteCount: number;
  hasVoted: boolean;
}
