import type { PetAllStar, VoteResponse, VoteStatus } from '../types/pet';

const API_BASE_URL = '/api';

export async function voteForPet(petId: number, userId: number): Promise<VoteResponse> {
  const response = await fetch(`${API_BASE_URL}/vote`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ petId, userId }),
  });
  if (!response.ok) {
    const error = await response.json();
    throw new Error(error.error || 'Failed to vote');
  }
  return response.json();
}

export async function getVoteStatus(petId: number, userId?: number): Promise<VoteStatus> {
  const params = new URLSearchParams();
  params.append('petId', petId.toString());
  if (userId) {
    params.append('userId', userId.toString());
  }
  const response = await fetch(`${API_BASE_URL}/vote/status?${params}`);
  if (!response.ok) {
    throw new Error('Failed to get vote status');
  }
  return response.json();
}

export async function getLeaderboard(): Promise<any[]> {
  const response = await fetch(`${API_BASE_URL}/vote/leaderboard`);
  if (!response.ok) {
    throw new Error('Failed to get leaderboard');
  }
  return response.json();
}

export async function getCurrentAllStar(): Promise<PetAllStar | null> {
  const response = await fetch(`${API_BASE_URL}/allstar`);
  if (response.status === 404) {
    return null;
  }
  if (!response.ok) {
    throw new Error('Failed to get current all-star');
  }
  return response.json();
}

export async function getAllStars(): Promise<PetAllStar[]> {
  const response = await fetch(`${API_BASE_URL}/allstars`);
  if (!response.ok) {
    throw new Error('Failed to get all-stars');
  }
  return response.json();
}

export async function electAllStar(): Promise<PetAllStar> {
  const response = await fetch(`${API_BASE_URL}/allstar`, {
    method: 'POST',
  });
  if (!response.ok) {
    throw new Error('Failed to elect all-star');
  }
  return response.json();
}