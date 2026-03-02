import { writable, derived, get } from 'svelte/store';
import { api, type FileMeta, type FileWithContent } from '$lib/api';

// ---- State ----

export const files = writable<FileMeta[]>([]);
export const activeFileId = writable<string | null>(null);
export const activeContent = writable<string>('');
export const activeName = writable<string>('untitled');
export const isDirty = writable<boolean>(false);
export const isSaving = writable<boolean>(false);
export const isLoading = writable<boolean>(false);
export const error = writable<string | null>(null);
export const searchQuery = writable<string>('');
export const theme = writable<'light' | 'dark'>('light');
export const viewMode = writable<'split' | 'editor' | 'preview'>('split');
export const previewFont = writable<string>('Lora');
export const sidebarOpen = writable<boolean>(false);

// ---- Font Config: headings + body ----
export interface FontConfig {
  headings: string;
  body: string;
}

const defaultFontConfig: FontConfig = { headings: 'Lora', body: 'Lora' };

export const fontConfig = writable<FontConfig>({ ...defaultFontConfig });

// ---- Derived ----

export const filteredFiles = derived(
  [files, searchQuery],
  ([$files, $q]) => {
    const q = $q.trim().toLowerCase();
    if (!q) return $files;
    return $files.filter((f) => f.name.toLowerCase().includes(q) || f.slug.includes(q));
  }
);

export const activeFile = derived(
  [files, activeFileId],
  ([$files, $id]) => $files.find((f) => f.id === $id) ?? null
);

// ---- Actions ----

export async function loadFiles(): Promise<void> {
  isLoading.set(true);
  error.set(null);
  try {
    const res = await api.list();
    files.set(res.files ?? []);
  } catch (e: unknown) {
    error.set(e instanceof Error ? e.message : 'Failed to load files');
  } finally {
    isLoading.set(false);
  }
}

export async function openFile(id: string): Promise<void> {
  isLoading.set(true);
  error.set(null);
  try {
    const fwc: FileWithContent = await api.get(id);
    activeFileId.set(fwc.id);
    activeName.set(fwc.name);
    activeContent.set(fwc.content);
    isDirty.set(false);
  } catch (e: unknown) {
    error.set(e instanceof Error ? e.message : 'Failed to open file');
  } finally {
    isLoading.set(false);
  }
}

export async function saveActiveFile(): Promise<void> {
  const id = get(activeFileId);
  const name = get(activeName);
  const content = get(activeContent);

  if (!id) {
    // New unsaved file
    await createFile(name, content);
    return;
  }

  isSaving.set(true);
  error.set(null);
  try {
    const updated = await api.update(id, name, content);
    isDirty.set(false);
    files.update((fs) => fs.map((f) => (f.id === id ? updated : f)));
  } catch (e: unknown) {
    error.set(e instanceof Error ? e.message : 'Failed to save');
  } finally {
    isSaving.set(false);
  }
}

export async function createFile(name = 'untitled', content = ''): Promise<void> {
  isLoading.set(true);
  error.set(null);
  try {
    const f = await api.create(name, content);
    files.update((fs) => [f, ...fs]);
    activeFileId.set(f.id);
    activeName.set(f.name);
    activeContent.set(content);
    isDirty.set(false);
  } catch (e: unknown) {
    error.set(e instanceof Error ? e.message : 'Failed to create file');
  } finally {
    isLoading.set(false);
  }
}

export async function deleteFile(id: string): Promise<void> {
  try {
    await api.delete(id);
    files.update((fs) => fs.filter((f) => f.id !== id));
    if (get(activeFileId) === id) {
      activeFileId.set(null);
      activeName.set('untitled');
      activeContent.set('');
      isDirty.set(false);
    }
  } catch (e: unknown) {
    error.set(e instanceof Error ? e.message : 'Failed to delete file');
  }
}

export async function importFile(file: File): Promise<void> {
  isLoading.set(true);
  error.set(null);
  try {
    const fwc = await api.importFile(file);
    files.update((fs) => [fwc, ...fs]);
    activeFileId.set(fwc.id);
    activeName.set(fwc.name);
    activeContent.set(fwc.content);
    isDirty.set(false);
  } catch (e: unknown) {
    error.set(e instanceof Error ? e.message : 'Failed to import file');
  } finally {
    isLoading.set(false);
  }
}

// Auto-save: updated content triggers dirty state
export function setContent(c: string): void {
  activeContent.set(c);
  isDirty.set(true);
}

// Toggle theme
export function toggleTheme(): void {
  theme.update((t) => {
    const next = t === 'light' ? 'dark' : 'light';
    document.documentElement.setAttribute('data-theme', next);
    localStorage.setItem('md-theme', next);
    return next;
  });
}

// Font management
export const fontOptions = [
  { name: 'Lora', family: "'Lora', Georgia, serif", category: 'serif' },
  { name: 'Merriweather', family: "'Merriweather', Georgia, serif", category: 'serif' },
  { name: 'Playfair Display', family: "'Playfair Display', Georgia, serif", category: 'serif' },
  { name: 'Source Serif 4', family: "'Source Serif 4', Georgia, serif", category: 'serif' },
  { name: 'Tangerine', family: "'Tangerine', serif", category: 'serif' },
  { name: 'Inter', family: "'Inter', system-ui, sans-serif", category: 'sans-serif' },
  { name: 'Roboto', family: "'Roboto', system-ui, sans-serif", category: 'sans-serif' },
  { name: 'Open Sans', family: "'Open Sans', system-ui, sans-serif", category: 'sans-serif' },
  { name: 'Poppins', family: "'Poppins', system-ui, sans-serif", category: 'sans-serif' },
  { name: 'Exo 2', family: "'Exo 2', system-ui, sans-serif", category: 'sans-serif' },
  { name: 'Ubuntu', family: "'Ubuntu', system-ui, sans-serif", category: 'sans-serif' },
  { name: 'Nunito Sans', family: "'Nunito Sans', system-ui, sans-serif", category: 'sans-serif' },
  { name: 'Raleway', family: "'Raleway', system-ui, sans-serif", category: 'sans-serif' },
  { name: 'Helvetica', family: "Helvetica, Arial, sans-serif", category: 'sans-serif' },
];

export function setPreviewFont(fontName: string): void {
  previewFont.set(fontName);
  localStorage.setItem('md-font', fontName);
  const font = fontOptions.find((f) => f.name === fontName);
  if (font) {
    document.documentElement.style.setProperty('--font-prose', font.family);
  }
}

/** Apply headings or body font */
export function setFontFor(slot: 'headings' | 'body', fontName: string): void {
  fontConfig.update((cfg) => {
    const next = { ...cfg, [slot]: fontName };
    localStorage.setItem('md-fontconfig', JSON.stringify(next));
    applyFontConfig(next);
    return next;
  });
}

function applyFontConfig(cfg: FontConfig): void {
  const root = document.documentElement;
  const hFont = fontOptions.find((f) => f.name === cfg.headings);
  const bFont = fontOptions.find((f) => f.name === cfg.body);
  if (hFont) root.style.setProperty('--font-headings', hFont.family);
  if (bFont) {
    root.style.setProperty('--font-body', bFont.family);
    root.style.setProperty('--font-prose', bFont.family);
    previewFont.set(cfg.body);
    localStorage.setItem('md-font', cfg.body);
  }
}

// Init theme from localStorage
export function initTheme(): void {
  const saved = localStorage.getItem('md-theme') as 'light' | 'dark' | null;
  const preferred = window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
  const t = saved ?? preferred;
  theme.set(t);
  document.documentElement.setAttribute('data-theme', t);

  // Restore font config (headings + body) or legacy single-font
  const savedConfig = localStorage.getItem('md-fontconfig');
  if (savedConfig) {
    try {
      const raw = JSON.parse(savedConfig);
      // Migrate from old per-H1..H5 config to simplified headings+body
      const cfg: FontConfig = {
        headings: raw.headings ?? raw.h1 ?? 'Lora',
        body: raw.body ?? 'Lora',
      };
      fontConfig.set(cfg);
      applyFontConfig(cfg);
    } catch {
      const savedFont = localStorage.getItem('md-font');
      if (savedFont) setPreviewFont(savedFont);
    }
  } else {
    const savedFont = localStorage.getItem('md-font');
    if (savedFont) setPreviewFont(savedFont);
  }

  // Restore sidebar preference (default: collapsed)
  const savedSidebar = localStorage.getItem('md-sidebar');
  sidebarOpen.set(savedSidebar === 'true');
}

export function toggleSidebar(): void {
  sidebarOpen.update((v) => {
    const next = !v;
    localStorage.setItem('md-sidebar', String(next));
    return next;
  });
}
