import { useState, useCallback } from 'react';
import type { RecallResult } from '../types';
import { api } from '../api/client';
import { MemoryCard } from '../components/MemoryCard';

export function Search() {
  const [query, setQuery] = useState('');
  const [project, setProject] = useState('');
  const [results, setResults] = useState<RecallResult[]>([]);
  const [isSearching, setIsSearching] = useState(false);
  const [hasSearched, setHasSearched] = useState(false);

  const handleSearch = useCallback(async () => {
    if (!query.trim()) return;
    setIsSearching(true);
    try {
      const r = await api.search({
        query: query.trim(),
        project: project || undefined,
        limit: 10,
      });
      setResults(r);
      setHasSearched(true);
    } finally {
      setIsSearching(false);
    }
  }, [query, project]);

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter') handleSearch();
  };

  return (
    <div className="space-y-6">
      <h2 className="text-xl font-semibold text-text">Semantic Search</h2>

      <div className="bg-surface rounded-[var(--radius)] p-5 border border-border space-y-4">
        <input
          type="text"
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          onKeyDown={handleKeyDown}
          placeholder="Search memories by meaning..."
          className="w-full h-11 px-4 text-sm bg-bg border border-border rounded-[var(--radius-sm)] text-text placeholder:text-text-muted"
          autoFocus
        />

        <div className="flex items-center gap-3">
          <input
            type="text"
            value={project}
            onChange={(e) => setProject(e.target.value)}
            placeholder="Project filter"
            className="h-9 px-3 text-sm bg-bg border border-border rounded-[var(--radius-sm)] text-text placeholder:text-text-muted w-40"
          />
          <button
            onClick={handleSearch}
            disabled={isSearching || !query.trim()}
            className="px-5 py-2 text-sm font-medium text-white bg-accent hover:bg-accent-hover rounded-[var(--radius-sm)] disabled:opacity-40 transition-colors"
          >
            {isSearching ? 'Searching...' : 'Search'}
          </button>
        </div>
      </div>

      {results.length > 0 && (
        <div className="space-y-3">
          <p className="text-xs text-text-muted">{results.length} results</p>
          {results.map((r) => (
            <MemoryCard key={r.memory.id} memory={r.memory} score={r.score} />
          ))}
        </div>
      )}

      {hasSearched && results.length === 0 && (
        <p className="text-sm text-text-muted text-center py-12">No matching memories found</p>
      )}
    </div>
  );
}
