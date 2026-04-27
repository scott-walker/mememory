import { useEffect, useState, useCallback } from 'react';
import type { PinnedPreview as PinnedPreviewData } from '../types';
import { api } from '../api/client';

export function PinnedPreview() {
  const [project, setProject] = useState('');
  const [preview, setPreview] = useState<PinnedPreviewData | null>(null);
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const data = await api.pinnedPreview(project || undefined);
      setPreview(data);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setLoading(false);
    }
  }, [project]);

  useEffect(() => {
    load();
  }, [load]);

  return (
    <div className="space-y-6">
      <div className="flex items-baseline justify-between">
        <h2 className="text-xl font-semibold text-text">Pinned Preview</h2>
        <p className="text-xs text-text-muted">
          Exact payload the UserPromptSubmit hook would inject for the selected project.
        </p>
      </div>

      <div className="flex items-center gap-3">
        <input
          type="text"
          placeholder="Project (empty = global only)"
          value={project}
          onChange={(e) => setProject(e.target.value)}
          className="h-9 px-3 text-sm bg-surface border border-border rounded-[var(--radius-sm)] text-text placeholder:text-text-muted w-72"
        />
        <button
          onClick={load}
          disabled={loading}
          className="h-9 px-4 text-sm font-medium text-white bg-accent hover:bg-accent-hover rounded-[var(--radius-sm)] disabled:opacity-40 transition-colors"
        >
          {loading ? 'Rendering...' : 'Render'}
        </button>
      </div>

      {error && <p className="text-destructive text-sm">{error}</p>}

      {preview && (
        <>
          <div className="flex items-center gap-6 text-xs text-text-muted">
            <span>Global pinned: <span className="font-mono text-text">{preview.stats.global}</span></span>
            <span>Project pinned: <span className="font-mono text-text">{preview.stats.project}</span></span>
            <span>Tokens (est.): <span className="font-mono text-text">{preview.stats.tokens}</span></span>
          </div>

          {preview.markdown ? (
            <pre className="bg-surface border border-border rounded-[var(--radius)] p-5 text-xs text-text whitespace-pre-wrap font-mono leading-relaxed overflow-x-auto">
              {preview.markdown}
            </pre>
          ) : (
            <p className="text-sm text-text-muted text-center py-12">
              No pinned memories for this scope. The hook would emit nothing on each turn.
            </p>
          )}
        </>
      )}
    </div>
  );
}
