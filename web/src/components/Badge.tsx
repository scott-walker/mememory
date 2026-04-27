import type { Scope, MemoryType, Delivery } from '../types';

const SCOPE_COLORS: Record<Scope, string> = {
  global: 'bg-scope-global/15 text-scope-global',
  project: 'bg-scope-project/15 text-scope-project',
};

const TYPE_COLORS: Record<MemoryType, string> = {
  fact: 'bg-type-fact/15 text-type-fact',
  rule: 'bg-type-rule/15 text-type-rule',
  decision: 'bg-type-decision/15 text-type-decision',
  feedback: 'bg-type-feedback/15 text-type-feedback',
  context: 'bg-type-context/15 text-type-context',
};

export function ScopeBadge({ scope }: { scope: Scope }) {
  return (
    <span className={`inline-block px-2 py-0.5 rounded-md text-xs font-medium ${SCOPE_COLORS[scope]}`}>
      {scope}
    </span>
  );
}

export function TypeBadge({ type }: { type: MemoryType }) {
  return (
    <span className={`inline-block px-2 py-0.5 rounded-md text-xs font-medium ${TYPE_COLORS[type]}`}>
      {type}
    </span>
  );
}

export function DeliveryBadge({ delivery }: { delivery: Delivery }) {
  if (delivery === 'bootstrap') {
    return (
      <span className="inline-block px-2 py-0.5 rounded-md text-xs font-medium bg-amber-500/15 text-amber-600">
        bootstrap
      </span>
    );
  }
  if (delivery === 'pinned') {
    return (
      <span className="inline-block px-2 py-0.5 rounded-md text-xs font-medium bg-rose-500/15 text-rose-600">
        pinned
      </span>
    );
  }
  return null;
}
