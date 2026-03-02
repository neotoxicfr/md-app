// MD API client

const BASE = '/api';

export interface FileMeta {
  id: string;
  name: string;
  slug: string;
  path: string;
  size: number;
  hash: string;
  created_at: string;
  updated_at: string;
}

export interface FileWithContent extends FileMeta {
  content: string;
}

export interface RenderResult {
  html: string;
  name?: string;
}

export interface TemplateSummary {
  id: string;
  name: string;
  description: string;
  category: string;
}

export interface TemplateDetail extends TemplateSummary {
  content: string;
}

export interface SearchResult {
  file_id: string;
  name: string;
  path: string;
  line: number;
  snippet: string;
}

export interface Version {
  id: string;
  file_id: string;
  hash: string;
  size: number;
  created_at: string;
  message: string;
}

export interface VersionWithContent extends Version {
  content: string;
}

export interface WebhookData {
  id: string;
  url: string;
  events: string[];
  secret: string;
  active: boolean;
  created_at: string;
}

export interface PluginInfo {
  name: string;
  description: string;
}

async function request<T>(method: string, path: string, body?: unknown): Promise<T> {
  const res = await fetch(BASE + path, {
    method,
    headers: body ? { 'Content-Type': 'application/json' } : {},
    body: body ? JSON.stringify(body) : undefined,
  });
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: res.statusText }));
    throw new Error(err.error ?? `HTTP ${res.status}`);
  }
  if (res.status === 204) return undefined as T;
  return res.json();
}

export const api = {
  // ---- Files ----
  list(): Promise<{ files: FileMeta[]; count: number }> {
    return request('GET', '/files');
  },

  create(name: string, content = '', path = ''): Promise<FileMeta> {
    return request('POST', '/files', { name, content, path });
  },

  get(id: string): Promise<FileWithContent> {
    return request('GET', `/files/${id}`);
  },

  update(id: string, name: string, content: string): Promise<FileMeta> {
    return request('PUT', `/files/${id}`, { name, content });
  },

  delete(id: string): Promise<void> {
    return request('DELETE', `/files/${id}`);
  },

  render(id: string): Promise<RenderResult> {
    return request('GET', `/files/${id}/render`);
  },

  renderRaw(content: string): Promise<RenderResult> {
    return request('POST', '/files/render', { content });
  },

  exportHTML(id: string): string {
    return `${BASE}/files/${id}/export/html`;
  },

  exportFormat(id: string, format: string): string {
    return `${BASE}/files/${id}/export/${format}`;
  },

  exportRawFormat(format: string): string {
    return `${BASE}/export/raw/${format}`;
  },

  async importFile(file: File): Promise<FileWithContent> {
    const form = new FormData();
    form.append('file', file);
    const res = await fetch(`${BASE}/files/import`, { method: 'POST', body: form });
    if (!res.ok) {
      const err = await res.json().catch(() => ({ error: res.statusText }));
      throw new Error(err.error ?? `HTTP ${res.status}`);
    }
    return res.json();
  },

  // ---- Templates ----
  listTemplates(): Promise<{ templates: TemplateSummary[]; count: number }> {
    return request('GET', '/templates');
  },

  getTemplate(id: string): Promise<TemplateDetail> {
    return request('GET', `/templates/${id}`);
  },

  // ---- Search ----
  search(q: string, path = ''): Promise<{ query: string; results: SearchResult[]; count: number }> {
    const params = new URLSearchParams({ q });
    if (path) params.set('path', path);
    return request('GET', `/search?${params}`);
  },

  // ---- Version History ----
  listVersions(fileId: string): Promise<{ file_id: string; versions: Version[]; count: number }> {
    return request('GET', `/files/${fileId}/versions`);
  },

  getVersion(fileId: string, versionId: string): Promise<VersionWithContent> {
    return request('GET', `/files/${fileId}/versions/${versionId}`);
  },

  restoreVersion(fileId: string, versionId: string): Promise<{ message: string; file: FileMeta; restored: string }> {
    return request('POST', `/files/${fileId}/versions/${versionId}/restore`);
  },

  // ---- Webhooks ----
  listWebhooks(): Promise<{ webhooks: WebhookData[]; count: number }> {
    return request('GET', '/webhooks');
  },

  createWebhook(url: string, events: string[], secret: string, active: boolean): Promise<WebhookData> {
    return request('POST', '/webhooks', { url, events, secret, active });
  },

  updateWebhook(id: string, url: string, events: string[], secret: string, active: boolean): Promise<WebhookData> {
    return request('PUT', `/webhooks/${id}`, { url, events, secret, active });
  },

  deleteWebhook(id: string): Promise<void> {
    return request('DELETE', `/webhooks/${id}`);
  },

  // ---- Plugins ----
  listPlugins(): Promise<{ plugins: PluginInfo[] }> {
    return request('GET', '/plugins');
  },

  // ---- Collaborative Editing ----
  connectSSE(fileId: string): EventSource {
    return new EventSource(`${BASE}/files/${fileId}/events`);
  },

  broadcast(fileId: string, type: string, content: string, user: string, cursor?: { line: number; ch: number }): Promise<{ delivered: number }> {
    return request('POST', `/files/${fileId}/broadcast`, { type, content, user, cursor });
  },

  // ---- Auth ----
  authMe(): Promise<{ sub: string; name: string; email: string }> {
    return request('GET', '/auth/me');
  },
};
