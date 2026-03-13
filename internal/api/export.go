package api

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"

	"md/internal/config"
	"md/internal/storage"
)

var stderrBufPool = sync.Pool{
	New: func() any { return new(bytes.Buffer) },
}

type exportHandler struct {
	store *storage.Storage
	cfg   *config.Config
}

func newExportHandler(store *storage.Storage, cfg *config.Config) *exportHandler {
	return &exportHandler{store: store, cfg: cfg}
}

// ─── Pandoc input format ────────────────────────────────────
// Comprehensive format string that matches the GFM-like rendering of
// marked.js in the webapp preview.  Every extension is explicit so that
// upgrades to the Pandoc version never silently remove support.
const pandocInputFmt = "markdown" +
	"+pipe_tables" +
	"+grid_tables" +
	"+multiline_tables" +
	"+simple_tables" +
	"+table_captions" +
	"+strikeout" +
	"+task_lists" +
	"+definition_lists" +
	"+footnotes" +
	"+smart" +
	"+emoji" +
	"+autolink_bare_uris" +
	"+raw_html" +
	"+fenced_code_blocks" +
	"+backtick_code_blocks" +
	"+fenced_code_attributes" +
	"+inline_code_attributes" +
	"+yaml_metadata_block" +
	"+tex_math_dollars" +
	"+superscript" +
	"+subscript" +
	"+abbreviations" +
	"+header_attributes"

// ─── Page-break support ─────────────────────────────────────
const pageBreakDiv = `<div class="pagebreak"></div>`

var rePageBreak = regexp.MustCompile(`(?m)^\\(?:newpage|pagebreak)\s*$|^<!--\s*pagebreak\s*-->\s*$|^---\s*pagebreak\s*---\s*$`)

func preprocessPageBreaks(content string) string {
	return rePageBreak.ReplaceAllString(content, pageBreakDiv)
}

// ─── Margin presets ─────────────────────────────────────────
type pdfMargins struct {
	Top    string
	Right  string
	Bottom string
	Left   string
}

var marginPresets = map[string]pdfMargins{
	"standard": {"2.2cm", "2.5cm", "2.5cm", "2.5cm"},
	"narrow":   {"1.5cm", "1.5cm", "1.5cm", "1.5cm"},
	"wide":     {"2.5cm", "3.5cm", "2.5cm", "3.5cm"},
}

// parseMargins reads margin settings from the HTTP request.
// Accepts: ?margin=standard|narrow|wide  (preset)
//
//	?mt=2&mr=2&mb=2&ml=2     (custom, in cm)
func parseMargins(r *http.Request) pdfMargins {
	preset := r.URL.Query().Get("margin")
	if m, ok := marginPresets[preset]; ok {
		return m
	}
	// Custom margins (cm) — all four must be set, otherwise use standard
	mt := r.URL.Query().Get("mt")
	mr := r.URL.Query().Get("mr")
	mb := r.URL.Query().Get("mb")
	ml := r.URL.Query().Get("ml")
	if mt != "" && mr != "" && mb != "" && ml != "" {
		return pdfMargins{asCM(mt), asCM(mr), asCM(mb), asCM(ml)}
	}
	return marginPresets["standard"]
}

func asCM(v string) string {
	if _, err := strconv.ParseFloat(v, 64); err == nil {
		return v + "cm"
	}
	return "2.5cm"
}

// marginOverrideCSS generates a <style> block that overrides @page margins.
func marginOverrideCSS(m pdfMargins) string {
	return fmt.Sprintf(
		"<style>@page { margin: %s %s %s %s; } @page:first { margin-top: %s; }</style>",
		m.Top, m.Right, m.Bottom, m.Left, m.Top,
	)
}

// ─── Supported export formats ───────────────────────────────
var pandocFormats = map[string]struct {
	ext         string
	contentType string
	to          string
}{
	"docx":      {".docx", "application/vnd.openxmlformats-officedocument.wordprocessingml.document", "docx"},
	"odt":       {".odt", "application/vnd.oasis.opendocument.text", "odt"},
	"epub":      {".epub", "application/epub+zip", "epub"},
	"rst":       {".rst", "text/plain; charset=utf-8", "rst"},
	"latex":     {".tex", "application/x-tex", "latex"},
	"pdf":       {".pdf", "application/pdf", "pdf"},
	"mediawiki": {".wiki", "text/plain; charset=utf-8", "mediawiki"},
	"asciidoc":  {".adoc", "text/plain; charset=utf-8", "asciidoc"},
	"textile":   {".textile", "text/plain; charset=utf-8", "textile"},
	"jira":      {".jira", "text/plain; charset=utf-8", "jira"},
	"plain":     {".txt", "text/plain; charset=utf-8", "plain"},
}

// POST /api/files/{id}/export/{format}
func (h *exportHandler) export(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	format := strings.ToLower(chi.URLParam(r, "format"))

	fmtInfo, ok := pandocFormats[format]
	if !ok {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("unsupported format: %s", format))
		return
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

	// Write input to a temp file (pandoc reads from stdin or file)
	tmpDir, err := os.MkdirTemp("", "md-export-*")
	if err != nil {
		slog.Error("create tmpdir", "error", err)
		writeError(w, http.StatusInternalServerError, "export failed")
		return
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			slog.Warn("cleanup tmp dir failed", "path", tmpDir, "error", err)
		}
	}()

	inputFile := filepath.Join(tmpDir, "input.md")
	outputFile := filepath.Join(tmpDir, "output"+fmtInfo.ext)

	// Preprocess page breaks for PDF export
	content := preprocessMarkdown(fwc.Content)
	if format == "pdf" {
		content = preprocessPageBreaks(content)
	}

	if err := os.WriteFile(inputFile, []byte(content), 0600); err != nil {
		writeError(w, http.StatusInternalServerError, "export failed")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 90*time.Second)
	defer cancel()

	// PDF: two-step pipeline (Pandoc → HTML → WeasyPrint → PDF)
	if format == "pdf" {
		margins := parseMargins(r)
		htmlFile := filepath.Join(tmpDir, "output.html")
		if err := h.runPDFExport(ctx, inputFile, htmlFile, outputFile, margins); err != nil {
			slog.Error("pdf export failed", "file", fwc.Name, "error", err)
			writeError(w, http.StatusInternalServerError, "export conversion failed")
			return
		}
	} else {
		if err := h.runPandocExport(ctx, inputFile, outputFile, fmtInfo.to); err != nil {
			slog.Error("pandoc export failed", "format", format, "file", fwc.Name, "error", err)
			writeError(w, http.StatusInternalServerError, "export conversion failed")
			return
		}
	}

	if err := streamFile(w, outputFile, fmtInfo.contentType, fwc.Slug+fmtInfo.ext); err != nil {
		slog.Warn("write export response failed", "error", err)
	}
}

// streamFile opens a file and streams it to the response writer.
func streamFile(w http.ResponseWriter, path, contentType, filename string) error {
	f, err := os.Open(path)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not read export output")
		return err
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not read export output")
		return err
	}

	safeFilename := strings.NewReplacer(`"`, "", `\`, "", "\r", "", "\n", "").Replace(filename)
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, safeFilename))
	w.Header().Set("Content-Length", strconv.FormatInt(stat.Size(), 10))
	w.WriteHeader(http.StatusOK)
	_, err = io.Copy(w, f)
	return err
}

// runPandocExport runs Pandoc for non-PDF formats.
func (h *exportHandler) runPandocExport(ctx context.Context, inputFile, outputFile, toFmt string) error {
	args := []string{
		"-f", pandocInputFmt,
		"-t", toFmt,
		"--standalone",
		"--highlight-style", "zenburn",
		"-o", outputFile,
		inputFile,
	}
	stderr := stderrBufPool.Get().(*bytes.Buffer)
	stderr.Reset()
	defer stderrBufPool.Put(stderr)
	cmd := exec.CommandContext(ctx, h.cfg.PandocBinary, args...)
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%w — %s", err, strings.TrimSpace(stderr.String()))
	}
	return nil
}

// runPDFExport converts Markdown to PDF via a two-step pipeline:
//
//  1. Pandoc converts Markdown → self-contained HTML5 (print.css embedded
//     inline via --embed-resources, syntax highlighted via zenburn).
//  2. If custom margins are requested, a <style> override is injected into
//     the HTML before passing to WeasyPrint.
//  3. WeasyPrint renders the standalone HTML → PDF.
//
// No filename/title metadata is injected; any title in the output comes
// exclusively from the document's own content (YAML frontmatter or headings).
func (h *exportHandler) runPDFExport(ctx context.Context, inputFile, htmlFile, outputFile string, margins pdfMargins) error {
	// Step 1: Pandoc → self-contained HTML
	pandocArgs := []string{
		"-f", pandocInputFmt,
		"-t", "html5",
		"--standalone",
		"--embed-resources",
		"--mathml",
		"--highlight-style", "zenburn",
		"--css", "/app/pandoc/print.css",
		"-o", htmlFile,
		inputFile,
	}
	buf1 := stderrBufPool.Get().(*bytes.Buffer)
	buf1.Reset()
	defer stderrBufPool.Put(buf1)
	cmd1 := exec.CommandContext(ctx, h.cfg.PandocBinary, pandocArgs...)
	cmd1.Stderr = buf1
	if err := cmd1.Run(); err != nil {
		return fmt.Errorf("pandoc html stage: %w — %s", err, strings.TrimSpace(buf1.String()))
	}

	// Step 2 (optional): inject margin overrides into the HTML
	if margins != marginPresets["standard"] {
		htmlBytes, err := os.ReadFile(htmlFile)
		if err != nil {
			return fmt.Errorf("read html: %w", err)
		}
		override := marginOverrideCSS(margins)
		// Inject right before </head>
		modified := strings.Replace(string(htmlBytes), "</head>", override+"\n</head>", 1)
		if err := os.WriteFile(htmlFile, []byte(modified), 0600); err != nil {
			return fmt.Errorf("write margin override: %w", err)
		}
	}

	// Step 3: WeasyPrint → PDF
	buf2 := stderrBufPool.Get().(*bytes.Buffer)
	buf2.Reset()
	defer stderrBufPool.Put(buf2)
	cmd2 := exec.CommandContext(ctx, h.cfg.WeasyprintBinary, htmlFile, outputFile)
	cmd2.Stderr = buf2
	if err := cmd2.Run(); err != nil {
		return fmt.Errorf("weasyprint stage: %w — %s", err, strings.TrimSpace(buf2.String()))
	}
	return nil
}

// GET /api/export/formats  — list available formats
func (h *exportHandler) listFormats(w http.ResponseWriter, r *http.Request) {
	formats := make([]string, 0, len(pandocFormats))
	for k := range pandocFormats {
		formats = append(formats, k)
	}
	writeJSON(w, http.StatusOK, map[string]any{"formats": formats})
}

// POST /api/export/raw/{format} — export raw markdown content without saving
func (h *exportHandler) exportRaw(w http.ResponseWriter, r *http.Request) {
	format := strings.ToLower(chi.URLParam(r, "format"))

	fmtInfo, ok := pandocFormats[format]
	if !ok {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("unsupported format: %s", format))
		return
	}

	var body struct {
		Content string `json:"content"`
		Name    string `json:"name"`
	}
	if err := decodeJSON(r, &body); err != nil || body.Content == "" {
		writeError(w, http.StatusBadRequest, "content is required")
		return
	}
	if body.Name == "" {
		body.Name = "document"
	}

	tmpDir, err := os.MkdirTemp("", "md-export-raw-*")
	if err != nil {
		slog.Error("create tmpdir", "error", err)
		writeError(w, http.StatusInternalServerError, "export failed")
		return
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			slog.Warn("cleanup tmp dir failed", "path", tmpDir, "error", err)
		}
	}()

	inputFile := filepath.Join(tmpDir, "input.md")
	outputFile := filepath.Join(tmpDir, "output"+fmtInfo.ext)

	rawContent := preprocessMarkdown(body.Content)
	if format == "pdf" {
		rawContent = preprocessPageBreaks(rawContent)
	}

	if err := os.WriteFile(inputFile, []byte(rawContent), 0600); err != nil {
		writeError(w, http.StatusInternalServerError, "export failed")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 90*time.Second)
	defer cancel()

	if format == "pdf" {
		margins := parseMargins(r)
		htmlFile := filepath.Join(tmpDir, "output.html")
		if err := h.runPDFExport(ctx, inputFile, htmlFile, outputFile, margins); err != nil {
			slog.Error("pdf export raw failed", "error", err)
			writeError(w, http.StatusInternalServerError, "export conversion failed")
			return
		}
	} else {
		if err := h.runPandocExport(ctx, inputFile, outputFile, fmtInfo.to); err != nil {
			slog.Error("pandoc export raw failed", "format", format, "error", err)
			writeError(w, http.StatusInternalServerError, "export conversion failed")
			return
		}
	}

	slug := strings.TrimSuffix(strings.ReplaceAll(strings.ToLower(body.Name), " ", "-"), ".md")
	if err := streamFile(w, outputFile, fmtInfo.contentType, slug+fmtInfo.ext); err != nil {
		slog.Warn("write export response failed", "error", err)
	}
}
