<script lang="ts">
  import { activeFileId, activeName, activeContent } from '$lib/stores/files';
  import { api } from '$lib/api';
  import { X, Download, Loader } from 'lucide-svelte';
  import { get } from 'svelte/store';

  let { isOpen, onClose }: { isOpen: boolean; onClose: () => void } = $props();

  const formats = [
    { id: 'markdown', label: 'Markdown (.md)', desc: 'Source Markdown file', icon: '✍️' },
    { id: 'html', label: 'HTML', desc: 'Standalone web page', icon: '🌐' },
    { id: 'pdf', label: 'PDF', desc: 'Portable document', icon: '📄' },
    { id: 'docx', label: 'Word (.docx)', desc: 'Microsoft Word', icon: '📝' },
    { id: 'odt', label: 'OpenDocument (.odt)', desc: 'LibreOffice', icon: '📃' },
    { id: 'epub', label: 'EPUB', desc: 'E-book format', icon: '📚' },
    { id: 'latex', label: 'LaTeX (.tex)', desc: 'LaTeX source', icon: '🔣' },
    { id: 'rst', label: 'reStructuredText', desc: 'Python docs', icon: '🐍' },
    { id: 'asciidoc', label: 'AsciiDoc (.adoc)', desc: 'Technical docs', icon: '📋' },
    { id: 'textile', label: 'Textile', desc: 'Lightweight markup', icon: '🧵' },
    { id: 'mediawiki', label: 'MediaWiki', desc: 'Wikipedia format', icon: '📖' },
    { id: 'plain', label: 'Plain text (.txt)', desc: 'Strip formatting', icon: '📜' },
  ];

  let exporting: string | null = $state(null);
  let exportError: string | null = $state(null);
  let pdfMargin: string = $state('standard');

  const marginOptions = [
    { id: 'standard', label: 'Standard', desc: '2.5 cm' },
    { id: 'narrow', label: 'Narrow', desc: '1.5 cm' },
    { id: 'wide', label: 'Wide', desc: '3.5 cm' },
  ];

  function downloadBlob(blob: Blob, filename: string): void {
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = filename;
    a.click();
    setTimeout(() => URL.revokeObjectURL(url), 5000);
  }

  async function handleExport(formatId: string): Promise<void> {
    const id = get(activeFileId);
    const name = get(activeName) || 'document';
    const content = get(activeContent);
    const ext = getExtension(formatId);

    exporting = formatId;
    exportError = null;

    try {
      // Markdown: pure client-side — content is already Markdown, no server call needed
      if (formatId === 'markdown') {
        const blob = new Blob([content ?? ''], { type: 'text/markdown;charset=utf-8' });
        downloadBlob(blob, `${name}${ext}`);
        exporting = null;
        return;
      }

      if (id) {
        // File is saved — use the saved-file export endpoint
        if (formatId === 'html') {
          const a = document.createElement('a');
          a.href = api.exportHTML(id);
          a.download = `${name}${ext}`;
          a.click();
        } else {
          const margin = formatId === 'pdf' ? pdfMargin : undefined;
          const res = await fetch(api.exportFormat(id, formatId, margin), { method: 'POST' });
          if (!res.ok) {
            const err = await res.json().catch(() => ({ error: res.statusText }));
            throw new Error(err.error ?? `Export failed: HTTP ${res.status}`);
          }
          downloadBlob(await res.blob(), `${name}${ext}`);
        }
      } else {
        // File NOT saved — use raw export endpoint (no save required!)
        if (formatId === 'html') {
          // Generate HTML client-side from current content
          const blob = new Blob([
            `<!DOCTYPE html><html><head><meta charset="utf-8"><title>${name}</title></head><body>\n${content}\n</body></html>`
          ], { type: 'text/html' });
          downloadBlob(blob, `${name}${ext}`);
        } else {
          const margin = formatId === 'pdf' ? pdfMargin : undefined;
          const res = await fetch(api.exportRawFormat(formatId, margin), {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ content, name }),
          });
          if (!res.ok) {
            const err = await res.json().catch(() => ({ error: res.statusText }));
            throw new Error(err.error ?? `Export failed: HTTP ${res.status}`);
          }
          downloadBlob(await res.blob(), `${name}${ext}`);
        }
      }
    } catch (e: unknown) {
      exportError = e instanceof Error ? e.message : 'Export failed';
    } finally {
      exporting = null;
    }
  }

  function getExtension(formatId: string): string {
    const exts: Record<string, string> = {
      markdown: '.md', pdf: '.pdf', docx: '.docx', odt: '.odt', epub: '.epub',
      latex: '.tex', rst: '.rst', asciidoc: '.adoc', textile: '.textile',
      mediawiki: '.wiki', plain: '.txt', html: '.html',
    };
    return exts[formatId] ?? '.md';
  }

  function handleBackdropClick(e: MouseEvent): void {
    if (e.target === e.currentTarget) onClose();
  }

  function handleKeydown(e: KeyboardEvent): void {
    if (e.key === 'Escape') onClose();
  }
</script>

{#if isOpen}
  <!-- svelte-ignore a11y_no_static_element_interactions -->
  <div
    class="modal-backdrop"
    onclick={handleBackdropClick}
    onkeydown={handleKeydown}
    role="dialog"
    aria-modal="true"
    aria-label="Export dialog"
    tabindex="-1"
  >
    <div class="modal">
      <div class="modal-header">
        <div>
          <h2 class="modal-title">Export document</h2>
          <p class="modal-subtitle">
            {#if $activeFileId}
              Download "<strong>{$activeName}</strong>"
            {:else}
              Export current content <span class="badge">no save needed</span>
            {/if}
          </p>
        </div>
        <button class="btn-icon" onclick={onClose} aria-label="Close">
          <X size={18} />
        </button>
      </div>

      {#if exportError}
        <div class="export-error">
          <span>⚠</span> {exportError}
        </div>
      {/if}

      <div class="margin-selector">
        <span class="margin-label">PDF margins</span>
        <div class="margin-options">
          {#each marginOptions as opt}
            <button
              class="margin-btn"
              class:active={pdfMargin === opt.id}
              onclick={() => pdfMargin = opt.id}
            >
              <span class="margin-btn-label">{opt.label}</span>
              <span class="margin-btn-desc">{opt.desc}</span>
            </button>
          {/each}
        </div>
      </div>

      <div class="formats-grid">
        {#each formats as fmt}
          <button
            class="format-card"
            class:loading={exporting === fmt.id}
            onclick={() => handleExport(fmt.id)}
            disabled={!!exporting}
          >
            <span class="format-icon">{fmt.icon}</span>
            <div class="format-info">
              <span class="format-label">{fmt.label}</span>
              <span class="format-desc">{fmt.desc}</span>
            </div>
            {#if exporting === fmt.id}
              <Loader size={14} class="spin" />
            {:else}
              <Download size={14} class="format-dl-icon" />
            {/if}
          </button>
        {/each}
      </div>

      <div class="modal-footer">
        <button class="btn" onclick={onClose}>Close</button>
      </div>
    </div>
  </div>
{/if}

<style>
  .modal-backdrop {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.55);
    backdrop-filter: blur(6px);
    -webkit-backdrop-filter: blur(6px);
    z-index: 1000;
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 1rem;
    animation: fade-in 0.15s ease-out;
  }

  @keyframes fade-in {
    from { opacity: 0; }
    to { opacity: 1; }
  }

  .modal {
    background: var(--bg-surface);
    backdrop-filter: var(--glass-blur);
    -webkit-backdrop-filter: var(--glass-blur);
    border: 1px solid var(--border);
    border-radius: var(--radius-xl);
    box-shadow: 0 24px 64px rgba(0, 0, 0, 0.5), 0 0 0 1px rgba(255,255,255,0.04);
    width: 100%;
    max-width: 560px;
    max-height: 90vh;
    overflow-y: auto;
    display: flex;
    flex-direction: column;
    animation: modal-in 0.2s ease-out;
  }

  @keyframes modal-in {
    from { opacity: 0; transform: translateY(10px) scale(0.98); }
    to { opacity: 1; transform: translateY(0) scale(1); }
  }

  .modal-header {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    padding: 1.25rem 1.5rem 1rem;
    border-bottom: 1px solid var(--border-subtle);
  }

  .modal-title {
    font-size: 1.1rem;
    font-weight: 700;
    margin: 0 0 0.25rem;
    font-family: var(--font-ui);
    color: var(--text-primary);
  }

  .modal-subtitle {
    font-size: 13px;
    color: var(--text-secondary);
    margin: 0;
    font-family: var(--font-ui);
  }

  .badge {
    display: inline-block;
    font-size: 10px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.04em;
    background: var(--accent-surface);
    color: var(--accent);
    border: 1px solid var(--accent-light);
    border-radius: 20px;
    padding: 0.1rem 0.5rem;
    margin-left: 0.3rem;
    vertical-align: middle;
  }

  .export-error {
    margin: 0.75rem 1.5rem 0;
    padding: 0.6rem 0.9rem;
    background: var(--danger-light);
    border: 1px solid rgba(239, 68, 68, 0.2);
    border-radius: var(--radius-sm);
    color: var(--danger);
    font-size: 13px;
    font-family: var(--font-ui);
  }

  .margin-selector {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    padding: 0.6rem 1.5rem;
    border-bottom: 1px solid var(--border-subtle);
  }

  .margin-label {
    font-size: 12px;
    font-weight: 600;
    color: var(--text-secondary);
    font-family: var(--font-ui);
    white-space: nowrap;
    text-transform: uppercase;
    letter-spacing: 0.04em;
  }

  .margin-options {
    display: flex;
    gap: 0.35rem;
    flex: 1;
  }

  .margin-btn {
    flex: 1;
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 0.05rem;
    padding: 0.35rem 0.5rem;
    border: 1px solid var(--border-subtle);
    border-radius: var(--radius-sm);
    background: transparent;
    cursor: pointer;
    transition: all 0.15s;
    font-family: var(--font-ui);
    color: var(--text-secondary);
  }
  .margin-btn:hover { background: var(--bg-hover); border-color: var(--border); }
  .margin-btn.active {
    background: var(--accent-surface);
    border-color: var(--accent);
    color: var(--accent);
  }
  .margin-btn-label { font-size: 12px; font-weight: 600; }
  .margin-btn-desc { font-size: 10px; opacity: 0.7; }

  .formats-grid {
    display: flex;
    flex-direction: column;
    padding: 0.75rem 1rem;
    gap: 0.3rem;
  }

  .format-card {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    padding: 0.65rem 0.9rem;
    border: 1px solid var(--border-subtle);
    border-radius: var(--radius);
    background: transparent;
    cursor: pointer;
    transition: all 0.15s;
    text-align: left;
    width: 100%;
    font-family: var(--font-ui);
    color: inherit;
  }
  .format-card:hover:not(:disabled) {
    background: var(--bg-hover);
    border-color: var(--accent);
    box-shadow: 0 0 0 1px var(--accent-light);
  }
  .format-card:disabled { opacity: 0.5; cursor: wait; }
  .format-card.loading {
    background: var(--accent-surface);
    border-color: var(--accent);
  }

  .format-icon { font-size: 1.25rem; flex-shrink: 0; }

  .format-info {
    flex: 1;
    display: flex;
    flex-direction: column;
    gap: 0.1rem;
  }

  .format-label {
    font-size: 13px;
    font-weight: 600;
    color: var(--text-primary);
  }

  .format-desc {
    font-size: 11px;
    color: var(--text-muted);
  }

  :global(.format-dl-icon) { color: var(--text-muted); flex-shrink: 0; transition: color 0.15s; }
  .format-card:hover :global(.format-dl-icon) { color: var(--accent); }

  :global(.spin) {
    animation: spin 0.8s linear infinite;
    color: var(--accent);
    flex-shrink: 0;
  }
  @keyframes spin { to { transform: rotate(360deg); } }

  .modal-footer {
    padding: 0.75rem 1.5rem 1.25rem;
    border-top: 1px solid var(--border-subtle);
    display: flex;
    align-items: center;
    justify-content: flex-end;
  }
</style>
