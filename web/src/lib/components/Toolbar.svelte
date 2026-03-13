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
    triggerFormatAction,
    type FormatActionKind,
  } from '$lib/stores/files';
  import FontPicker from './FontPicker.svelte';

  let showShortcuts = $state(false);
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
    Bold,
    Italic,
    Underline,
    Strikethrough,
    Heading1,
    Heading2,
    Heading3,
    Pilcrow,
    List,
    ListOrdered,
    ListChecks,
    Code,
    SquareCode,
    Quote,
    Link2,
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

  const formatButtons: Array<{
    action: FormatActionKind;
    label: string;
    icon: typeof Bold;
  }> = [
    { action: 'bold', label: 'Bold', icon: Bold },
    { action: 'italic', label: 'Italic', icon: Italic },
    { action: 'underline', label: 'Underline', icon: Underline },
    { action: 'strike', label: 'Strike', icon: Strikethrough },
    { action: 'heading1', label: 'H1', icon: Heading1 },
    { action: 'heading2', label: 'H2', icon: Heading2 },
    { action: 'heading3', label: 'H3', icon: Heading3 },
    { action: 'paragraph', label: 'Paragraph', icon: Pilcrow },
    { action: 'unorderedList', label: 'Bullet list', icon: List },
    { action: 'orderedList', label: 'Numbered list', icon: ListOrdered },
    { action: 'taskList', label: 'Task list', icon: ListChecks },
    { action: 'quote', label: 'Quote', icon: Quote },
    { action: 'codeInline', label: 'Inline code', icon: Code },
    { action: 'codeBlock', label: 'Code block', icon: SquareCode },
    { action: 'link', label: 'Link', icon: Link2 },
  ];

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
      // Don't intercept when CodeMirror editor is focused (it handles Ctrl+K for link insertion)
      const active = document.activeElement;
      if (active && active.closest('.cm-editor')) return;
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
      value={$activeName}
      oninput={(e) => { activeName.set((e.target as HTMLInputElement).value); isDirty.set(true); }}
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

    <div class="relative">
      <button
        class="btn btn-icon"
        title="Keyboard shortcuts"
        onclick={() => showShortcuts = !showShortcuts}
        onblur={() => showShortcuts = false}
      >
        <Keyboard size={15} />
      </button>
      {#if showShortcuts}
        <div class="shortcuts-popover">
          <div><kbd>Ctrl+S</kbd> Save</div>
          <div><kbd>Ctrl+B</kbd> Bold</div>
          <div><kbd>Ctrl+I</kbd> Italic</div>
          <div><kbd>Ctrl+K</kbd> Link</div>
          <div><kbd>Ctrl+Shift+P</kbd> Toggle preview</div>
        </div>
      {/if}
    </div>
  </div>
</header>

<div class="quick-format no-print" aria-label="Quick formatting toolbar" role="toolbar">
  {#each formatButtons as button (button.action)}
    {@const FormatIcon = button.icon}
    <button
      class="btn btn-format"
      title={button.label}
      aria-label={button.label}
      onclick={() => triggerFormatAction(button.action)}
    >
      <FormatIcon size={14} />
      <span>{button.label}</span>
    </button>
  {/each}
</div>

<style>
  .toolbar {
    display: flex;
    align-items: center;
    gap: 0.45rem;
    flex-wrap: wrap;
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
    min-width: 240px;
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
    min-width: 250px;
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

  .quick-format {
    display: flex;
    align-items: center;
    gap: 0.3rem;
    padding: 0.4rem 1rem;
    border-bottom: 1px solid var(--border-subtle);
    background: color-mix(in srgb, var(--bg-toolbar) 92%, transparent);
    overflow-x: auto;
    scrollbar-width: thin;
  }

  .btn-format {
    min-height: 32px;
    padding: 0.35rem 0.58rem;
    gap: 0.3rem;
    font-size: 12px;
    flex-shrink: 0;
  }

  .btn-format span {
    line-height: 1;
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

  .shortcuts-popover {
    position: absolute;
    top: calc(100% + 6px);
    right: 0;
    z-index: 100;
    background: var(--bg-surface);
    border: 1px solid var(--border);
    border-radius: var(--radius);
    box-shadow: 0 8px 24px rgba(0, 0, 0, 0.4);
    padding: 0.5rem 0.75rem;
    font-family: var(--font-ui);
    font-size: 12px;
    color: var(--text-primary);
    white-space: nowrap;
    display: flex;
    flex-direction: column;
    gap: 0.3rem;
  }
  .shortcuts-popover kbd {
    display: inline-block;
    background: var(--bg-app);
    border: 1px solid var(--border-subtle);
    border-radius: 3px;
    padding: 0.05rem 0.35rem;
    font-size: 10px;
    font-family: var(--font-mono, monospace);
    margin-right: 0.4rem;
  }

  @media (max-width: 1024px) {
    .toolbar {
      padding: 0.45rem 0.75rem;
      gap: 0.4rem;
    }

    .toolbar-left {
      order: 1;
      min-width: 100%;
    }

    .toolbar-center {
      order: 2;
      margin-right: auto;
    }

    .toolbar-right {
      order: 3;
      min-width: 0;
      flex: 1 1 auto;
      overflow-x: auto;
      justify-content: flex-start;
      padding-bottom: 1px;
    }

    .doc-title {
      max-width: none;
    }
  }

  @media (max-width: 720px) {
    .toolbar {
      padding: 0.4rem 0.6rem;
    }

    .toolbar-center {
      width: 100%;
      justify-content: center;
      order: 3;
    }

    .toolbar-right {
      width: 100%;
      order: 2;
      gap: 0.25rem;
    }

    .view-btn .view-label {
      display: none;
    }

    .btn {
      min-height: 32px;
      padding: 0.35rem 0.55rem;
    }

    .divider-v {
      height: 16px;
      margin: 0 0.1rem;
    }

    .quick-format {
      padding: 0.35rem 0.6rem;
    }

    .btn-format {
      min-height: 30px;
      padding: 0.3rem 0.48rem;
      font-size: 11.5px;
    }
  }
</style>
