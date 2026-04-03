import { useEffect, useState } from 'react';
import { useParams, useNavigate, Link } from 'react-router-dom';
import Markdown from 'react-markdown';
import type { Memory } from '../types';
import { api } from '../api/client';
import { ScopeBadge, TypeBadge } from '../components/Badge';

export function MemoryDetail() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const [memory, setMemory] = useState<Memory | null>(null);
  const [isEditing, setIsEditing] = useState(false);
  const [editContent, setEditContent] = useState('');
  const [error, setError] = useState('');

  useEffect(() => {
    if (!id) return;
    api.get(id).then((m) => {
      setMemory(m);
      setEditContent(m.content);
    }).catch((e) => setError(e.message));
  }, [id]);

  const handleSave = async () => {
    if (!id || !editContent.trim()) return;
    const updated = await api.update(id, editContent.trim());
    setMemory(updated);
    setIsEditing(false);
  };

  const handleDelete = async () => {
    if (!id) return;
    await api.delete(id);
    navigate('/memories');
  };

  if (error) return <p className="text-destructive text-sm">{error}</p>;
  if (!memory) return <p className="text-text-muted text-sm">Loading...</p>;

  return (
    <div className="max-w-2xl space-y-6">
      <button onClick={() => navigate('/memories')} className="text-sm text-text-muted hover:text-accent transition-colors">
        &larr; Back to memories
      </button>

      <div className="bg-surface rounded-[var(--radius)] p-6 border border-border space-y-4">
        <div className="flex items-center gap-2 flex-wrap">
          <ScopeBadge scope={memory.scope} />
          <TypeBadge type={memory.type} />
          {memory.project && <span className="text-xs text-text-muted">{memory.project}</span>}
          {memory.persona && <span className="text-xs text-text-muted">/ {memory.persona}</span>}
          {memory.supersedes && (
            <Link to={`/memories/${memory.supersedes}`} className="text-xs px-1.5 py-0.5 rounded bg-accent/10 text-accent hover:bg-accent/20 transition-colors">
              overrides {memory.supersedes.slice(0, 8)}...
            </Link>
          )}
        </div>

        {/* Weight bar */}
        <div className="flex items-center gap-3">
          <span className="text-xs font-medium text-text-muted w-12">Weight</span>
          <div className="flex-1 h-2 bg-border rounded-full overflow-hidden">
            <div
              className={`h-full rounded-full transition-all ${
                memory.weight >= 0.7 ? 'bg-accent' : memory.weight >= 0.4 ? 'bg-yellow-500' : 'bg-destructive'
              }`}
              style={{ width: `${Math.round(memory.weight * 100)}%` }}
            />
          </div>
          <span className="text-xs font-mono text-text-muted w-8 text-right">{memory.weight.toFixed(2)}</span>
        </div>

        {isEditing ? (
          <div className="space-y-3">
            <textarea
              value={editContent}
              onChange={(e) => setEditContent(e.target.value)}
              rows={6}
              className="w-full px-4 py-3 text-sm bg-bg border border-border rounded-[var(--radius-sm)] text-text resize-y"
            />
            <div className="flex gap-3">
              <button onClick={handleSave}
                className="px-4 py-2 text-sm font-medium text-white bg-accent hover:bg-accent-hover rounded-[var(--radius-sm)] transition-colors">
                Save
              </button>
              <button onClick={() => { setIsEditing(false); setEditContent(memory.content); }}
                className="px-4 py-2 text-sm text-text-secondary hover:text-text transition-colors">
                Cancel
              </button>
            </div>
          </div>
        ) : (
          <div className="text-sm text-text leading-relaxed prose prose-sm max-w-none">
            <Markdown>{memory.content}</Markdown>
          </div>
        )}

        {memory.tags && memory.tags.length > 0 && (
          <div className="flex gap-1 flex-wrap">
            {memory.tags.map((tag) => (
              <span key={tag} className="text-xs px-2 py-0.5 bg-bg rounded text-text-muted">{tag}</span>
            ))}
          </div>
        )}

        <div className="flex items-center justify-between pt-4 border-t border-border">
          <div className="text-xs text-text-muted space-y-1">
            <p>ID: <span className="font-mono">{memory.id}</span></p>
            <p>Created: {new Date(memory.created_at).toLocaleString()}</p>
            <p>Updated: {new Date(memory.updated_at).toLocaleString()}</p>
            {memory.ttl && <p>Expires: {new Date(memory.ttl).toLocaleString()}</p>}
          </div>
          <div className="flex gap-3">
            {!isEditing && (
              <button onClick={() => setIsEditing(true)}
                className="px-4 py-2 text-sm font-medium text-accent hover:bg-accent-light rounded-[var(--radius-sm)] transition-colors">
                Edit
              </button>
            )}
            <button onClick={handleDelete}
              className="px-4 py-2 text-sm font-medium text-destructive hover:bg-destructive-light rounded-[var(--radius-sm)] transition-colors">
              Delete
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}
