import { useState } from 'react';
import { api } from '../api/client';

export function Settings() {
  const [status, setStatus] = useState('');
  const [bulkScope, setBulkScope] = useState('');
  const [bulkProject, setBulkProject] = useState('');
  const [importData, setImportData] = useState('');

  const handleExport = async () => {
    setStatus('Exporting...');
    const memories = await api.export();
    const blob = new Blob([JSON.stringify(memories, null, 2)], { type: 'application/json' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = 'memories.json';
    a.click();
    URL.revokeObjectURL(url);
    setStatus(`Exported ${memories.length} memories`);
  };

  const handleImport = async () => {
    if (!importData.trim()) return;
    setStatus('Importing...');
    try {
      const memories = JSON.parse(importData);
      const result = await api.import(memories);
      setStatus(`Imported ${result.imported} memories`);
      setImportData('');
    } catch (e) {
      setStatus(`Import error: ${e instanceof Error ? e.message : 'invalid JSON'}`);
    }
  };

  const handleBulkDelete = async () => {
    const params: Record<string, string> = {};
    if (bulkScope) params.scope = bulkScope;
    if (bulkProject) params.project = bulkProject;

    if (Object.keys(params).length === 0) {
      setStatus('Select at least one filter for bulk delete');
      return;
    }

    setStatus('Deleting...');
    const result = await api.bulkDelete(params);
    setStatus(`Deleted ${result.deleted} memories`);
  };

  return (
    <div className="max-w-2xl space-y-8">
      <h2 className="text-xl font-semibold text-text">Settings</h2>

      {status && (
        <div className="bg-accent-light text-accent px-4 py-3 rounded-[var(--radius-sm)] text-sm">
          {status}
        </div>
      )}

      <section className="bg-surface rounded-[var(--radius)] p-6 border border-border space-y-4">
        <h3 className="text-base font-medium text-text">Export</h3>
        <p className="text-sm text-text-muted">Download all memories as a JSON file.</p>
        <button onClick={handleExport}
          className="px-5 py-2.5 text-sm font-medium text-white bg-accent hover:bg-accent-hover rounded-[var(--radius-sm)] transition-colors">
          Export All
        </button>
      </section>

      <section className="bg-surface rounded-[var(--radius)] p-6 border border-border space-y-4">
        <h3 className="text-base font-medium text-text">Import</h3>
        <p className="text-sm text-text-muted">Paste a JSON array of memory objects.</p>
        <textarea
          value={importData}
          onChange={(e) => setImportData(e.target.value)}
          rows={4}
          placeholder='[{"content": "...", "scope": "global", "type": "fact"}]'
          className="w-full px-4 py-3 text-sm bg-bg border border-border rounded-[var(--radius-sm)] text-text placeholder:text-text-muted resize-y font-mono"
        />
        <button onClick={handleImport} disabled={!importData.trim()}
          className="px-5 py-2.5 text-sm font-medium text-white bg-accent hover:bg-accent-hover rounded-[var(--radius-sm)] disabled:opacity-40 transition-colors">
          Import
        </button>
      </section>

      <section className="bg-surface rounded-[var(--radius)] p-6 border border-border space-y-4">
        <h3 className="text-base font-medium text-text text-destructive">Bulk Delete</h3>
        <p className="text-sm text-text-muted">Delete all memories matching the filters. This cannot be undone.</p>
        <div className="flex items-center gap-3">
          <select value={bulkScope} onChange={(e) => setBulkScope(e.target.value)}
            className="h-9 px-3 text-sm bg-bg border border-border rounded-[var(--radius-sm)] text-text">
            <option value="">All scopes</option>
            <option value="global">global</option>
            <option value="project">project</option>
          </select>
          <input value={bulkProject} onChange={(e) => setBulkProject(e.target.value)}
            placeholder="Project"
            className="h-9 px-3 text-sm bg-bg border border-border rounded-[var(--radius-sm)] text-text placeholder:text-text-muted w-40" />
          <button onClick={handleBulkDelete}
            className="px-5 py-2 text-sm font-medium text-white bg-destructive hover:bg-destructive/90 rounded-[var(--radius-sm)] transition-colors">
            Delete Matching
          </button>
        </div>
      </section>
    </div>
  );
}
