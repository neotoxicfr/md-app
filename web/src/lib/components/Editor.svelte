<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { EditorView, keymap, lineNumbers, drawSelection, dropCursor } from '@codemirror/view';
  import { EditorState, type Extension } from '@codemirror/state';
  import { defaultKeymap, history, historyKeymap, indentWithTab } from '@codemirror/commands';
  import { searchKeymap, highlightSelectionMatches } from '@codemirror/search';
  import { markdown, markdownLanguage } from '@codemirror/lang-markdown';
  import { languages } from '@codemirror/language-data';
  import { syntaxHighlighting, defaultHighlightStyle, indentOnInput, foldGutter } from '@codemirror/language';
  import { autocompletion, completionKeymap } from '@codemirror/autocomplete';
  import { activeContent, setContent, theme } from '$lib/stores/files';

  let container: HTMLDivElement;
  let view: EditorView | undefined;

  function buildTheme(dark: boolean): Extension {
    return EditorView.theme(
      {
        '&': {
          backgroundColor: 'transparent',
          color: dark ? '#e4e4e7' : '#18181b',
          height: '100%',
        },
        '.cm-content': {
          caretColor: dark ? '#8b5cf6' : '#7c3aed',
          fontFamily: 'var(--font-mono)',
          fontSize: '14px',
          lineHeight: '1.7',
          padding: '0.75rem 0',
        },
        '.cm-cursor': { borderLeftColor: dark ? '#8b5cf6' : '#7c3aed', borderLeftWidth: '2px' },
        '.cm-gutters': {
          backgroundColor: 'transparent',
          color: dark ? '#3f3f46' : '#a1a1aa',
          borderRight: `1px solid ${dark ? 'rgba(255,255,255,0.04)' : 'rgba(0,0,0,0.06)'}`,
          minWidth: '3.2rem',
        },
        '.cm-gutter': { fontSize: '12px' },
        '.cm-activeLine': {
          backgroundColor: dark ? 'rgba(255,255,255,0.03)' : 'rgba(0,0,0,0.02)',
        },
        '.cm-activeLineGutter': {
          backgroundColor: dark ? 'rgba(255,255,255,0.03)' : 'rgba(0,0,0,0.02)',
          color: dark ? '#71717a' : '#52525b',
        },
        '.cm-selectionBackground, ::selection': {
          backgroundColor: dark ? 'rgba(139,92,246,0.2) !important' : 'rgba(124,58,237,0.12) !important',
        },
        '.cm-matchingBracket': {
          color: dark ? '#c4b5fd' : '#7c3aed',
          fontWeight: '600',
          backgroundColor: dark ? 'rgba(139,92,246,0.15)' : 'rgba(124,58,237,0.1)',
          borderRadius: '2px',
        },
        '.cm-foldGutter': { color: dark ? '#3f3f46' : '#d4d4d8' },
        '.cm-tooltip': {
          backgroundColor: dark ? '#18181b' : '#ffffff',
          border: `1px solid ${dark ? 'rgba(255,255,255,0.08)' : 'rgba(0,0,0,0.08)'}`,
          borderRadius: '8px',
          boxShadow: dark ? '0 8px 32px rgba(0,0,0,0.5)' : '0 8px 32px rgba(0,0,0,0.12)',
        },
        '.cm-tooltip-autocomplete ul li[aria-selected]': {
          backgroundColor: dark ? 'rgba(139,92,246,0.15)' : 'rgba(124,58,237,0.08)',
        },
        '.cm-panels': {
          backgroundColor: dark ? '#18181b' : '#fafafa',
          borderBottom: `1px solid ${dark ? 'rgba(255,255,255,0.06)' : 'rgba(0,0,0,0.06)'}`,
        },
        '.cm-search': {
          fontSize: '13px',
        },
        '.cm-button': {
          backgroundImage: 'none',
          backgroundColor: dark ? 'rgba(255,255,255,0.06)' : 'rgba(0,0,0,0.04)',
          border: `1px solid ${dark ? 'rgba(255,255,255,0.1)' : 'rgba(0,0,0,0.1)'}`,
          borderRadius: '4px',
          color: dark ? '#e4e4e7' : '#18181b',
        },
        '.cm-textfield': {
          backgroundColor: dark ? 'rgba(255,255,255,0.05)' : '#ffffff',
          border: `1px solid ${dark ? 'rgba(255,255,255,0.1)' : 'rgba(0,0,0,0.1)'}`,
          borderRadius: '4px',
          color: dark ? '#e4e4e7' : '#18181b',
        },
      },
      { dark }
    );
  }

  function createExtensions(dark: boolean): Extension[] {
    return [
      lineNumbers(),
      foldGutter(),
      drawSelection(),
      dropCursor(),
      history(),
      indentOnInput(),
      syntaxHighlighting(defaultHighlightStyle, { fallback: true }),
      markdown({
        base: markdownLanguage,
        codeLanguages: languages,
        addKeymap: true,
      }),
      highlightSelectionMatches(),
      autocompletion(),
      buildTheme(dark),
      keymap.of([
        indentWithTab,
        ...defaultKeymap,
        ...historyKeymap,
        ...searchKeymap,
        ...completionKeymap,
      ]),
      keymap.of([
        {
          key: 'Ctrl-b',
          run: (v) => wrapSelection(v, '**'),
        },
        {
          key: 'Ctrl-i',
          run: (v) => wrapSelection(v, '_'),
        },
        {
          key: 'Ctrl-k',
          run: (v) => {
            const sel = v.state.sliceDoc(
              v.state.selection.main.from,
              v.state.selection.main.to
            );
            v.dispatch({
              changes: {
                from: v.state.selection.main.from,
                to: v.state.selection.main.to,
                insert: `[${sel}](url)`,
              },
            });
            return true;
          },
        },
      ]),
      EditorView.updateListener.of((update) => {
        if (update.docChanged) {
          setContent(update.state.doc.toString());
        }
      }),
      EditorView.lineWrapping,
    ];
  }

  function wrapSelection(v: EditorView, wrapper: string): boolean {
    const { from, to } = v.state.selection.main;
    const sel = v.state.sliceDoc(from, to);
    v.dispatch({
      changes: { from, to, insert: `${wrapper}${sel}${wrapper}` },
      selection: { anchor: from + wrapper.length, head: to + wrapper.length },
    });
    return true;
  }

  let currentTheme: 'light' | 'dark' = 'light';
  const unsubTheme = theme.subscribe((t) => {
    currentTheme = t;
    if (view) recreateExtensions(t === 'dark');
  });

  let lastExternalContent = '';
  const unsubContent = activeContent.subscribe((c) => {
    if (!view) return;
    const current = view.state.doc.toString();
    if (c !== current && c !== lastExternalContent) {
      lastExternalContent = c;
      view.dispatch({
        changes: { from: 0, to: view.state.doc.length, insert: c },
      });
    }
  });

  function recreateExtensions(dark: boolean): void {
    if (!view) return;
    view.dispatch({
      effects: EditorView.scrollIntoView(0),
    });
    const doc = view.state.doc.toString();
    const state = EditorState.create({
      doc,
      extensions: createExtensions(dark),
    });
    view.setState(state);
  }

  onMount(() => {
    const state = EditorState.create({
      doc: $activeContent,
      extensions: createExtensions(currentTheme === 'dark'),
    });
    view = new EditorView({ state, parent: container });
    lastExternalContent = $activeContent;
  });

  onDestroy(() => {
    unsubTheme();
    unsubContent();
    view?.destroy();
  });
</script>

<div class="editor-wrapper" bind:this={container}></div>

<style>
  .editor-wrapper {
    flex: 1;
    height: 100%;
    overflow: hidden;
    display: flex;
    flex-direction: column;
  }

  :global(.editor-wrapper .cm-editor) {
    height: 100%;
    width: 100%;
  }

  :global(.editor-wrapper .cm-scroller) {
    overflow: auto;
  }

  :global(.editor-wrapper .cm-editor.cm-focused) {
    outline: none;
  }
</style>
