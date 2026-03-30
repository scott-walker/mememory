import { useEffect, useState } from 'react';
import type { Memory, StatsResult } from '../types';
import { api } from '../api/client';
import { StatsCards } from '../components/StatsCards';
import { MemoryCard } from '../components/MemoryCard';

export function Dashboard() {
  const [stats, setStats] = useState<StatsResult | null>(null);
  const [recent, setRecent] = useState<Memory[]>([]);
  const [error, setError] = useState('');

  useEffect(() => {
    Promise.all([
      api.stats(),
      api.list({ limit: '5' }),
    ]).then(([s, m]) => {
      setStats(s);
      setRecent(m);
    }).catch((e) => setError(e.message));
  }, []);

  return (
    <div className="space-y-8">
      <h2 className="text-xl font-semibold text-text">Dashboard</h2>

      {error && <p className="text-destructive text-sm">{error}</p>}

      <StatsCards stats={stats} />

      {stats && (
        <div className="grid grid-cols-2 gap-4">
          <div className="bg-surface rounded-[var(--radius)] p-5 border border-border">
            <p className="text-text-muted text-xs font-medium uppercase tracking-wide mb-3">By Type</p>
            <div className="space-y-2">
              {Object.entries(stats.by_type).map(([type, count]) => (
                <div key={type} className="flex items-center justify-between">
                  <span className="text-sm text-text">{type}</span>
                  <span className="text-sm font-mono text-text-secondary">{count}</span>
                </div>
              ))}
            </div>
          </div>
          <div className="bg-surface rounded-[var(--radius)] p-5 border border-border">
            <p className="text-text-muted text-xs font-medium uppercase tracking-wide mb-3">By Project</p>
            <div className="space-y-2">
              {Object.entries(stats.by_project).length > 0 ? (
                Object.entries(stats.by_project).map(([project, count]) => (
                  <div key={project} className="flex items-center justify-between">
                    <span className="text-sm text-text">{project}</span>
                    <span className="text-sm font-mono text-text-secondary">{count}</span>
                  </div>
                ))
              ) : (
                <p className="text-sm text-text-muted">No project-scoped memories yet</p>
              )}
            </div>
          </div>
        </div>
      )}

      <div>
        <h3 className="text-base font-medium text-text mb-4">Recent Memories</h3>
        {recent.length > 0 ? (
          <div className="space-y-3">
            {recent.map((m) => (
              <MemoryCard key={m.id} memory={m} />
            ))}
          </div>
        ) : (
          <p className="text-sm text-text-muted">No memories stored yet</p>
        )}
      </div>
    </div>
  );
}
