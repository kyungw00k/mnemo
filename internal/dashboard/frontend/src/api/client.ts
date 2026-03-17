import type {
  StatsData,
  MemoriesResponse,
  NotesResponse,
  SearchResponse,
  GraphData,
  NoteItem,
} from '../types';

async function apiFetch<T>(path: string): Promise<T> {
  const res = await fetch(path);
  if (!res.ok) {
    throw new Error(`API error ${res.status}: ${res.statusText}`);
  }
  return res.json() as Promise<T>;
}

export function fetchStats(): Promise<StatsData> {
  return apiFetch<StatsData>('/api/stats');
}

export function fetchMemories(params: {
  host?: string;
  category?: string;
  page?: number;
  limit?: number;
}): Promise<MemoriesResponse> {
  const q = new URLSearchParams();
  if (params.host) q.set('host', params.host);
  if (params.category) q.set('category', params.category);
  if (params.page) q.set('page', String(params.page));
  if (params.limit) q.set('limit', String(params.limit));
  return apiFetch<MemoriesResponse>(`/api/memories?${q}`);
}

export function fetchNotes(params: {
  host?: string;
  project?: string;
  page?: number;
  limit?: number;
}): Promise<NotesResponse> {
  const q = new URLSearchParams();
  if (params.host) q.set('host', params.host);
  if (params.project) q.set('project', params.project);
  if (params.page) q.set('page', String(params.page));
  if (params.limit) q.set('limit', String(params.limit));
  return apiFetch<NotesResponse>(`/api/notes?${q}`);
}

export function fetchSearch(params: {
  q: string;
  type?: 'memory' | 'note' | 'all';
  host?: string;
  limit?: number;
}): Promise<SearchResponse> {
  const q = new URLSearchParams();
  q.set('q', params.q);
  if (params.type) q.set('type', params.type);
  if (params.host) q.set('host', params.host);
  if (params.limit) q.set('limit', String(params.limit));
  return apiFetch<SearchResponse>(`/api/search?${q}`);
}

export function fetchGraph(params: { host?: string }): Promise<GraphData> {
  const q = new URLSearchParams();
  if (params.host) q.set('host', params.host);
  return apiFetch<GraphData>(`/api/graph?${q}`);
}

export function fetchNoteDetail(id: number): Promise<NoteItem> {
  return apiFetch<NoteItem>(`/api/notes/${id}`);
}
