import type { Scope, MemoryType, Delivery } from '../types';

interface Filters {
  scope: string;
  type: string;
  delivery: string;
  project: string;
}

interface FilterBarProps {
  filters: Filters;
  onChange: (filters: Filters) => void;
}

const SCOPES: Array<Scope | ''> = ['', 'global', 'project'];
const TYPES: Array<MemoryType | ''> = ['', 'fact', 'rule', 'decision', 'feedback', 'context'];
const DELIVERIES: Array<Delivery | ''> = ['', 'bootstrap', 'pinned', 'on_demand'];

export function FilterBar({ filters, onChange }: FilterBarProps) {
  const update = (key: keyof Filters, value: string) => {
    onChange({ ...filters, [key]: value });
  };

  return (
    <div className="flex items-center gap-3 flex-wrap">
      <Select
        label="Scope"
        value={filters.scope}
        options={SCOPES}
        onChange={(v) => update('scope', v)}
      />
      <Select
        label="Type"
        value={filters.type}
        options={TYPES}
        onChange={(v) => update('type', v)}
      />
      <Select
        label="Delivery"
        value={filters.delivery}
        options={DELIVERIES}
        onChange={(v) => update('delivery', v)}
      />
      <input
        type="text"
        placeholder="Project"
        value={filters.project}
        onChange={(e) => update('project', e.target.value)}
        className="h-9 px-3 text-sm bg-surface border border-border rounded-[var(--radius-sm)] text-text placeholder:text-text-muted w-32"
      />
    </div>
  );
}

function Select({ label, value, options, onChange }: {
  label: string;
  value: string;
  options: string[];
  onChange: (v: string) => void;
}) {
  return (
    <select
      value={value}
      onChange={(e) => onChange(e.target.value)}
      className="h-9 px-3 text-sm bg-surface border border-border rounded-[var(--radius-sm)] text-text appearance-none cursor-pointer"
    >
      <option value="">All {label}s</option>
      {options.filter(Boolean).map((opt) => (
        <option key={opt} value={opt}>{opt}</option>
      ))}
    </select>
  );
}
