<script lang="ts">
  import {
    activeName,
    activeFileId,
    isDirty,
    isSaving,
    viewMode,
    saveActiveFile,
    toggleTheme,
    toggleSidebar,
    sidebarOpen,
    theme,
    createFile,
  } from '$lib/stores/files';
  import FontPicker from './FontPicker.svelte';
  import {
    Save,
    FilePlus,
    Download,
    Printer,
    Sun,
    Moon,
    Columns2,
    PanelLeft,
    PanelLeftClose,
    Eye,
    Keyboard,
    LayoutTemplate,
    Search,
    History,
  } from 'lucide-svelte';

  let {
    onExport,
    onTemplates,
    onSearch,
    onHistory,
  }: {
    onExport: () => void;
    onTemplates: () => void;
    onSearch: () => void;
    onHistory: () => void;
  } = $props();

  const viewIcons: Record<string, typeof Columns2> = {
    split: Columns2,
    editor: PanelLeft,
    preview: Eye,
  };

  const viewLabels = { split: 'Split', editor: 'Editor', preview: 'Preview' };

  function handlePrint(): void {
    window.print();
  }

  async function handleSave(): Promise<void> {
    await saveActiveFile();
  }

  function handleKeyboardShortcuts(e: KeyboardEvent): void {
    if ((e.metaKey || e.ctrlKey) && e.key === 's') {
      e.preventDefault();
      handleSave();
    }
    if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
      e.preventDefault();
      onSearch();
    }
  }
</script>

<svelte:window onkeydown={handleKeyboardShortcuts} />

<header class="toolbar no-print">
  <!-- Left: sidebar toggle + title -->
  <div class="toolbar-left">
    <button
      class="btn btn-icon sidebar-toggle"
      title={$sidebarOpen ? 'Hide files panel' : 'Show files panel'}
      onclick={toggleSidebar}
    >
      {#if $sidebarOpen}
        <PanelLeftClose size={16} />
      {:else}
        <PanelLeft size={16} />
      {/if}
    </button>
    <input
      class="doc-title"
      type="text"
      bind:value={$activeName}
      placeholder="Untitled document"
      spellcheck="false"
    />
    {#if $isDirty}
      <span class="dirty-indicator" title="Unsaved changes">●</span>
    {/if}
  </div>

  <!-- Center: view mode -->
  <div class="toolbar-center">
    {#each Object.entries(viewLabels) as [mode, label]}
      {@const IconComp = viewIcons[mode]}
      <button
        class="btn btn-icon view-btn"
        class:active={$viewMode === mode}
        title={label}
        onclick={() => viewMode.set(mode as 'split' | 'editor' | 'preview')}
      >
        <IconComp size={15} />
        <span class="view-label">{label}</span>
      </button>
    {/each}
  </div>

  <!-- Right: actions -->
  <div class="toolbar-right">
    <button
      class="btn"
      class:btn-primary={$isDirty}
      disabled={$isSaving}
      title="Save (Ctrl+S)"
      onclick={handleSave}
    >
      <Save size={14} />
      {$isSaving ? 'Saving…' : 'Save'}
    </button>

    <div class="divider-v"></div>

    <button class="btn" title="New file" onclick={() => createFile()}>
      <FilePlus size={14} />
    </button>

    <button class="btn" title="From template" onclick={onTemplates}>
      <LayoutTemplate size={14} />
    </button>

    <button class="btn" title="Search (Ctrl+K)" onclick={onSearch}>
      <Search size={14} />
    </button>

    {#if $activeFileId}
      <button class="btn" title="Version history" onclick={onHistory}>
        <History size={14} />
      </button>
    {/if}

    <button class="btn" title="Export" onclick={onExport}>
      <Download size={14} />
      Export
    </button>

    <button class="btn" title="Print" onclick={handlePrint}>
      <Printer size={14} />
    </button>

    <div class="divider-v"></div>

    <FontPicker />

    <button class="btn btn-icon" title="Toggle theme" onclick={toggleTheme}>
      {#if $theme === 'dark'}
        <Sun size={15} />
      {:else}
        <Moon size={15} />
      {/if}
    </button>

    <button
      class="btn btn-icon"
      title="Keyboard shortcuts"
      onclick={() => alert('Ctrl+S: Save\nCtrl+B: Bold\nCtrl+I: Italic\nCtrl+K: Link\nCtrl+Shift+P: Toggle preview')}
    >
      <Keyboard size={15} />
    </button>
  </div>
</header>

<style>
  .toolbar {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    padding: 0.45rem 1rem;
    background: var(--bg-toolbar);
    backdrop-filter: var(--glass-blur);
    -webkit-backdrop-filter: var(--glass-blur);
    border-bottom: 1px solid var(--border);
    z-index: 10;
  }

  .toolbar-left {
    display: flex;
    align-items: center;
    gap: 0.4rem;
    flex: 1;
    min-width: 0;
  }

  .toolbar-center {
    display: flex;
    gap: 0.15rem;
    background: var(--bg-surface);
    border: 1px solid var(--border-subtle);
    border-radius: var(--radius);
    padding: 2px;
  }

  .toolbar-right {
    display: flex;
    align-items: center;
    gap: 0.3rem;
    flex: 1;
    justify-content: flex-end;
  }

  .doc-title {
    font-family: var(--font-ui);
    font-size: 14px;
    font-weight: 600;
    color: var(--text-primary);
    background: transparent;
    border: 1px solid transparent;
    border-radius: var(--radius-sm);
    padding: 0.3rem 0.5rem;
    outline: none;
    min-width: 0;
    max-width: 300px;
    flex: 1;
    transition: all var(--transition);
  }
  .doc-title:focus {
    border-color: var(--accent);
    background: var(--bg-surface);
    box-shadow: 0 0 0 2px var(--accent-light);
  }
  .doc-title::placeholder { color: var(--text-muted); font-weight: 400; }

  .dirty-indicator {
    color: var(--accent);
    font-size: 14px;
    line-height: 1;
    animation: pulse-glow 2s ease-in-out infinite;
  }

  @keyframes pulse-glow {
    0%, 100% { opacity: 0.6; }
    50% { opacity: 1; text-shadow: 0 0 6px var(--accent-glow); }
  }

  .view-btn {
    display: flex;
    align-items: center;
    gap: 0.3rem;
    border-radius: calc(var(--radius) - 3px);
    padding: 0.28rem 0.55rem;
    color: var(--text-secondary);
    background: transparent;
    border: none;
    transition: all 0.15s;
  }
  .view-btn.active {
    background: var(--accent-surface);
    color: var(--accent);
  }
  .view-btn:hover:not(.active) { background: var(--bg-hover); }

  .view-label {
    font-size: 12px;
    font-weight: 500;
  }

  .divider-v {
    width: 1px;
    height: 20px;
    background: var(--border-subtle);
    margin: 0 0.2rem;
  }

  .sidebar-toggle {
    flex-shrink: 0;
    color: var(--text-secondary);
    transition: color 0.15s;
  }
  .sidebar-toggle:hover { color: var(--accent); }
</style>
