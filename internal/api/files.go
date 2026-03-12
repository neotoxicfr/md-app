package api

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/yuin/goldmark"
	emoji "github.com/yuin/goldmark-emoji"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
	"go.abhg.dev/goldmark/frontmatter"

	"md/internal/cache"
	"md/internal/storage"
)

// markdown engine singleton
var md = goldmark.New(
	goldmark.WithExtensions(
		extension.GFM,
		extension.DefinitionList,
		extension.Footnote,
		extension.Typographer,
		emoji.Emoji,
		&frontmatter.Extender{},
		highlighting.NewHighlighting(
			highlighting.WithStyle("github"),
			highlighting.WithGuessLanguage(true),
		),
	),
	goldmark.WithParserOptions(
		parser.WithAutoHeadingID(),
	),
	goldmark.WithRendererOptions(
		html.WithHardWraps(),
		html.WithXHTML(),
		html.WithUnsafe(), // allow raw HTML in markdown
	),
)

// renderMarkdown converts markdown content to an HTML string.
func renderMarkdown(content string) (string, error) {
	content = preprocessMarkdown(content)
	var buf bytes.Buffer
	if err := md.Convert([]byte(content), &buf); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// ---- Handlers ----

type filesHandler struct {
	store *storage.Storage
	cache *cache.Client
}

func newFilesHandler(store *storage.Storage, c *cache.Client) *filesHandler {
	return &filesHandler{store: store, cache: c}
}

// GET /api/files
func (h *filesHandler) list(w http.ResponseWriter, r *http.Request) {
	files, err := h.store.List()
	if err != nil {
		slog.Error("list files", "error", err)
		writeError(w, http.StatusInternalServerError, "could not list files")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"files": files, "count": len(files)})
}

// POST /api/files
func (h *filesHandler) create(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name    string `json:"name"`
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if strings.TrimSpace(body.Name) == "" {
		body.Name = "untitled"
	}
	f, err := h.store.Create(body.Name, body.Path, body.Content)
	if err != nil {
		slog.Error("create file", "error", err)
		writeError(w, http.StatusInternalServerError, "could not create file")
		return
	}
	writeJSON(w, http.StatusCreated, f)
}

// GET /api/files/{id}
func (h *filesHandler) get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	fwc, err := h.store.GetContent(id)
	if err != nil {
		if err == storage.ErrNotFound {
			writeError(w, http.StatusNotFound, "file not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "could not read file")
		return
	}
	writeJSON(w, http.StatusOK, fwc)
}

// PUT /api/files/{id}
func (h *filesHandler) update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body struct {
		Name    string `json:"name"`
		Content string `json:"content"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	// Auto-version: save current content before overwriting.
	if current, err := h.store.GetContent(id); err == nil {
		_, _ = h.store.SaveVersion(id, current.Content, "auto-save")
	}

	f, err := h.store.Update(id, body.Name, body.Content)
	if err != nil {
		if err == storage.ErrNotFound {
			writeError(w, http.StatusNotFound, "file not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "could not update file")
		return
	}
	// Invalidate cache
	if h.cache != nil {
		_ = h.cache.Delete(r.Context(), "render:"+id)
	}
	writeJSON(w, http.StatusOK, f)
}

// DELETE /api/files/{id}
func (h *filesHandler) delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.store.Delete(id); err != nil {
		if err == storage.ErrNotFound {
			writeError(w, http.StatusNotFound, "file not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "could not delete file")
		return
	}
	if h.cache != nil {
		_ = h.cache.Delete(r.Context(), "render:"+id)
	}
	w.WriteHeader(http.StatusNoContent)
}

// GET /api/files/{id}/render
// Returns rendered HTML (from cache if available).
func (h *filesHandler) render(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	cacheKey := "render:" + id

	// Check cache
	if h.cache != nil {
		cached, err := h.cache.Get(r.Context(), cacheKey)
		if err == nil && cached != "" {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.Header().Set("X-Cache", "HIT")
			fmt.Fprint(w, cached)
			return
		}
	}

	fwc, err := h.store.GetContent(id)
	if err != nil {
		if err == storage.ErrNotFound {
			writeError(w, http.StatusNotFound, "file not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "could not read file")
		return
	}

	rendered, err := renderMarkdown(fwc.Content)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "render error")
		return
	}

	result := map[string]string{"html": rendered, "name": fwc.Name}
	if h.cache != nil {
		if b, err := marshalJSON(result); err == nil {
			_ = h.cache.Set(context.Background(), cacheKey, string(b))
		}
	}

	w.Header().Set("X-Cache", "MISS")
	writeJSON(w, http.StatusOK, result)
}

// POST /api/files/render  (ad-hoc render without saving)
func (h *filesHandler) renderRaw(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Content string `json:"content"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	rendered, err := renderMarkdown(body.Content)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "render error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"html": rendered})
}

// POST /api/files/import  (multipart form upload)
func (h *filesHandler) importFile(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(50 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "bad multipart form")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing file field")
		return
	}
	defer file.Close()

	name := strings.TrimSuffix(header.Filename, ".md")
	name = strings.TrimSuffix(name, ".txt")
	name = strings.TrimSuffix(name, ".html")

	f, err := h.store.ImportReader(name, file)
	if err != nil {
		slog.Error("import file", "error", err)
		writeError(w, http.StatusInternalServerError, "could not import file")
		return
	}
	writeJSON(w, http.StatusCreated, f)
}

// ---- Full-page HTML export template ----
var htmlExportTmpl = template.Must(template.New("export").Parse(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>{{.Title}}</title>
<style>
  :root { --font-body: 'Georgia', serif; --font-mono: 'JetBrains Mono', 'Fira Code', monospace; }
  * { box-sizing: border-box; margin: 0; padding: 0; }
  body { font-family: var(--font-body); font-size: 18px; line-height: 1.75;
         color: #1a1a1a; background: #fff; max-width: 780px; margin: 0 auto;
         padding: 3rem 2rem; }
  h1,h2,h3,h4,h5,h6 { font-weight: 600; margin: 2rem 0 0.75rem; line-height: 1.3; }
  h1 { font-size: 2.2rem; border-bottom: 2px solid #e5e7eb; padding-bottom: 0.5rem; }
  h2 { font-size: 1.7rem; } h3 { font-size: 1.35rem; }
  p { margin: 1rem 0; }
  a { color: #2563eb; text-decoration: underline; }
  code { font-family: var(--font-mono); font-size: 0.875em; background: #f3f4f6;
         padding: 0.15em 0.4em; border-radius: 4px; }
  pre { background: #1e1e2e; color: #cdd6f4; padding: 1.25rem; border-radius: 8px;
        overflow-x: auto; margin: 1.5rem 0; font-size: 0.875rem; line-height: 1.6; }
  pre code { background: none; padding: 0; color: inherit; }
  blockquote { border-left: 4px solid #93c5fd; margin: 1.5rem 0;
               padding: 0.75rem 1.25rem; background: #eff6ff;
               color: #1e40af; border-radius: 0 6px 6px 0; }
  table { border-collapse: collapse; width: 100%; margin: 1.5rem 0; }
  th, td { border: 1px solid #e5e7eb; padding: 0.6rem 1rem; text-align: left; }
  th { background: #f9fafb; font-weight: 600; }
  tr:nth-child(even) { background: #f9fafb; }
  ul, ol { margin: 1rem 0 1rem 1.75rem; }
  li { margin: 0.35rem 0; }
  img { max-width: 100%; height: auto; border-radius: 6px; margin: 1rem 0; }
  hr { border: none; border-top: 1px solid #e5e7eb; margin: 2rem 0; }
  .task-list-item { list-style: none; margin-left: -1.75rem; padding-left: 1.75rem; }
  @media print { body { padding: 0; max-width: none; }
                 pre { break-inside: avoid; } }
</style>
</head>
<body>
{{.Body}}
</body>
</html>`))

// GET /api/files/{id}/export/html
func (h *filesHandler) exportHTML(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	fwc, err := h.store.GetContent(id)
	if err != nil {
		if err == storage.ErrNotFound {
			writeError(w, http.StatusNotFound, "file not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "could not read file")
		return
	}

	rendered, err := renderMarkdown(fwc.Content)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "render error")
		return
	}

	var buf bytes.Buffer
	if err := htmlExportTmpl.Execute(&buf, map[string]any{
		"Title": fwc.Name,
		"Body":  template.HTML(rendered), //nolint:gosec
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "template error")
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.html"`, fwc.Slug))
	w.Write(buf.Bytes())
}
