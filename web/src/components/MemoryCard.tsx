import { Link } from 'react-router-dom';
import Markdown from 'react-markdown';
import type { Memory } from '../types';
import { ScopeBadge, TypeBadge, DeliveryBadge } from './Badge';

interface MemoryCardProps {
  memory: Memory;
  score?: number;
  onDelete?: (id: string) => void;
}

export function MemoryCard({ memory, score, onDelete }: MemoryCardProps) {
  const timeAgo = formatTimeAgo(memory.created_at);

  return (
    <div className="bg-surface rounded-[var(--radius)] p-5 border border-border">
      <div className="flex items-start justify-between gap-4 mb-3">
        <div className="flex items-center gap-2 flex-wrap">
          <ScopeBadge scope={memory.scope} />
          <TypeBadge type={memory.type} />
          <DeliveryBadge delivery={memory.delivery} />
          {memory.weight !== undefined && memory.weight < 1 && (
            <WeightIndicator weight={memory.weight} />
          )}
          {memory.supersedes && (
            <span className="text-xs px-1.5 py-0.5 rounded bg-accent/10 text-accent" title={`Supersedes: ${memory.supersedes}`}>
              override
            </span>
          )}
          {memory.project && (
            <span className="text-xs text-text-muted">
              {memory.project}
            </span>
          )}
        </div>
        {score !== undefined && (
          <div className="flex items-center gap-2 shrink-0">
            <div className="w-16 h-1.5 bg-border rounded-full overflow-hidden">
              <div
                className="h-full bg-accent rounded-full"
                style={{ width: `${Math.round(score * 100)}%` }}
              />
            </div>
            <span className="text-xs text-text-muted font-mono">{score.toFixed(2)}</span>
          </div>
        )}
      </div>

      <Link to={`/memories/${memory.id}`} className="block">
        <div className="text-sm text-text leading-relaxed line-clamp-3 prose prose-sm max-w-none">
          <Markdown>{memory.content}</Markdown>
        </div>
      </Link>

      <div className="flex items-center justify-between mt-3 pt-3 border-t border-border">
        <div className="flex items-center gap-3">
          <span className="text-xs text-text-muted">{timeAgo}</span>
          {memory.tags && memory.tags.length > 0 && (
            <div className="flex gap-1">
              {memory.tags.map((tag) => (
                <span key={tag} className="text-xs px-1.5 py-0.5 bg-bg rounded text-text-muted">
                  {tag}
                </span>
              ))}
            </div>
          )}
        </div>
        {onDelete && (
          <button
            onClick={() => onDelete(memory.id)}
            className="text-xs text-text-muted hover:text-destructive transition-colors"
          >
            Delete
          </button>
        )}
      </div>
    </div>
  );
}

function WeightIndicator({ weight }: { weight: number }) {
  const pct = Math.round(weight * 100);
  const color = weight >= 0.7 ? 'bg-accent' : weight >= 0.4 ? 'bg-yellow-500' : 'bg-destructive';
  return (
    <div className="flex items-center gap-1.5" title={`Weight: ${weight.toFixed(2)}`}>
      <div className="w-10 h-1.5 bg-border rounded-full overflow-hidden">
        <div className={`h-full rounded-full ${color}`} style={{ width: `${pct}%` }} />
      </div>
      <span className="text-xs text-text-muted font-mono">{weight.toFixed(1)}</span>
    </div>
  );
}

function formatTimeAgo(dateStr: string): string {
  const date = new Date(dateStr);
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffMin = Math.floor(diffMs / 60000);
  const diffHr = Math.floor(diffMin / 60);
  const diffDay = Math.floor(diffHr / 24);

  if (diffMin < 1) return 'just now';
  if (diffMin < 60) return `${diffMin}m ago`;
  if (diffHr < 24) return `${diffHr}h ago`;
  if (diffDay < 30) return `${diffDay}d ago`;
  return date.toLocaleDateString();
}
