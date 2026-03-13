<script lang="ts">
  import { onMount } from 'svelte';
  import { Marked } from 'marked';
  import { markedHighlight } from 'marked-highlight';
  import markedFootnote from 'marked-footnote';
  import hljs from 'highlight.js';
  import { activeContent } from '$lib/stores/files';
  import DOMPurify from 'dompurify';

  // ── Mermaid (lazy‑loaded) ──
  let mermaidReady = false;
  let mermaidModule: typeof import('mermaid')['default'] | null = null;

  async function ensureMermaid() {
    if (mermaidModule) return;
    const m = await import('mermaid');
    mermaidModule = m.default;
    mermaidModule.initialize({ startOnLoad: false, theme: 'dark', securityLevel: 'strict' });
    mermaidReady = true;
  }

  // ── KaTeX (lazy‑loaded) ──
  let katexRender: ((tex: string, opts?: object) => string) | null = null;

  async function ensureKaTeX() {
    if (katexRender) return;
    const k = await import('katex');
    katexRender = k.default.renderToString;
  }

  // Configure Marked with extensions
  const marked = new Marked(
    markedHighlight({
      langPrefix: 'hljs language-',
      highlight(code, lang) {
        if (lang === 'mermaid') return code; // pass-through for mermaid
        const language = hljs.getLanguage(lang) ? lang : 'plaintext';
        return hljs.highlight(code, { language }).value;
      },
    }),
    {
      gfm: true,
      breaks: false,
      pedantic: false,
    }
  );

  marked.use(markedFootnote());
  marked.use({
    renderer: {
      heading(this: any, { depth, tokens }: { depth: number; tokens: any[] }): string {
        const text = this.parser.parseInline(tokens);
        const slug = text.replace(/<[^>]*>/g, '').toLowerCase().replace(/[^\w]+/g, '-');
        return `<h${depth} id="${slug}">${text}</h${depth}>\n`;
      },
      link(this: any, { href, title, tokens }: { href: string; title?: string | null; tokens: any[] }): string {
        const text = this.parser.parseInline(tokens);
        const safeHref = /^\s*javascript\s*:/i.test(href ?? '') ? '' : href ?? '';
        const escapedHref = safeHref.replace(/&/g, '&amp;').replace(/"/g, '&quot;');
        const external = safeHref.startsWith('http') && !safeHref.startsWith(window.location.origin);
        const attrs = external ? ' target="_blank" rel="noopener noreferrer"' : '';
        const t = title ? ` title="${title.replace(/&/g, '&amp;').replace(/"/g, '&quot;').replace(/</g, '&lt;').replace(/>/g, '&gt;')}"` : '';
        return `<a href="${escapedHref}"${t}${attrs}>${text}</a>`;
      },
      image({ href, title, text }: { href: string; title?: string | null; text: string }): string {
        const escAttr = (s: string) => s.replace(/&/g, '&amp;').replace(/"/g, '&quot;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
        const t = title ? ` title="${escAttr(title)}"` : '';
        return `<img src="${href}" alt="${escAttr(text)}"${t} loading="lazy">`;
      },
      code({ text, lang }: { text: string; lang?: string }): string {
        if (lang === 'mermaid') {
          return `<pre class="mermaid-block" data-mermaid>${text}</pre>`;
        }
        const language = lang && hljs.getLanguage(lang) ? lang : 'plaintext';
        const highlighted = hljs.highlight(text, { language }).value;
        return `<pre><code class="hljs language-${language}">${highlighted}</code></pre>`;
      },
    },
  });

  // ── KaTeX inline/block processing ──
  function processKaTeX(html: string): string {
    if (!katexRender) return html;
    // Block math: $$...$$
    html = html.replace(/\$\$([\s\S]+?)\$\$/g, (_match, tex) => {
      try {
        return katexRender!(tex.trim(), { displayMode: true, throwOnError: false });
      } catch { return _match; }
    });
    // Inline math: $...$  (not $$)
    html = html.replace(/(?<!\$)\$(?!\$)(.+?)(?<!\$)\$(?!\$)/g, (_match, tex) => {
      try {
        return katexRender!(tex.trim(), { displayMode: false, throwOnError: false });
      } catch { return _match; }
    });
    return html;
  }

  // ── Markdown normalization (keeps preview consistent with backend/export) ──
  function normalizeMarkdown(content: string): string {
    if (!content) return content;

    content = content.replace(/\r\n/g, '\n').replace(/\r/g, '\n').replace(/\u00a0/g, ' ');

    const lines = content.split('\n');
    const out: string[] = [];
    let inFence = false;

    const normalizeInlineTableLine = (line: string): string[] | null => {
      const trimmed = line.trim();
      if (!trimmed || !trimmed.includes('||') || (trimmed.match(/\|/g) ?? []).length < 4) {
        return null;
      }

      let normalized = line.replaceAll('—', '-').replaceAll('–', '-');
      const outRows: string[] = [];
      let hadPrefix = false;

      const firstPipe = normalized.indexOf('|');
      if (firstPipe > 0) {
        const prefix = normalized.slice(0, firstPipe).trim();
        if (prefix) {
          outRows.push(prefix);
          hadPrefix = true;
        }
        normalized = normalized.slice(firstPipe);
      }

      if (hadPrefix) outRows.push('');

      let tableRows = 0;
      for (const chunk of normalized.split('||')) {
        let row = chunk.trim();
        if (!row) continue;
        if ((row.match(/\|/g) ?? []).length < 2) {
          outRows.push(row);
          continue;
        }
        if (!row.startsWith('|')) row = `| ${row}`;
        if (!row.endsWith('|')) row = `${row} |`;

        const inner = row.replace(/^\|/, '').replace(/\|$/, '').trim();
        if (/^[\s|:\-]+$/.test(inner)) {
          const cells = row
            .replace(/^\|/, '')
            .replace(/\|$/, '')
            .split('|')
            .map((raw) => {
              const c = raw.trim();
              const left = c.startsWith(':');
              const right = c.endsWith(':');
              let sep = '---';
              if (left) sep = `:${sep}`;
              if (right) sep = `${sep}:`;
              return sep;
            });
          row = `| ${cells.join(' | ')} |`;
        }

        outRows.push(row);
        tableRows++;
      }

      if (tableRows < 2) return null;
      outRows.push('');
      return outRows;
    };

    for (let line of lines) {
      const trimmed = line.trim();

      if (trimmed.startsWith('```') || trimmed.startsWith('~~~')) {
        inFence = !inFence;
        out.push(line);
        continue;
      }

      if (inFence) {
        out.push(line);
        continue;
      }

      line = line.replace(/^\s{4,}(#{1,6}\s+)/, '$1');

      const t = line.trim();
      if (t.startsWith('• ')) {
        line = `- ${t.slice(2)}`;
      } else if (t.startsWith('◦ ')) {
        line = `  - ${t.slice(2)}`;
      }

      if (!line.trimStart().startsWith('#') && /\s+(#{1,6}\s+)/.test(line)) {
        line = line.replace(/\s+(#{1,6}\s+)/g, '\n\n$1');
      }

      if (!/^\s*[-*+]/.test(line) && /\s+•\s+/.test(line)) {
        line = line.replace(/\s+•\s+/g, '\n- ');
      }

      if (/\s+◦\s+/.test(line)) {
        line = line.replace(/\s+◦\s+/g, '\n  - ');
      }

      for (const segment of line.split('\n')) {
        const tableLines = normalizeInlineTableLine(segment);
        if (tableLines) out.push(...tableLines);
        else out.push(segment);
      }
    }

    return out.join('\n');
  }

  // ── Page break marker regex (matches \newpage, \pagebreak, <!-- pagebreak -->, --- pagebreak ---) ──
  const pageBreakRe = /^(\\(?:newpage|pagebreak)\s*$|<!--\s*pagebreak\s*-->\s*$|---\s*pagebreak\s*---\s*$)/gm;
  const pageBreakHtml = '<div class="pagebreak-indicator" aria-label="Page break"><span>⸻ Saut de page ⸻</span></div>';

  // ── Reactive state (Svelte 5 runes) ──
  let renderedHtml = $state('');
  let container = $state<HTMLElement | undefined>(undefined);
  let mermaidCounter = 0;
  let renderTimer: ReturnType<typeof setTimeout>;

  // Re‑render on content change (debounced to avoid thrashing during fast typing)
  $effect(() => {
    const content = $activeContent;
    // Pre-load KaTeX/Mermaid eagerly so they're ready when render fires
    if (content.includes('$')) ensureKaTeX();
    if (content.includes('```mermaid')) ensureMermaid();

    clearTimeout(renderTimer);
    renderTimer = setTimeout(() => {
      try {
        const preprocessed = normalizeMarkdown(content).replace(pageBreakRe, pageBreakHtml);
        let html = marked.parse(preprocessed) as string;
        html = processKaTeX(html);
        renderedHtml = DOMPurify.sanitize(html, {
          ADD_TAGS: ['math', 'semantics', 'mrow', 'mi', 'mo', 'mn', 'msup', 'msub', 'mfrac', 'munder', 'mover', 'munderover', 'msqrt', 'mroot', 'mtable', 'mtr', 'mtd', 'mtext', 'mspace', 'annotation'],
          ADD_ATTR: ['xmlns', 'display', 'mathvariant', 'encoding', 'data-mermaid', 'data-copy', 'data-rendered'],
        });
      } catch {
        renderedHtml = `<p class="render-error">Render error</p>`;
      }
    }, 150);

    return () => clearTimeout(renderTimer);
  });

  // Post-render: copy buttons, emojis, mermaid diagrams
  $effect(() => {
    if (!container || !renderedHtml) return;
    let cancelled = false;
    const timeoutId = setTimeout(async () => {
      if (cancelled) return;
      // Copy buttons on code blocks
      container?.querySelectorAll('pre:not([data-copy]):not([data-mermaid])').forEach((pre) => {
        pre.setAttribute('data-copy', '1');
        const btn = document.createElement('button');
        btn.className = 'copy-btn';
        btn.textContent = 'Copy';
        btn.addEventListener('click', () => {
          const code = pre.querySelector('code');
          if (!code) return;
          navigator.clipboard.writeText(code.textContent ?? '').then(() => {
            btn.textContent = 'Copied!';
            btn.classList.add('copied');
            setTimeout(() => {
              btn.textContent = 'Copy';
              btn.classList.remove('copied');
            }, 1800);
          }).catch(() => {
            btn.textContent = 'Failed';
            setTimeout(() => { btn.textContent = 'Copy'; }, 1800);
          });
        });
        pre.appendChild(btn);
      });

      // Emoji shortcodes
      container?.querySelectorAll('p, li, h1, h2, h3, h4, h5, h6').forEach((el) => {
        if (el.children.length === 0 && el.textContent?.includes(':')) {
          el.innerHTML = el.innerHTML.replace(/:([a-z0-9_+-]+):/g, (match, name) => {
            const emoji = emojiMap[name];
            return emoji ? `<span class="emoji" title=":${name}:">${emoji}</span>` : match;
          });
        }
      });

      // Mermaid diagrams
      if (mermaidReady && mermaidModule) {
        const blocks = container?.querySelectorAll('pre[data-mermaid]:not([data-rendered])') ?? [];
        for (const block of blocks) {
          block.setAttribute('data-rendered', '1');
          const src = block.textContent ?? '';
          try {
            mermaidCounter++;
            const { svg } = await mermaidModule.render(`mermaid-${mermaidCounter}`, src);
            const div = document.createElement('div');
            div.className = 'mermaid-diagram';
            div.innerHTML = svg;
            block.replaceWith(div);
          } catch (err) {
            block.classList.add('mermaid-error');
            console.warn('Mermaid render failed:', err);
          }
        }
      }
    }, 0);
    return () => { cancelled = true; clearTimeout(timeoutId); };
  });

  onMount(() => {
    document.title = 'MD';
  });

  const emojiMap: Record<string, string> = {
    smile: '😊', thumbsup: '👍', heart: '❤️', fire: '🔥', star: '⭐',
    rocket: '🚀', check: '✅', warning: '⚠️', info: 'ℹ️', bulb: '💡',
    eyes: '👀', tada: '🎉', wave: '👋', point_right: '👉', ok_hand: '👌',
    zap: '⚡', lock: '🔒', key: '🔑', bug: '🐛', wrench: '🔧',
    pencil: '✏️', book: '📖', folder: '📁', file: '📄', computer: '💻',
    coffee: '☕', pizza: '🍕', music: '🎵', art: '🎨', camera: '📷',
  };
</script>

<div class="preview-wrapper">
  <article
    class="prose preview-content"
    bind:this={container}
  >
    {#if renderedHtml}
      {@html renderedHtml}
    {:else}
      <div class="preview-empty">
        <div class="empty-icon">
          <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M14 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V8z"/>
            <polyline points="14 2 14 8 20 8"/>
            <line x1="16" y1="13" x2="8" y2="13"/>
            <line x1="16" y1="17" x2="8" y2="17"/>
            <polyline points="10 9 9 9 8 9"/>
          </svg>
        </div>
        <p>Start writing to see a live preview</p>
        <span class="empty-hint">Supports full CommonMark, GFM tables, footnotes, emoji & syntax highlighting</span>
      </div>
    {/if}
  </article>
</div>

<style>
  .preview-wrapper {
    flex: 1;
    height: 100%;
    overflow-y: auto;
    position: relative;
  }

  .preview-content {
    padding: 2.5rem 3rem;
    max-width: 780px;
    margin: 0 auto;
    min-height: 100%;
  }

  .preview-empty {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 0.75rem;
    height: 50vh;
    color: var(--text-muted);
    font-size: 15px;
    font-family: var(--font-ui);
    text-align: center;
  }

  .empty-icon {
    opacity: 0.2;
    margin-bottom: 0.5rem;
  }

  .empty-hint {
    font-size: 12px;
    color: var(--text-muted);
    opacity: 0.6;
    max-width: 280px;
  }

  .render-error {
    color: var(--danger);
    font-family: var(--font-ui);
  }

  /* Copy button injected into pre */
  :global(pre) { position: relative !important; }

  :global(.copy-btn) {
    position: absolute;
    top: 0.5rem;
    right: 0.5rem;
    padding: 0.2rem 0.6rem;
    font-size: 11px;
    font-family: var(--font-ui);
    font-weight: 500;
    background: rgba(255, 255, 255, 0.06);
    color: rgba(255, 255, 255, 0.5);
    border: 1px solid rgba(255, 255, 255, 0.08);
    border-radius: var(--radius-sm);
    cursor: pointer;
    transition: all 0.2s;
    backdrop-filter: blur(8px);
    opacity: 0;
  }
  :global(pre:hover .copy-btn) { opacity: 1; }
  :global(.copy-btn:hover) {
    background: rgba(255, 255, 255, 0.12);
    color: rgba(255, 255, 255, 0.8);
  }
  :global(.copy-btn.copied) {
    background: rgba(16, 185, 129, 0.2);
    color: #10b981;
    border-color: rgba(16, 185, 129, 0.3);
  }

  /* Print */
  @media print {
    .preview-wrapper { border: none; }
    .preview-content { max-width: 100%; padding: 0; }
  }
</style>
