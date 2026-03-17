export interface MemoryItem {
  id: number;
  host: string;
  category: string;
  key: string;
  value: string;
  metadata: string;
  created_at: string;
  updated_at: string;
}

export interface NoteItem {
  id: number;
  host: string;
  project: string;
  title: string;
  content: string;
  tags: string;
  created_at: string;
  updated_at: string;
}

export interface StatsData {
  memories: number;
  notes: number;
  categories: string[];
  hosts: string[];
}

export interface MemoriesResponse {
  items: MemoryItem[];
  total: number;
  page: number;
  limit: number;
}

export interface NotesResponse {
  items: NoteItem[];
  total: number;
  page: number;
  limit: number;
}

export interface MemorySearchItem {
  id: number;
  host: string;
  category: string;
  key: string;
  value: string;
  similarity: number;
}

export interface NoteSearchItem {
  id: number;
  host: string;
  project: string;
  title: string;
  content: string;
  similarity: number;
}

export interface SearchResponse {
  memories: MemorySearchItem[];
  notes: NoteSearchItem[];
}

export interface GraphNode {
  id: string;
  type: string;
  label: string;
  data: Record<string, string>;
}

export interface GraphEdge {
  id: string;
  source: string;
  target: string;
  type: string;
}

export interface GraphData {
  nodes: GraphNode[];
  edges: GraphEdge[];
}
