import type { StatsResult } from '../types';

interface StatsCardsProps {
  stats: StatsResult | null;
}

export function StatsCards({ stats }: StatsCardsProps) {
  if (!stats) return null;

  return (
    <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
      <Card label="Total Memories" value={stats.total} accent="accent" />
      {Object.entries(stats.by_scope).map(([scope, count]) => (
        <Card key={scope} label={scope} value={count} accent={
          scope === 'global' ? 'scope-global' : scope === 'project' ? 'scope-project' : 'scope-persona'
        } />
      ))}
    </div>
  );
}

function Card({ label, value, accent }: { label: string; value: number; accent: string }) {
  return (
    <div className="bg-surface rounded-[var(--radius)] p-5 border border-border">
      <p className="text-text-muted text-xs font-medium uppercase tracking-wide mb-1">{label}</p>
      <p className={`text-2xl font-bold text-${accent}`}>{value}</p>
    </div>
  );
}
