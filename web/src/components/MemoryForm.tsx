import { useState } from 'react';
import type { RememberInput, Scope, MemoryType, Delivery } from '../types';

interface MemoryFormProps {
  onSubmit: (input: RememberInput) => Promise<void>;
  onCancel?: () => void;
  initialContent?: string;
}

export function MemoryForm({ onSubmit, onCancel, initialContent }: MemoryFormProps) {
  const [content, setContent] = useState(initialContent ?? '');
  const [scope, setScope] = useState<Scope>('global');
  const [type, setType] = useState<MemoryType>('fact');
  const [delivery, setDelivery] = useState<Delivery>('on_demand');
  const [project, setProject] = useState('');
  const [persona, setPersona] = useState('');
  const [tags, setTags] = useState('');
  const [ttl, setTtl] = useState('');
  const [weight, setWeight] = useState('1.0');
  const [supersedes, setSupersedes] = useState('');
  const [isSubmitting, setIsSubmitting] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!content.trim()) return;

    setIsSubmitting(true);
    try {
      await onSubmit({
        content: content.trim(),
        scope,
        type,
        delivery,
        project: project || undefined,
        persona: persona || undefined,
        tags: tags ? tags.split(',').map((t) => t.trim()).filter(Boolean) : undefined,
        weight: parseFloat(weight) || undefined,
        supersedes: supersedes || undefined,
        ttl: ttl || undefined,
      });
      setContent('');
      setTags('');
      setTtl('');
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <form onSubmit={handleSubmit} className="bg-surface rounded-[var(--radius)] p-6 border border-border space-y-4">
      <textarea
        value={content}
        onChange={(e) => setContent(e.target.value)}
        placeholder="Memory content..."
        rows={3}
        className="w-full px-4 py-3 text-sm bg-bg border border-border rounded-[var(--radius-sm)] text-text placeholder:text-text-muted resize-y"
      />

      <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
        <label className="space-y-1">
          <span className="text-xs font-medium text-text-muted">Scope</span>
          <select value={scope} onChange={(e) => setScope(e.target.value as Scope)}
            className="w-full h-9 px-3 text-sm bg-bg border border-border rounded-[var(--radius-sm)] text-text">
            <option value="global">global</option>
            <option value="project">project</option>
            <option value="persona">persona</option>
          </select>
        </label>

        <label className="space-y-1">
          <span className="text-xs font-medium text-text-muted">Type</span>
          <select value={type} onChange={(e) => setType(e.target.value as MemoryType)}
            className="w-full h-9 px-3 text-sm bg-bg border border-border rounded-[var(--radius-sm)] text-text">
            <option value="fact">fact</option>
            <option value="rule">rule</option>
            <option value="decision">decision</option>
            <option value="feedback">feedback</option>
            <option value="context">context</option>
          </select>
        </label>

        <label className="space-y-1">
          <span className="text-xs font-medium text-text-muted">Delivery</span>
          <select value={delivery} onChange={(e) => setDelivery(e.target.value as Delivery)}
            className="w-full h-9 px-3 text-sm bg-bg border border-border rounded-[var(--radius-sm)] text-text">
            <option value="on_demand">on_demand</option>
            <option value="bootstrap">bootstrap</option>
          </select>
        </label>

        <label className="space-y-1">
          <span className="text-xs font-medium text-text-muted">Project</span>
          <input value={project} onChange={(e) => setProject(e.target.value)}
            placeholder="e.g. match"
            className="w-full h-9 px-3 text-sm bg-bg border border-border rounded-[var(--radius-sm)] text-text placeholder:text-text-muted" />
        </label>
      </div>

      <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
        <label className="space-y-1">
          <span className="text-xs font-medium text-text-muted">Persona</span>
          <input value={persona} onChange={(e) => setPersona(e.target.value)}
            placeholder="e.g. architect"
            className="w-full h-9 px-3 text-sm bg-bg border border-border rounded-[var(--radius-sm)] text-text placeholder:text-text-muted" />
        </label>
      </div>

      <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
        <label className="space-y-1">
          <span className="text-xs font-medium text-text-muted">Tags (comma-separated)</span>
          <input value={tags} onChange={(e) => setTags(e.target.value)}
            placeholder="e.g. frontend, performance"
            className="w-full h-9 px-3 text-sm bg-bg border border-border rounded-[var(--radius-sm)] text-text placeholder:text-text-muted" />
        </label>

        <label className="space-y-1">
          <span className="text-xs font-medium text-text-muted">TTL</span>
          <input value={ttl} onChange={(e) => setTtl(e.target.value)}
            placeholder="e.g. 24h, 7d"
            className="w-full h-9 px-3 text-sm bg-bg border border-border rounded-[var(--radius-sm)] text-text placeholder:text-text-muted" />
        </label>

        <label className="space-y-1">
          <span className="text-xs font-medium text-text-muted">Weight</span>
          <input value={weight} onChange={(e) => setWeight(e.target.value)}
            type="number" min="0.1" max="1.0" step="0.1"
            className="w-full h-9 px-3 text-sm bg-bg border border-border rounded-[var(--radius-sm)] text-text placeholder:text-text-muted" />
        </label>

        <label className="space-y-1">
          <span className="text-xs font-medium text-text-muted">Supersedes ID</span>
          <input value={supersedes} onChange={(e) => setSupersedes(e.target.value)}
            placeholder="memory ID to replace"
            className="w-full h-9 px-3 text-sm bg-bg border border-border rounded-[var(--radius-sm)] text-text font-mono placeholder:font-sans placeholder:text-text-muted" />
        </label>
      </div>

      <div className="flex gap-3 pt-2">
        <button
          type="submit"
          disabled={isSubmitting || !content.trim()}
          className="px-5 py-2.5 text-sm font-medium text-white bg-accent hover:bg-accent-hover rounded-[var(--radius-sm)] disabled:opacity-40 transition-colors"
        >
          {isSubmitting ? 'Saving...' : 'Save Memory'}
        </button>
        {onCancel && (
          <button type="button" onClick={onCancel}
            className="px-5 py-2.5 text-sm font-medium text-text-secondary hover:text-text transition-colors">
            Cancel
          </button>
        )}
      </div>
    </form>
  );
}
