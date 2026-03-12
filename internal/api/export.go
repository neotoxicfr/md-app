package api

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"md/internal/config"
	"md/internal/storage"
)

type exportHandler struct {
	store *storage.Storage
	cfg   *config.Config
}

func newExportHandler(store *storage.Storage, cfg *config.Config) *exportHandler {
	return &exportHandler{store: store, cfg: cfg}
}

// pageBreakDiv is the raw HTML injected for page breaks in PDF output.
const pageBreakDiv = `<div class="pagebreak"></div>`

// rePageBreak matches \newpage, \pagebreak, <!-- pagebreak -->, or --- pagebreak ---
var rePageBreak = regexp.MustCompile(`(?m)^\\(?:newpage|pagebreak)\s*$|^<!--\s*pagebreak\s*-->\s*$|^---\s*pagebreak\s*---\s*$`)

// preprocessPageBreaks replaces page-break markers with a raw HTML div
// that WeasyPrint will interpret as a forced page break via CSS.
func preprocessPageBreaks(content string) string {
	return rePageBreak.ReplaceAllString(content, pageBreakDiv)
}

// Supported export formats via Pandoc
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
	defer os.RemoveAll(tmpDir)

	inputFile := filepath.Join(tmpDir, "input.md")
	outputFile := filepath.Join(tmpDir, "output"+fmtInfo.ext)

	// Preprocess page breaks for PDF export
	content := fwc.Content
	if format == "pdf" {
		content = preprocessPageBreaks(content)
	}

	if err := os.WriteFile(inputFile, []byte(content), 0644); err != nil {
		writeError(w, http.StatusInternalServerError, "export failed")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()

	args := []string{
		"-f", "markdown+emoji+smart+autolink_bare_uris+footnotes+task_lists+definition_lists",
		"-t", fmtInfo.to,
		"--standalone",
		"--highlight-style", "zenburn",
		"-o", outputFile,
		inputFile,
	}

	// PDF: two-step pipeline (Pandoc → HTML → WeasyPrint → PDF)
	if format == "pdf" {
		htmlFile := filepath.Join(tmpDir, "output.html")
		if err := h.runPDFExport(ctx, inputFile, htmlFile, outputFile); err != nil {
			slog.Error("pdf export failed",
				"file", fwc.Name,
				"error", err,
			)
			writeError(w, http.StatusInternalServerError, "export conversion failed: "+err.Error())
			return
		}
	} else {
		cmd := exec.CommandContext(ctx, h.cfg.PandocBinary, args...)
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			slog.Error("pandoc export failed",
				"format", format,
				"file", fwc.Name,
				"stderr", stderr.String(),
				"error", err,
			)
			writeError(w, http.StatusInternalServerError, "export conversion failed: "+stderr.String())
			return
		}
	}

	output, err := os.ReadFile(outputFile)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not read export output")
		return
	}

	filename := fwc.Slug + fmtInfo.ext
	w.Header().Set("Content-Type", fmtInfo.contentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(output)))
	w.WriteHeader(http.StatusOK)
	w.Write(output)
}

func (h *exportHandler) buildPDFArgs(inputFile, outputFile, title string) []string {
	// Kept for reference — replaced by runPDFExport (two-step pipeline).
	// Not called directly anymore.
	return nil
}

// runPDFExport converts Markdown to PDF via a two-step pipeline:
//
//  1. Pandoc converts Markdown → self-contained HTML (CSS embedded inline
//     via --embed-resources so WeasyPrint has no external file dependencies).
//  2. WeasyPrint renders the standalone HTML → PDF.
//
// No filename/title metadata is injected; any title in the output comes
// exclusively from the document's own content (YAML frontmatter or headings).
func (h *exportHandler) runPDFExport(ctx context.Context, inputFile, htmlFile, outputFile string) error {
	// Step 1: Pandoc → self-contained HTML
	pandocArgs := []string{
		"-f", "markdown+emoji+smart+autolink_bare_uris+footnotes+task_lists+definition_lists+raw_html",
		"-t", "html5",
		"--standalone",
		"--embed-resources", // inline CSS/images so WeasyPrint needs no external files
		"--highlight-style", "zenburn",
		"--css", "/app/pandoc/print.css",
		"-o", htmlFile,
		inputFile,
	}
	var buf1 bytes.Buffer
	cmd1 := exec.CommandContext(ctx, h.cfg.PandocBinary, pandocArgs...)
	cmd1.Stderr = &buf1
	if err := cmd1.Run(); err != nil {
		return fmt.Errorf("pandoc html stage: %w — %s", err, strings.TrimSpace(buf1.String()))
	}

	// Step 2: WeasyPrint → PDF
	var buf2 bytes.Buffer
	cmd2 := exec.CommandContext(ctx, h.cfg.WeasyprintBinary, htmlFile, outputFile)
	cmd2.Stderr = &buf2
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
	defer os.RemoveAll(tmpDir)

	inputFile := filepath.Join(tmpDir, "input.md")
	outputFile := filepath.Join(tmpDir, "output"+fmtInfo.ext)

	// Preprocess page breaks for PDF export
	rawContent := body.Content
	if format == "pdf" {
		rawContent = preprocessPageBreaks(rawContent)
	}

	if err := os.WriteFile(inputFile, []byte(rawContent), 0644); err != nil {
		writeError(w, http.StatusInternalServerError, "export failed")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()

	args := []string{
		"-f", "markdown+emoji+smart+autolink_bare_uris+footnotes+task_lists+definition_lists",
		"-t", fmtInfo.to,
		"--standalone",
		"--highlight-style", "zenburn",
		"-o", outputFile,
		inputFile,
	}

	// PDF: two-step pipeline (Pandoc → HTML → WeasyPrint → PDF)
	if format == "pdf" {
		htmlFile := filepath.Join(tmpDir, "output.html")
		if err := h.runPDFExport(ctx, inputFile, htmlFile, outputFile); err != nil {
			slog.Error("pdf export raw failed",
				"error", err,
			)
			writeError(w, http.StatusInternalServerError, "export conversion failed: "+err.Error())
			return
		}
	} else {
		cmd := exec.CommandContext(ctx, h.cfg.PandocBinary, args...)
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			slog.Error("pandoc export raw failed",
				"format", format,
				"stderr", stderr.String(),
				"error", err,
			)
			writeError(w, http.StatusInternalServerError, "export conversion failed: "+stderr.String())
			return
		}
	}

	output, err := os.ReadFile(outputFile)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not read export output")
		return
	}

	slug := strings.TrimSuffix(strings.ReplaceAll(strings.ToLower(body.Name), " ", "-"), ".md")
	filename := slug + fmtInfo.ext
	w.Header().Set("Content-Type", fmtInfo.contentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(output)))
	w.WriteHeader(http.StatusOK)
	w.Write(output)
}
