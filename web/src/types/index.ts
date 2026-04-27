export type Scope = 'global' | 'project';
export type MemoryType = 'fact' | 'rule' | 'decision' | 'feedback' | 'context';
export type Delivery = 'bootstrap' | 'pinned' | 'on_demand';

export interface Memory {
  id: string;
  content: string;
  scope: Scope;
  project?: string;
  type: MemoryType;
  delivery: Delivery;
  tags?: string[];
  weight: number;
  supersedes?: string;
  created_at: string;
  updated_at: string;
  ttl?: string;
}

export interface RecallResult {
  memory: Memory;
  score: number;
}

export interface StatsResult {
  total: number;
  by_scope: Record<string, number>;
  by_project: Record<string, number>;
  by_type: Record<string, number>;
  by_delivery: Record<string, number>;
}

export interface RememberInput {
  content: string;
  scope: Scope;
  project?: string;
  type: MemoryType;
  delivery?: Delivery;
  tags?: string[];
  weight?: number;
  supersedes?: string;
  ttl?: string;
}

export interface RecallInput {
  query: string;
  scope?: string;
  project?: string;
  limit?: number;
}

export interface PinnedPreview {
  markdown: string;
  stats: {
    global: number;
    project: number;
    tokens: number;
  };
}

export interface ContradictionMatch {
  memory: Memory;
  similarity: number;
}

export interface RememberResult {
  memory: Memory;
  contradictions?: ContradictionMatch[];
}
