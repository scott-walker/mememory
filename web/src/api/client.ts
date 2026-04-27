import type { Memory, RecallResult, StatsResult, RememberInput, RememberResult, RecallInput, PinnedPreview } from '../types';

const BASE = '/api';

async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const res = await fetch(`${BASE}${path}`, {
    headers: { 'Content-Type': 'application/json' },
    ...options,
  });
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: res.statusText }));
    throw new Error(err.error || res.statusText);
  }
  return res.json();
}

export const api = {
  stats: () => request<StatsResult>('/stats'),

  list: (params?: Record<string, string>) => {
    const qs = params ? '?' + new URLSearchParams(params).toString() : '';
    return request<Memory[]>(`/memories${qs}`);
  },

  get: (id: string) => request<Memory>(`/memories/${id}`),

  create: (input: RememberInput) =>
    request<RememberResult>('/memories', { method: 'POST', body: JSON.stringify(input) }),

  update: (id: string, content: string) =>
    request<Memory>(`/memories/${id}`, { method: 'PUT', body: JSON.stringify({ content }) }),

  delete: (id: string) =>
    request<{ ok: boolean }>(`/memories/${id}`, { method: 'DELETE' }),

  search: (input: RecallInput) =>
    request<RecallResult[]>('/memories/search', { method: 'POST', body: JSON.stringify(input) }),

  bulkDelete: (params: Record<string, string>) => {
    const qs = '?' + new URLSearchParams(params).toString();
    return request<{ deleted: number }>(`/memories${qs}`, { method: 'DELETE' });
  },

  export: () =>
    request<Memory[]>('/memories/export', { method: 'POST' }),

  import: (memories: RememberInput[]) =>
    request<{ imported: number }>('/memories/import', { method: 'POST', body: JSON.stringify(memories) }),

  pinnedPreview: (project?: string) => {
    const qs = project ? '?' + new URLSearchParams({ project }).toString() : '';
    return request<PinnedPreview>(`/pinned/preview${qs}`);
  },
};
