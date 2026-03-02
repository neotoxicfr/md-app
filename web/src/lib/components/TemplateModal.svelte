<script lang="ts">
  import { api, type TemplateSummary } from '$lib/api';
  import { createFile } from '$lib/stores/files';
  import { X, FileText, BookOpen, Code, Briefcase, PenTool } from 'lucide-svelte';

  let { isOpen = false, onClose }: { isOpen: boolean; onClose: () => void } = $props();

  let templates = $state<TemplateSummary[]>([]);
  let loading = $state(false);
  let applying = $state(false);

  const categoryIcons: Record<string, typeof FileText> = {
    writing: PenTool,
    business: Briefcase,
    engineering: Code,
    project: BookOpen,
  };

  $effect(() => {
    if (isOpen && templates.length === 0) {
      loadTemplates();
    }
  });

  async function loadTemplates() {
    loading = true;
    try {
      const res = await api.listTemplates();
      templates = res.templates;
    } catch {
      templates = [];
    } finally {
      loading = false;
    }
  }

  async function applyTemplate(id: string) {
    applying = true;
    try {
      const detail = await api.getTemplate(id);
      await createFile(detail.name, detail.content);
      onClose();
    } catch {
      // error handled by store
    } finally {
      applying = false;
    }
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
        <h2>Templates</h2>
        <button class="btn-icon" onclick={onClose}><X size={18} /></button>
      </div>

      <p class="modal-subtitle">Start with a pre-built template to save time</p>

      <div class="template-grid">
        {#if loading}
          <div class="template-loading">Loading templates…</div>
        {:else}
          {#each templates as t (t.id)}
            {@const IconComp = categoryIcons[t.category] ?? FileText}
            <button
              class="template-card"
              disabled={applying}
              onclick={() => applyTemplate(t.id)}
            >
              <div class="template-icon">
                <IconComp size={20} />
              </div>
              <div class="template-info">
                <span class="template-name">{t.name}</span>
                <span class="template-desc">{t.description}</span>
              </div>
              <span class="template-category">{t.category}</span>
            </button>
          {/each}
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
    -webkit-backdrop-filter: blur(4px);
    animation: fadeIn 0.15s ease;
  }

  .modal-content {
    background: var(--bg-surface);
    border: 1px solid var(--border);
    border-radius: var(--radius-lg, 12px);
    padding: 1.5rem;
    width: 600px;
    max-width: 92vw;
    max-height: 80vh;
    overflow-y: auto;
    box-shadow: 0 20px 60px rgba(0, 0, 0, 0.4);
  }

  .modal-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 0.25rem;
  }

  .modal-header h2 {
    font-family: var(--font-ui);
    font-size: 18px;
    font-weight: 700;
    color: var(--text-primary);
  }

  .modal-subtitle {
    font-family: var(--font-ui);
    font-size: 13px;
    color: var(--text-muted);
    margin-bottom: 1.25rem;
  }

  .template-grid {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
  }

  .template-card {
    display: flex;
    align-items: center;
    gap: 0.85rem;
    padding: 0.75rem 1rem;
    background: var(--bg-app);
    border: 1px solid var(--border-subtle);
    border-radius: var(--radius);
    cursor: pointer;
    text-align: left;
    transition: all 0.15s;
    font-family: var(--font-ui);
  }

  .template-card:hover {
    border-color: var(--accent);
    background: var(--accent-surface);
    transform: translateY(-1px);
  }

  .template-card:disabled {
    opacity: 0.5;
    pointer-events: none;
  }

  .template-icon {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 36px;
    height: 36px;
    border-radius: var(--radius-sm);
    background: var(--accent-surface);
    color: var(--accent);
    flex-shrink: 0;
  }

  .template-info {
    flex: 1;
    display: flex;
    flex-direction: column;
    min-width: 0;
  }

  .template-name {
    font-weight: 600;
    font-size: 13.5px;
    color: var(--text-primary);
  }

  .template-desc {
    font-size: 12px;
    color: var(--text-muted);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .template-category {
    font-size: 10px;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    color: var(--text-muted);
    background: var(--bg-surface);
    padding: 0.15rem 0.5rem;
    border-radius: 99px;
    white-space: nowrap;
  }

  .template-loading {
    text-align: center;
    color: var(--text-muted);
    font-size: 13px;
    padding: 2rem 0;
  }

  @keyframes fadeIn {
    from { opacity: 0; }
    to { opacity: 1; }
  }
</style>
