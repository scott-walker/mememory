import { useEffect, useState, useCallback } from 'react';
import { Link } from 'react-router-dom';
import type { Memory, ContradictionMatch } from '../types';
import { api } from '../api/client';
import { MemoryCard } from '../components/MemoryCard';
import { MemoryForm } from '../components/MemoryForm';
import { FilterBar } from '../components/FilterBar';

export function MemoryList() {
  const [memories, setMemories] = useState<Memory[]>([]);
  const [filters, setFilters] = useState({ scope: '', type: '', delivery: '', project: '', persona: '' });
  const [showForm, setShowForm] = useState(false);
  const [contradictions, setContradictions] = useState<ContradictionMatch[]>([]);
  const [error, setError] = useState('');

  const load = useCallback(() => {
    const params: Record<string, string> = { limit: '100' };
    if (filters.scope) params.scope = filters.scope;
    if (filters.type) params.type = filters.type;
    if (filters.delivery) params.delivery = filters.delivery;
    if (filters.project) params.project = filters.project;
    if (filters.persona) params.persona = filters.persona;

    api.list(params).then(setMemories).catch((e) => setError(e.message));
  }, [filters]);

  useEffect(() => { load(); }, [load]);

  const handleDelete = async (id: string) => {
    await api.delete(id);
    load();
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h2 className="text-xl font-semibold text-text">Memories</h2>
        <button
          onClick={() => setShowForm(!showForm)}
          className="px-4 py-2 text-sm font-medium text-white bg-accent hover:bg-accent-hover rounded-[var(--radius-sm)] transition-colors"
        >
          {showForm ? 'Cancel' : '+ New Memory'}
        </button>
      </div>

      {error && <p className="text-destructive text-sm">{error}</p>}

      {showForm && (
        <MemoryForm
          onSubmit={async (input) => {
            const result = await api.create(input);
            if (result.contradictions && result.contradictions.length > 0) {
              setContradictions(result.contradictions);
            }
            setShowForm(false);
            load();
          }}
          onCancel={() => setShowForm(false)}
        />
      )}

      {contradictions.length > 0 && (
        <div className="bg-yellow-500/10 border border-yellow-500/30 rounded-[var(--radius)] p-5 space-y-3">
          <div className="flex items-center justify-between">
            <h3 className="text-sm font-semibold text-yellow-600">Potential Contradictions Detected</h3>
            <button onClick={() => setContradictions([])} className="text-xs text-text-muted hover:text-text">
              Dismiss
            </button>
          </div>
          <p className="text-xs text-text-muted">Similar memories found that may conflict with the one you just created:</p>
          {contradictions.map((c) => (
            <div key={c.memory.id} className="flex items-start gap-3 p-3 bg-surface rounded-[var(--radius-sm)] border border-border">
              <div className="flex-1 min-w-0">
                <Link to={`/memories/${c.memory.id}`} className="text-sm text-text hover:text-accent transition-colors line-clamp-2">
                  {c.memory.content}
                </Link>
                <p className="text-xs text-text-muted mt-1 font-mono">{c.memory.id.slice(0, 8)}...</p>
              </div>
              <div className="shrink-0 flex items-center gap-2">
                <div className="w-12 h-1.5 bg-border rounded-full overflow-hidden">
                  <div className="h-full bg-yellow-500 rounded-full" style={{ width: `${Math.round(c.similarity * 100)}%` }} />
                </div>
                <span className="text-xs font-mono text-yellow-600">{Math.round(c.similarity * 100)}%</span>
              </div>
            </div>
          ))}
        </div>
      )}

      <FilterBar filters={filters} onChange={setFilters} />

      <p className="text-xs text-text-muted">{memories.length} memories</p>

      <div className="space-y-3">
        {memories.map((m) => (
          <MemoryCard key={m.id} memory={m} onDelete={handleDelete} />
        ))}
      </div>

      {memories.length === 0 && (
        <p className="text-sm text-text-muted text-center py-12">No memories match the filters</p>
      )}
    </div>
  );
}
