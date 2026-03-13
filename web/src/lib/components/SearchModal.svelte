<script lang="ts">
  import { api, type SearchResult } from '$lib/api';
  import { openFile } from '$lib/stores/files';
  import { Search, X, FileText } from 'lucide-svelte';

  let { isOpen = false, onClose }: { isOpen: boolean; onClose: () => void } = $props();

  let query = $state('');
  let results = $state<SearchResult[]>([]);
  let searching = $state(false);
  let searched = $state(false);
  let searchError = $state<string | null>(null);
  let debounceTimer: ReturnType<typeof setTimeout>;
  let inputEl = $state<HTMLInputElement | undefined>(undefined);

  $effect(() => {
    if (isOpen) {
      query = '';
      results = [];
      searched = false;
      const focusTimer = setTimeout(() => inputEl?.focus(), 50);
      return () => { clearTimeout(focusTimer); clearTimeout(debounceTimer); };
    }
  });

  function handleInput() {
    clearTimeout(debounceTimer);
    searchError = null;
    if (query.trim().length < 2) {
      results = [];
      searched = false;
      return;
    }
    debounceTimer = setTimeout(doSearch, 300);
  }

  async function doSearch() {
    if (query.trim().length < 2) return;
    searching = true;
    searchError = null;
    try {
      const res = await api.search(query.trim());
      results = res.results;
      searched = true;
    } catch (e) {
      results = [];
      searchError = e instanceof Error ? e.message : 'Search failed';
    } finally {
      searching = false;
    }
  }

  function selectResult(r: SearchResult) {
    openFile(r.file_id);
    onClose();
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') onClose();
    if (e.key === 'Enter' && results.length > 0) {
      selectResult(results[0]);
    }
  }

  function highlightMatch(text: string, q: string): string {
    if (!q) return text;
    text = text.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;');
    const escaped = q.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
    return text.replace(new RegExp(`(${escaped})`, 'gi'), '<mark>$1</mark>');
  }
</script>

{#if isOpen}
  <!-- svelte-ignore a11y_no_static_element_interactions -->
  <div class="modal-backdrop" onclick={onClose} onkeydown={handleKeydown}>
    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <div class="modal-content" onclick={(e) => e.stopPropagation()} onkeydown={handleKeydown}>
      <div class="search-input-row">
        <Search size={16} />
        <input
          bind:this={inputEl}
          type="text"
          placeholder="Search across all files…"
          bind:value={query}
          oninput={handleInput}
          class="search-field"
        />
        <button class="btn-icon" onclick={onClose}><X size={16} /></button>
      </div>

      <div class="search-results">
        {#if searching}
          <div class="search-status">Searching…</div>
        {:else if searchError}
          <div class="search-status" style="color: var(--danger)">{searchError}</div>
        {:else if searched && results.length === 0}
          <div class="search-status">No results for "{query}"</div>
        {:else}
          {#each results as r, i (r.file_id + '-' + r.line + '-' + i)}
            <button class="result-item" onclick={() => selectResult(r)}>
              <FileText size={14} />
              <div class="result-info">
                <span class="result-name">{r.name}</span>
                {#if r.line > 0}
                  <span class="result-line">Line {r.line}</span>
                {/if}
                <span class="result-snippet">{@html highlightMatch(r.snippet, query)}</span>
              </div>
            </button>
          {/each}
        {/if}
      </div>

      <div class="search-hint">
        <kbd>Enter</kbd> to open first result &middot; <kbd>Esc</kbd> to close
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
    align-items: flex-start;
    justify-content: center;
    padding-top: 15vh;
    background: rgba(0, 0, 0, 0.5);
    backdrop-filter: blur(4px);
    animation: fadeIn 0.1s ease;
  }

  .modal-content {
    background: var(--bg-surface);
    border: 1px solid var(--border);
    border-radius: var(--radius-lg, 12px);
    width: 560px;
    max-width: 92vw;
    max-height: 60vh;
    display: flex;
    flex-direction: column;
    box-shadow: 0 20px 60px rgba(0, 0, 0, 0.45);
    overflow: hidden;
  }

  .search-input-row {
    display: flex;
    align-items: center;
    gap: 0.6rem;
    padding: 0.65rem 1rem;
    border-bottom: 1px solid var(--border-subtle);
    color: var(--text-muted);
  }

  .search-field {
    flex: 1;
    background: transparent;
    border: none;
    outline: none;
    font-family: var(--font-ui);
    font-size: 14px;
    color: var(--text-primary);
  }
  .search-field::placeholder { color: var(--text-muted); }

  .search-results {
    flex: 1;
    overflow-y: auto;
    padding: 0.35rem 0;
  }

  .search-status {
    text-align: center;
    color: var(--text-muted);
    font-size: 13px;
    padding: 1.5rem 0;
    font-family: var(--font-ui);
  }

  .result-item {
    display: flex;
    align-items: flex-start;
    gap: 0.65rem;
    padding: 0.5rem 1rem;
    width: 100%;
    background: transparent;
    border: none;
    cursor: pointer;
    text-align: left;
    font-family: var(--font-ui);
    color: var(--text-primary);
    transition: background 0.1s;
  }

  .result-item:hover {
    background: var(--bg-hover);
  }

  .result-info {
    flex: 1;
    display: flex;
    flex-direction: column;
    min-width: 0;
  }

  .result-name {
    font-weight: 600;
    font-size: 13px;
  }

  .result-line {
    font-size: 11px;
    color: var(--text-muted);
  }

  .result-snippet {
    font-size: 12px;
    color: var(--text-secondary);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .result-snippet :global(mark) {
    background: var(--accent-surface);
    color: var(--accent);
    border-radius: 2px;
    padding: 0 1px;
  }

  .search-hint {
    display: flex;
    gap: 0.5rem;
    align-items: center;
    justify-content: center;
    padding: 0.4rem 1rem;
    border-top: 1px solid var(--border-subtle);
    font-family: var(--font-ui);
    font-size: 11px;
    color: var(--text-muted);
  }

  .search-hint kbd {
    background: var(--bg-app);
    border: 1px solid var(--border-subtle);
    border-radius: 3px;
    padding: 0.05rem 0.35rem;
    font-size: 10px;
    font-family: var(--font-mono, monospace);
  }

  @keyframes fadeIn {
    from { opacity: 0; }
    to { opacity: 1; }
  }
</style>
