export type Scope = 'global' | 'project' | 'persona';
export type MemoryType = 'fact' | 'rule' | 'decision' | 'feedback' | 'context';

export interface Memory {
  id: string;
  content: string;
  scope: Scope;
  project?: string;
  persona?: string;
  type: MemoryType;
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
  by_persona: Record<string, number>;
  by_type: Record<string, number>;
}

export interface RememberInput {
  content: string;
  scope: Scope;
  project?: string;
  persona?: string;
  type: MemoryType;
  tags?: string[];
  weight?: number;
  supersedes?: string;
  ttl?: string;
}

export interface RecallInput {
  query: string;
  scope?: string;
  project?: string;
  persona?: string;
  limit?: number;
}

export interface ContradictionMatch {
  memory: Memory;
  similarity: number;
}

export interface RememberResult {
  memory: Memory;
  contradictions?: ContradictionMatch[];
}
