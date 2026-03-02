<script lang="ts">
  import { onMount } from 'svelte';
  import Sidebar from '$lib/components/Sidebar.svelte';
  import Toolbar from '$lib/components/Toolbar.svelte';
  import Editor from '$lib/components/Editor.svelte';
  import Preview from '$lib/components/Preview.svelte';
  import ExportModal from '$lib/components/ExportModal.svelte';
  import TemplateModal from '$lib/components/TemplateModal.svelte';
  import SearchModal from '$lib/components/SearchModal.svelte';
  import VersionHistory from '$lib/components/VersionHistory.svelte';
  import Particles from '$lib/components/Particles.svelte';
  import { loadFiles, initTheme, viewMode, error, theme, sidebarOpen } from '$lib/stores/files';
  import { AlertCircle, X } from 'lucide-svelte';

  let exportOpen = $state(false);
  let templatesOpen = $state(false);
  let searchOpen = $state(false);
  let historyOpen = $state(false);

  onMount(async () => {
    initTheme();
    await loadFiles();
  });

  function dismissError(): void {
    error.set(null);
  }
</script>

<div class="app-shell" data-theme={$theme}>
  <!-- Background effects layer -->
  <div class="bg-effects" aria-hidden="true">
    <div class="orb orb-1"></div>
    <div class="orb orb-2"></div>
    <div class="orb orb-3"></div>
  </div>
  <Particles />

  <!-- App layout -->
  <div class="app-layout">
    <!-- Sidebar (collapsible) -->
    {#if $sidebarOpen}
      <Sidebar />
    {/if}

    <!-- Main area -->
    <div class="main-area">
      <!-- Toolbar -->
      <Toolbar
        onExport={() => (exportOpen = true)}
        onTemplates={() => (templatesOpen = true)}
        onSearch={() => (searchOpen = true)}
        onHistory={() => (historyOpen = true)}
      />

      <!-- Error banner -->
      {#if $error}
        <div class="error-banner fade-in">
          <AlertCircle size={15} />
          <span>{$error}</span>
          <button class="btn-icon" onclick={dismissError}>
            <X size={14} />
          </button>
        </div>
      {/if}

      <!-- Editor / Preview workspace -->
      <div class="workspace">
        {#if $viewMode === 'split'}
          <div class="pane editor-pane">
            <Editor />
          </div>
          <div class="pane-divider"></div>
          <div class="pane preview-pane">
            <Preview />
          </div>
        {:else if $viewMode === 'editor'}
          <div class="pane editor-pane full">
            <Editor />
          </div>
        {:else}
          <div class="pane preview-pane full">
            <Preview />
          </div>
        {/if}
      </div>
    </div>
  </div>

  <!-- Footer -->
  <footer class="app-footer no-print">
    <div class="footer-left">
      <svg width="14" height="10" viewBox="0 0 3 2" aria-label="France">
        <rect width="1" height="2" fill="#002395"/>
        <rect x="1" width="1" height="2" fill="#fff"/>
        <rect x="2" width="1" height="2" fill="#ED2939"/>
      </svg>
      <span>MD, une solution <a href="https://cybergraphe.fr" target="_blank" rel="noopener noreferrer">Cybergraphe</a></span>
    </div>
    <div class="footer-right">
      <a href="https://ko-fi.com/cybergraphe" target="_blank" rel="noopener noreferrer" class="kofi-link" title="Support MD on Ko-fi">
        <svg width="13" height="13" viewBox="0 0 24 24" fill="currentColor"><path d="M23.05 7.04c-.06-.67-.46-1.47-1.18-1.9a3.57 3.57 0 0 0-1.87-.58h-1.26V4.5c0-.83-.67-1.5-1.5-1.5H4.24c-.83 0-1.5.67-1.5 1.5v8.76c0 3.05 2.48 5.53 5.53 5.53h5.41c3.05 0 5.53-2.48 5.53-5.53v-.62h.79c1.77 0 3.24-1.17 3.24-3.3 0-.82-.05-1.62-.19-2.3ZM17.21 13.26c0 1.94-1.59 3.53-3.53 3.53H8.27c-1.94 0-3.53-1.59-3.53-3.53V5h12.47v8.26Zm3.32-3.67c0 .83-.5 1.3-1.24 1.3h-.28V6.56h.28c.53 0 .93.16 1.1.43.2.3.28.88.14 2.6Z"/><path d="M12.34 7.54c-.4-.23-.87-.33-1.35-.33-.48 0-.95.1-1.35.33-.82.47-1.33 1.35-1.33 2.3 0 1.79 1.9 3.94 2.68 4.1.78-.16 2.68-2.31 2.68-4.1 0-.95-.51-1.83-1.33-2.3Z"/></svg>
        Support
      </a>
    </div>
  </footer>

  <!-- Export modal -->
  <ExportModal isOpen={exportOpen} onClose={() => (exportOpen = false)} />

  <!-- Template picker -->
  <TemplateModal isOpen={templatesOpen} onClose={() => (templatesOpen = false)} />

  <!-- Search modal -->
  <SearchModal isOpen={searchOpen} onClose={() => (searchOpen = false)} />

  <!-- Version history -->
  <VersionHistory isOpen={historyOpen} onClose={() => (historyOpen = false)} />
</div>

<style>
  .app-shell {
    position: relative;
    height: 100vh;
    overflow: hidden;
    background: var(--bg-app);
    display: flex;
    flex-direction: column;
  }

  .app-layout {
    position: relative;
    z-index: 1;
    display: flex;
    flex: 1;
    min-height: 0;
  }

  .main-area {
    flex: 1;
    display: flex;
    flex-direction: column;
    min-width: 0;
    height: 100%;
    overflow: hidden;
  }

  .error-banner {
    display: flex;
    align-items: center;
    gap: 0.6rem;
    padding: 0.5rem 1rem;
    background: var(--danger-light);
    border-bottom: 1px solid rgba(239, 68, 68, 0.2);
    color: var(--danger);
    font-size: 13px;
    font-family: var(--font-ui);
    backdrop-filter: blur(8px);
  }
  .error-banner span { flex: 1; }

  .workspace {
    flex: 1;
    display: flex;
    overflow: hidden;
    height: 0;
  }

  .pane {
    display: flex;
    flex-direction: column;
    flex: 1;
    min-width: 0;
    height: 100%;
    overflow: hidden;
  }

  .pane.full { flex: 1; }

  .editor-pane {
    flex: 1;
    background: var(--bg-editor);
    backdrop-filter: var(--glass-blur);
    -webkit-backdrop-filter: var(--glass-blur);
  }

  .preview-pane {
    flex: 1;
    background: var(--bg-preview);
    backdrop-filter: var(--glass-blur);
    -webkit-backdrop-filter: var(--glass-blur);
    overflow-y: auto;
  }

  .pane-divider {
    width: 1px;
    background: var(--border);
    flex-shrink: 0;
    position: relative;
  }
  .pane-divider::after {
    content: '';
    position: absolute;
    inset: 0 -3px;
    cursor: col-resize;
  }

  .app-footer {
    position: relative;
    z-index: 1;
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 0.2rem 1rem;
    font-family: var(--font-ui);
    font-size: 10.5px;
    color: var(--text-muted);
    border-top: 1px solid var(--border-subtle);
    background: var(--bg-toolbar);
    backdrop-filter: var(--glass-blur);
    -webkit-backdrop-filter: var(--glass-blur);
    flex-shrink: 0;
  }

  .footer-left {
    display: flex;
    align-items: center;
    gap: 0.4rem;
  }
  .footer-left a {
    color: var(--text-secondary);
    text-decoration: none;
    transition: color 0.15s;
  }
  .footer-left a:hover { color: var(--accent); }

  .footer-right {
    display: flex;
    align-items: center;
  }

  .kofi-link {
    display: inline-flex;
    align-items: center;
    gap: 0.3rem;
    color: var(--text-muted);
    text-decoration: none;
    font-size: 10.5px;
    padding: 0.15rem 0.5rem;
    border-radius: var(--radius-sm);
    transition: all 0.15s;
  }
  .kofi-link:hover {
    color: #ff5e5b;
    background: rgba(255, 94, 91, 0.08);
  }
</style>
