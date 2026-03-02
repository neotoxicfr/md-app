# MD ✏️

> **Open-source markdown editor & file manager** — beautiful, fast, self-hosted.

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.25-00ADD8?logo=go)](https://go.dev)
[![Svelte](https://img.shields.io/badge/Svelte-5-FF3E00?logo=svelte)](https://svelte.dev)
[![Docker](https://img.shields.io/badge/Docker-ready-2496ED?logo=docker)](docker-compose.nas.yml)

---

## What is MD?

**MD** is a self-hosted webapp for writing, managing and exporting Markdown documents. It combines a split-pane editor with a live, typographically polished preview, and supports exporting to a wide range of formats — all from a clean, minimal interface designed for long-form writing in 2026.

It is distributed under the **MIT licence** and designed to run on your own infrastructure behind your existing security stack (Traefik, CrowdSec, Redis, etc.).

---

## Features

| Category | Details |
|---|---|
| **Markdown engine** | Full [CommonMark](https://commonmark.org) + GFM (tables, task lists, strikethrough, autolinks) |
| **Extended syntax** | Footnotes, definition lists, typographic quotes, emoji `:fire:` 🔥, frontmatter (YAML) |
| **Images** | Full image rendering in preview (standalone + linked badges), lazy-loading, external images via HTTPS |
| **Code blocks** | Syntax highlighting via `highlight.js` (200+ languages), one-click copy |
| **Editor** | CodeMirror 6 – markdown syntax highlighting, line numbers, fold gutter, Vim-like keyboard, autocomplete |
| **Font picker** | Per-style font selection (H1–H5, body text). 14 fonts: Lora, Merriweather, Playfair Display, Source Serif 4, Tangerine, Inter, Roboto, Open Sans, Poppins, Exo 2, Ubuntu, Nunito Sans, Raleway, Helvetica — assign any font to any heading level or body text. "Apply everywhere" shortcut for global change. Config persisted in localStorage |
| **Typography** | Lora serif default, Inter for UI, JetBrains Mono for code |
| **Themes** | Light / Dark, auto-detects system preference, persists in localStorage |
| **View modes** | Split (editor + preview), Editor only, Preview only |
| **Sidebar** | Collapsible file manager panel — toggle via toolbar button, state persisted in localStorage |
| **Export** | HTML, PDF (via Pandoc + WeasyPrint), DOCX, ODT, EPUB, LaTeX, RST, AsciiDoc, Textile, MediaWiki, Plain text |
| **Export without save** | Export the current editor content directly — no need to save first |
| **Import** | Upload `.md`, `.txt`, `.html` files via UI drag-and-drop or API |
| **Print** | Native browser print with dedicated print stylesheet |
| **File management** | Create, read, update, delete, rename, list — stored as flat files on disk |
| **REST API** | Full JSON API with optional API key auth |
| **Cache** | Redis-backed rendered HTML cache for fast preview |
| **Security** | Traefik-first, CrowdSec middleware, HSTS, CSP, no tracking |

---

## Screenshots

> _Screenshots coming soon — run the app locally to see it live!_

---

## Tech Stack

| Layer | Technology | Version |
|---|---|---|
| Backend | Go | 1.25 |
| HTTP router | chi | v5 |
| Markdown (server) | goldmark | v1.7 |
| Cache | Redis | 7.4 |
| Frontend | SvelteKit / Svelte | 5 |
| Build tool | Vite | 6 |
| CSS | TailwindCSS | 4 |
| Editor | CodeMirror | 6 |
| Markdown (browser) | marked.js | v15 |
| Syntax highlight | highlight.js | v11 |
| Export engine | Pandoc + WeasyPrint | latest |
| Container | Docker / Alpine | 3.21 |
| Reverse proxy | Traefik | v3 |
| Security | CrowdSec | latest |

---

## Quick Start

### Prerequisites

- Docker & Docker Compose v2+
- (Optional) Traefik + CrowdSec stack already running (`infra_proxy` network)

### Local development (all-in-one)

```bash
git clone https://github.com/md-app/md.git
cd md

# Copy and adjust env vars
cp .env.example .env

# Build & start
docker compose up -d --build

# Open in browser
open http://localhost:8080
```

### NAS / Synology with Traefik

```bash
# 1. Set your env variables
cp .env.example .env
# Edit .env: MD_DOMAIN, MD_API_KEY, REDIS_PASSWORD...

# 2. Deploy
docker compose -f docker-compose.nas.yml up -d --build

# 3. Your app is now available at https://MD_DOMAIN
```

### Frontend development

```bash
cd web
npm install
npm run dev      # starts on http://localhost:5173 (proxies /api → localhost:8080)
```

---

## Configuration

All configuration is done via **environment variables**.

| Variable | Default | Description |
|---|---|---|
| `MD_HTTP_ADDR` | `:8080` | HTTP listen address |
| `MD_STORAGE_PATH` | `/data` | Root directory for file storage |
| `MD_REDIS_URL` | _(empty)_ | Redis URL (disable cache if empty) |
| `MD_API_KEY` | _(empty)_ | Optional API key (`X-API-Key` header). Empty = no auth |
| `MD_APP_URL` | `http://localhost:8080` | Public URL of the app |
| `MD_CORS_ORIGINS` | `*` | Comma-separated allowed CORS origins |
| `MD_MAX_FILE_SIZE_MB` | `10` | Max upload size in MB |
| `MD_PANDOC_BINARY` | `pandoc` | Path to pandoc binary |
| `MD_OIDC_ISSUER` | _(empty)_ | OIDC issuer URL (empty = no auth). Enables SSO |
| `MD_OIDC_CLIENT_ID` | _(empty)_ | OIDC client ID |
| `MD_OIDC_CLIENT_SECRET` | _(empty)_ | OIDC client secret |
| `MD_OIDC_REDIRECT_URL` | _(empty)_ | OIDC callback URL (`https://…/api/auth/callback`) |
| `MD_OIDC_SESSION_KEY` | _(random)_ | HMAC key for session cookies |

See [`.env.example`](.env.example) for a fully documented sample.

---

## REST API

Base URL: `https://your-domain/api`

| Method | Path | Description |
|---|---|---|
| `GET` | `/health` | Health check |
| `GET` | `/api/files` | List all documents |
| `POST` | `/api/files` | Create a document `{name, content, path?}` |
| `GET` | `/api/files/:id` | Get document with content |
| `PUT` | `/api/files/:id` | Update document `{name, content}` |
| `DELETE` | `/api/files/:id` | Delete document |
| `GET` | `/api/files/:id/render` | Get rendered HTML (cached) |
| `POST` | `/api/files/render` | Ad-hoc render `{content}` |
| `GET` | `/api/files/:id/export/html` | Export as standalone HTML |
| `POST` | `/api/files/:id/export/:format` | Export (pdf, docx, odt, epub, latex, rst, asciidoc, textile, mediawiki, plain) |
| `POST` | `/api/files/import` | Import via multipart form (`file` field) |
| `POST` | `/api/export/raw/:format` | Export raw content without saving `{content, name}` |
| `GET` | `/api/export/formats` | List supported export formats |
| `GET` | `/api/templates` | List 8 built-in templates |
| `GET` | `/api/templates/:id` | Get template with full content |
| `GET` | `/api/search?q=…&path=…` | Full-text search across documents |
| `GET` | `/api/files/:id/versions` | List version history |
| `GET` | `/api/files/:id/versions/:vid` | Get version content |
| `POST` | `/api/files/:id/versions/:vid/restore` | Restore a version |
| `GET` | `/api/files/:id/events` | SSE stream for collaborative editing |
| `POST` | `/api/files/:id/broadcast` | Broadcast edit to collaborators |
| `GET` | `/api/webhooks` | List webhooks |
| `POST` | `/api/webhooks` | Create webhook `{url, events[], secret}` |
| `PUT` | `/api/webhooks/:id` | Update webhook |
| `DELETE` | `/api/webhooks/:id` | Delete webhook |
| `GET` | `/api/plugins` | List loaded plugins |
| `GET` | `/api/auth/login` | OIDC login redirect _(when configured)_ |
| `GET` | `/api/auth/callback` | OIDC callback |
| `GET` | `/api/auth/me` | Current user info |
| `GET` | `/api/auth/logout` | Logout |

**Authentication** (when `MD_API_KEY` is set):
```http
X-API-Key: your-key
```
or `?api_key=your-key` query param.

---

## Export Formats

| Format | File | Engine |
|---|---|---|
| HTML | `.html` | goldmark (Go, built-in) |
| PDF | `.pdf` | Pandoc + WeasyPrint |
| Word | `.docx` | Pandoc |
| OpenDocument | `.odt` | Pandoc |
| EPUB | `.epub` | Pandoc |
| LaTeX | `.tex` | Pandoc |
| reStructuredText | `.rst` | Pandoc |
| AsciiDoc | `.adoc` | Pandoc |
| Textile | `.textile` | Pandoc |
| MediaWiki | `.wiki` | Pandoc |
| Plain text | `.txt` | Pandoc |

> **Note**: PDF and non-HTML formats require `pandoc` (and `weasyprint` for PDF) to be present in the runtime environment. These are pre-installed in the production Docker image.

---

## Keyboard Shortcuts

| Shortcut | Action |
|---|---|
| `Ctrl/⌘ + S` | Save document |
| `Ctrl/⌘ + B` | Bold selection |
| `Ctrl/⌘ + I` | Italic selection |
| `Ctrl/⌘ + K` | Insert link |
| `Ctrl/⌘ + Z` / `Ctrl/⌘ + Y` | Undo / Redo |
| `Ctrl/⌘ + F` | Search in editor |
| `Tab` | Indent |
| `Shift + Tab` | Dedent |

---

## Security Architecture

```
Internet → Traefik (TLS termination)
             │
             ├── CrowdSec Bouncer (rate limiting, IP reputation)
             │
             └── MD (Go API + SvelteKit SPA)
                   │
                   └── Redis (cache, localhost only)
```

- All traffic is HTTPS-only (HSTS enforced via Traefik labels)
- CrowdSec provides IP reputation filtering and rate limiting
- HTTP security headers: `X-Frame-Options`, `X-Content-Type-Options`, `CSP`, `Referrer-Policy`
- No authentication cookies — optional stateless API key auth
- Non-root Docker user (`md:md`)
- `no-new-privileges` security option enabled
- Redis bound to internal Docker network only

---

## Project Structure

```
apps/md/           ← (rename to "md" in production)
├── cmd/
│   └── server/
│       └── main.go         ← Entry point
├── internal/
│   ├── api/
│   │   ├── router.go       ← chi router + middleware wiring
│   │   ├── files.go        ← CRUD handlers + markdown render
│   │   ├── export.go       ← Multi-format export via Pandoc
│   │   ├── health.go       ← Health + JSON helpers
│   │   └── middleware.go   ← Logging, API key, security headers
│   ├── config/
│   │   └── config.go       ← Env-based configuration
│   ├── storage/
│   │   └── storage.go      ← File system CRUD (JSON meta + .md content)
│   └── cache/
│       └── redis.go        ← Redis client wrapper
├── web/                    ← SvelteKit 5 frontend
│   ├── src/
│   │   ├── App.svelte      ← Root component, layout
│   │   ├── main.ts         ← Entry point
│   │   ├── app.css         ← Global styles + prose + CodeMirror overrides
│   │   └── lib/
│   │       ├── api.ts      ← Typed API client
│   │       ├── stores/
│   │       │   └── files.ts ← Svelte stores + async actions
│   │       ├── components/
│   │       │   ├── Sidebar.svelte      ← Collapsible file list + search + import
│   │       │   ├── Toolbar.svelte      ← Title, sidebar toggle, view toggle, save, export
│   │       │   ├── Editor.svelte       ← CodeMirror 6 editor
│   │       │   ├── Preview.svelte      ← marked.js live preview (images, links, badges)
│   │       │   ├── ExportModal.svelte  ← Format picker + download
│   │       │   ├── FontPicker.svelte   ← Per-style font picker (H1–H5, body)
│   │       │   └── Particles.svelte    ← Canvas particle animation
│   ├── package.json
│   ├── vite.config.ts
│   ├── svelte.config.js
│   └── tsconfig.json
├── pandoc/
│   └── print.css           ← PDF/print stylesheet
├── Dockerfile.app          ← Multi-stage: Node → Go → Alpine+Pandoc
├── docker-compose.yml      ← Local dev
├── docker-compose.nas.yml  ← NAS Synology + Traefik + CrowdSec
├── docker-compose.cloud.yml ← Cloud/VPS + Traefik
├── go.mod
├── .env.example
├── .gitignore
└── README.md
```

---

## Development Guide

### Run backend only (Go)

```bash
# Install dependencies
go mod tidy

# Run (hot reload with Air)
go run ./cmd/server

# With custom config
MD_HTTP_ADDR=:9090 MD_STORAGE_PATH=./dev-data go run ./cmd/server
```

### Run frontend only (Vite)

```bash
cd web
npm install
npm run dev       # http://localhost:5173 → proxies /api to localhost:8080
```

### Build production image

```bash
docker build \
  -f Dockerfile.app \
  --build-arg VERSION=1.0.0 \
  --build-arg GIT_SHA=$(git rev-parse --short HEAD) \
  --build-arg BUILD_DATE=$(date -u +%Y-%m-%dT%H:%M:%SZ) \
  -t md/app:1.0.0 .
```

### Run tests

```bash
# Go tests
go test -v ./internal/...

# Frontend
cd web && npm test
```

---

## Roadmap

- [x] **Templates** – 8 built-in starter templates (blog post, meeting notes, RFC, README, changelog, tutorial, report, letter) with picker UI
- [x] **Search** – full-text search across all documents (name + content, Ctrl+K shortcut)
- [x] **Version history** – automatic versioning on every save, preview & one-click restore
- [x] **Mermaid diagrams** – lazy-loaded live diagram rendering in preview (dark theme)
- [x] **Math (KaTeX)** – block (`$$…$$`) and inline (`$…$`) LaTeX equation rendering
- [x] **Offline mode** – full PWA with service worker, network-first API / cache-first assets
- [x] **Collaborative editing** – SSE-based real-time co-editing with presence tracking
- [x] **Plugin system** – extensible pipeline with 4 built-in plugins (TOC, word count, reading time, auto-link)
- [x] **S3/SFTP backends** – `StorageBackend` interface with local + S3 adapters
- [x] **Webhook notifications** – CRUD + HMAC-SHA256 signed async delivery with retries on save/create/delete
- [x] **LDAP / SSO authentication** – OIDC/SSO with discovery, JWKS, RS256 JWT, session cookies
- [ ] **Folders & tree navigation** – hierarchical file organization (backend supports paths, UI planned)

---

## Contributing

Contributions are welcome! Please:

1. Fork this repository
2. Create a feature branch (`git checkout -b feat/your-feature`)
3. Commit with clear messages
4. Push and open a Pull Request

Please run `go vet ./...` and `cd web && npm run typecheck` before submitting.

---

## Licence

MIT © 2026 [MD Contributors](https://github.com/md-app/md/graphs/contributors)

---

## Acknowledgements

- [goldmark](https://github.com/yuin/goldmark) – excellent Go markdown parser
- [CodeMirror 6](https://codemirror.net/) – powerful browser-based editor
- [marked.js](https://marked.js.org/) – fast browser markdown rendering
- [highlight.js](https://highlightjs.org/) – syntax highlighting
- [Pandoc](https://pandoc.org/) – the universal document converter
- [Svelte 5](https://svelte.dev/) – reactive, fast UI framework
- [Tailwind CSS 4](https://tailwindcss.com/) – utility-first CSS
- [Lora](https://fonts.google.com/specimen/Lora) + [inter](https://rsms.me/inter/) – beautiful open-source fonts
- Tangerine – self-hosted custom font (Emilie Vizcano)

---

<p align="center">
  <img src="https://upload.wikimedia.org/wikipedia/en/c/c3/Flag_of_France.svg" alt="FR" width="20" height="14" />
  &nbsp;
  <strong>MD</strong>, a <a href="https://cybergraphe.fr"><strong>Cybergraphe</strong></a> product.
</p>
