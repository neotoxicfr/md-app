<script lang="ts">
  import { fontConfig, fontOptions, setFontFor } from '$lib/stores/files';
  import { Type } from 'lucide-svelte';

  let open = $state(false);
  let pickerEl = $state<HTMLElement | undefined>(undefined);

  function toggle(): void {
    open = !open;
  }

  function handleWindowClick(e: MouseEvent): void {
    if (open && pickerEl && !pickerEl.contains(e.target as Node)) {
      open = false;
    }
  }
</script>

<svelte:window onclick={handleWindowClick} />

<div class="font-picker" bind:this={pickerEl}>
  <button
    class="btn btn-icon font-trigger"
    title="Polices"
    onclick={toggle}
  >
    <Type size={15} />
  </button>

  {#if open}
    <div class="font-dropdown">
      <!-- Headings font -->
      <div class="font-section-header">Titres</div>
      <div class="font-section-sub">H1, H2, H3, H4, H5, H6</div>
      <div class="font-category">Serif</div>
      {#each fontOptions.filter(f => f.category === 'serif') as font}
        <button
          class="font-option"
          class:active={$fontConfig.headings === font.name}
          style="font-family: {font.family}"
          onclick={() => setFontFor('headings', font.name)}
        >
          {font.name}
        </button>
      {/each}
      <div class="font-category">Sans-serif</div>
      {#each fontOptions.filter(f => f.category === 'sans-serif') as font}
        <button
          class="font-option"
          class:active={$fontConfig.headings === font.name}
          style="font-family: {font.family}"
          onclick={() => setFontFor('headings', font.name)}
        >
          {font.name}
        </button>
      {/each}

      <!-- Body font -->
      <div class="font-separator"></div>
      <div class="font-section-header">Corps de texte</div>
      <div class="font-section-sub">Paragraphes, listes, citations</div>
      <div class="font-category">Serif</div>
      {#each fontOptions.filter(f => f.category === 'serif') as font}
        <button
          class="font-option"
          class:active={$fontConfig.body === font.name}
          style="font-family: {font.family}"
          onclick={() => setFontFor('body', font.name)}
        >
          {font.name}
        </button>
      {/each}
      <div class="font-category">Sans-serif</div>
      {#each fontOptions.filter(f => f.category === 'sans-serif') as font}
        <button
          class="font-option"
          class:active={$fontConfig.body === font.name}
          style="font-family: {font.family}"
          onclick={() => setFontFor('body', font.name)}
        >
          {font.name}
        </button>
      {/each}
    </div>
  {/if}
</div>

<style>
  .font-picker {
    position: relative;
  }

  .font-trigger {
    display: flex;
    align-items: center;
    gap: 0.3rem;
    color: var(--text-secondary);
  }

  .font-dropdown {
    position: absolute;
    top: calc(100% + 6px);
    right: 0;
    z-index: 100;
    width: 240px;
    max-height: 460px;
    overflow-y: auto;
    background: #18181b;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    box-shadow: 0 12px 40px rgba(0, 0, 0, 0.7);
    animation: dropdown-in 0.15s ease-out;
  }

  @keyframes dropdown-in {
    from { opacity: 0; transform: translateY(-4px); }
    to { opacity: 1; transform: none; }
  }

  .font-section-header {
    padding: 0.6rem 0.75rem 0;
    font-size: 12px;
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.04em;
    color: var(--text-primary);
    font-family: var(--font-ui);
  }

  .font-section-sub {
    padding: 0 0.75rem 0.25rem;
    font-size: 10px;
    color: var(--text-muted);
    font-family: var(--font-ui);
  }

  .font-separator {
    height: 1px;
    background: var(--border);
    margin: 0.5rem 0.75rem;
  }

  .font-category {
    padding: 0.35rem 0.75rem 0.15rem;
    font-size: 10px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.06em;
    color: var(--accent);
    font-family: var(--font-ui);
  }

  .font-option {
    display: block;
    width: 100%;
    text-align: left;
    padding: 0.35rem 0.75rem;
    font-size: 15px;
    border: none;
    background: transparent;
    color: var(--text-primary);
    cursor: pointer;
    transition: background 0.1s;
  }
  .font-option:hover {
    background: rgba(255, 255, 255, 0.1);
  }
  .font-option.active {
    background: rgba(139, 92, 246, 0.22);
    color: #c4b5fd;
    font-weight: 600;
    border-left: 3px solid var(--accent);
    padding-left: calc(0.75rem - 3px);
  }
</style>
