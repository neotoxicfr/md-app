<script lang="ts">
  import { api, type Version } from '$lib/api';
  import { activeFileId, openFile } from '$lib/stores/files';
  import { X, History, RotateCcw, Clock } from 'lucide-svelte';

  let { isOpen = false, onClose }: { isOpen: boolean; onClose: () => void } = $props();

  let versions = $state<Version[]>([]);
  let loading = $state(false);
  let restoring = $state(false);
  let previewContent = $state<string | null>(null);
  let loadError = $state<string | null>(null);

  $effect(() => {
    if (isOpen && $activeFileId) {
      loadVersions();
    } else {
      versions = [];
      previewContent = null;
    }
  });

  async function loadVersions() {
    if (!$activeFileId) return;
    loading = true;
    try {
      const res = await api.listVersions($activeFileId);
      versions = res.versions;
    } catch (e) {
      versions = [];
      loadError = e instanceof Error ? e.message : 'Failed to load versions';
    } finally {
      loading = false;
    }
  }

  async function previewVersion(v: Version) {
    if (!$activeFileId) return;
    try {
      const vc = await api.getVersion($activeFileId, v.id);
      previewContent = vc.content;
    } catch (e) {
      previewContent = null;
      console.warn('Failed to load version preview:', e);
    }
  }

  async function restoreVersion(v: Version) {
    if (!$activeFileId) return;
    if (!confirm(`Restore to version from ${formatDate(v.created_at)}?`)) return;
    restoring = true;
    try {
      await api.restoreVersion($activeFileId, v.id);
      // Reload the file
      await openFile($activeFileId);
      onClose();
    } catch (e) {
      loadError = e instanceof Error ? e.message : 'Restore failed';
    } finally {
      restoring = false;
    }
  }

  function formatDate(iso: string): string {
    return new Intl.DateTimeFormat('fr-FR', {
      day: '2-digit', month: '2-digit', year: '2-digit',
      hour: '2-digit', minute: '2-digit', second: '2-digit',
    }).format(new Date(iso));
  }

  function formatSize(bytes: number): string {
    if (bytes < 1024) return `${bytes} B`;
    return `${(bytes / 1024).toFixed(1)} KB`;
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') onClose();
  }
</script>

{#if isOpen}
  <!-- svelte-ignore a11y_no_static_element_interactions -->
  <div class="modal-backdrop" onclick={onClose} onkeydown={handleKeydown}>
    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <div class="modal-content" onclick={(e) => e.stopPropagation()} onkeydown={handleKeydown}>
      <div class="modal-header">
        <div class="header-title">
          <History size={18} />
          <h2>Version History</h2>
        </div>
        <button class="btn-icon" onclick={onClose}><X size={18} /></button>
      </div>

      <div class="version-layout">
        <div class="version-list">
          {#if loading}
            <div class="version-empty">Loading history…</div>
          {:else if loadError}
            <div class="version-empty" style="color: var(--danger)">{loadError}</div>
          {:else if versions.length === 0}
            <div class="version-empty">
              <Clock size={24} />
              <p>No versions yet</p>
              <span>Versions are created automatically on save</span>
            </div>
          {:else}
            {#each versions as v (v.id)}
              <!-- svelte-ignore a11y_no_static_element_interactions a11y_click_events_have_key_events -->
              <div class="version-item" onclick={() => previewVersion(v)}>
                <div class="version-info">
                  <span class="version-date">{formatDate(v.created_at)}</span>
                  <span class="version-meta">{formatSize(v.size)} · {v.hash.slice(0, 8)}</span>
                  {#if v.message}
                    <span class="version-msg">{v.message}</span>
                  {/if}
                </div>
                <button
                  class="btn btn-sm"
                  disabled={restoring}
                  onclick={(e) => { e.stopPropagation(); restoreVersion(v); }}
                  title="Restore this version"
                >
                  <RotateCcw size={12} />
                  Restore
                </button>
              </div>
            {/each}
          {/if}
        </div>

        {#if previewContent !== null}
          <div class="version-preview">
            <div class="preview-header">Preview</div>
            <pre class="preview-code">{previewContent}</pre>
          </div>
        {/if}
      </div>
    </div>
  </div>
{/if}

<style>
  .modal-backdrop {
    position: fixed;
    inset: 0;
    z-index: 100;
    display: flex;
    align-items: center;
    justify-content: center;
    background: rgba(0, 0, 0, 0.6);
    backdrop-filter: blur(4px);
    animation: fadeIn 0.15s ease;
  }

  .modal-content {
    background: var(--bg-surface);
    border: 1px solid var(--border);
    border-radius: var(--radius-lg, 12px);
    width: 800px;
    max-width: 92vw;
    max-height: 80vh;
    display: flex;
    flex-direction: column;
    box-shadow: 0 20px 60px rgba(0, 0, 0, 0.4);
    overflow: hidden;
  }

  .modal-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 1rem 1.25rem;
    border-bottom: 1px solid var(--border-subtle);
  }

  .header-title {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    color: var(--text-primary);
  }

  .header-title h2 {
    font-family: var(--font-ui);
    font-size: 16px;
    font-weight: 700;
  }

  .version-layout {
    flex: 1;
    display: flex;
    overflow: hidden;
  }

  .version-list {
    width: 320px;
    flex-shrink: 0;
    overflow-y: auto;
    border-right: 1px solid var(--border-subtle);
  }

  .version-empty {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 0.5rem;
    padding: 2rem 1rem;
    color: var(--text-muted);
    font-family: var(--font-ui);
    font-size: 13px;
    text-align: center;
  }

  .version-item {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 0.5rem;
    padding: 0.65rem 1rem;
    width: 100%;
    background: transparent;
    border: none;
    border-bottom: 1px solid var(--border-subtle);
    cursor: pointer;
    text-align: left;
    font-family: var(--font-ui);
    transition: background 0.1s;
  }

  .version-item:hover {
    background: var(--bg-hover);
  }

  .version-info {
    display: flex;
    flex-direction: column;
    gap: 0.1rem;
    min-width: 0;
  }

  .version-date {
    font-size: 12.5px;
    font-weight: 600;
    color: var(--text-primary);
  }

  .version-meta {
    font-size: 11px;
    color: var(--text-muted);
  }

  .version-msg {
    font-size: 11px;
    color: var(--text-secondary);
    font-style: italic;
  }

  .btn-sm {
    font-size: 11px;
    padding: 0.2rem 0.5rem;
    display: flex;
    align-items: center;
    gap: 0.25rem;
    flex-shrink: 0;
  }

  .version-preview {
    flex: 1;
    display: flex;
    flex-direction: column;
    overflow: hidden;
  }

  .preview-header {
    padding: 0.5rem 1rem;
    font-family: var(--font-ui);
    font-size: 12px;
    font-weight: 600;
    color: var(--text-muted);
    border-bottom: 1px solid var(--border-subtle);
    text-transform: uppercase;
    letter-spacing: 0.05em;
  }

  .preview-code {
    flex: 1;
    padding: 1rem;
    overflow: auto;
    font-family: var(--font-mono, 'JetBrains Mono', monospace);
    font-size: 12px;
    line-height: 1.6;
    color: var(--text-secondary);
    white-space: pre-wrap;
    word-break: break-word;
    margin: 0;
    background: var(--bg-app);
  }

  @keyframes fadeIn {
    from { opacity: 0; }
    to { opacity: 1; }
  }
</style>
