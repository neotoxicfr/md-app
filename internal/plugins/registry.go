package plugins

import (
	"fmt"
	"math"
	"regexp"
	"strings"
	"unicode/utf8"
)

// Plugin defines the interface for markdown processing plugins.
type Plugin interface {
	// Name returns a unique identifier for the plugin.
	Name() string
	// Description returns a human-readable summary of what the plugin does.
	Description() string
	// ProcessMarkdown is a pre-processor that runs before markdown→HTML rendering.
	ProcessMarkdown(content string) string
	// ProcessHTML is a post-processor that runs after markdown→HTML rendering.
	ProcessHTML(html string) string
}

// PluginInfo provides metadata about a registered plugin.
type PluginInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// Registry manages a pipeline of ordered plugins.
type Registry struct {
	plugins []Plugin
}

// NewRegistry creates a new plugin registry pre-loaded with built-in plugins.
func NewRegistry() *Registry {
	r := &Registry{}
	r.Register(&tocPlugin{})
	r.Register(&wordCountPlugin{})
	r.Register(&readingTimePlugin{})
	r.Register(&autoLinkPlugin{})
	return r
}

// Register appends a plugin to the processing pipeline.
func (r *Registry) Register(p Plugin) {
	r.plugins = append(r.plugins, p)
}

// List returns metadata for all registered plugins.
func (r *Registry) List() []PluginInfo {
	infos := make([]PluginInfo, len(r.plugins))
	for i, p := range r.plugins {
		infos[i] = PluginInfo{Name: p.Name(), Description: p.Description()}
	}
	return infos
}

// PreProcess runs all pre-processors in registration order.
func (r *Registry) PreProcess(content string) string {
	for _, p := range r.plugins {
		content = p.ProcessMarkdown(content)
	}
	return content
}

// PostProcess runs all post-processors in registration order.
func (r *Registry) PostProcess(html string) string {
	for _, p := range r.plugins {
		html = p.ProcessHTML(html)
	}
	return html
}

// ===========================================================================
// Built-in plugin: Table of Contents
// ===========================================================================

type tocPlugin struct{}

func (p *tocPlugin) Name() string        { return "toc" }
func (p *tocPlugin) Description() string { return "Generates a table of contents from headings" }

// ProcessMarkdown replaces a [TOC] marker with a generated table of contents.
func (p *tocPlugin) ProcessMarkdown(content string) string {
	if !strings.Contains(content, "[TOC]") {
		return content
	}

	var toc strings.Builder
	toc.WriteString("\n")

	inCodeBlock := false
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)

		// Track fenced code blocks to avoid treating # in code as headings.
		if strings.HasPrefix(trimmed, "```") {
			inCodeBlock = !inCodeBlock
			continue
		}
		if inCodeBlock {
			continue
		}
		if !strings.HasPrefix(trimmed, "#") {
			continue
		}

		level := 0
		for _, ch := range trimmed {
			if ch == '#' {
				level++
			} else {
				break
			}
		}

		title := strings.TrimSpace(strings.TrimLeft(trimmed, "#"))
		if title == "" {
			continue
		}

		anchor := headingToAnchor(title)
		indent := strings.Repeat("  ", level-1)
		fmt.Fprintf(&toc, "%s- [%s](#%s)\n", indent, title, anchor)
	}

	toc.WriteString("\n")
	return strings.Replace(content, "[TOC]", toc.String(), 1)
}

func (p *tocPlugin) ProcessHTML(html string) string { return html }

var nonAlphaNumRegex = regexp.MustCompile(`[^a-z0-9\-]`)

func headingToAnchor(title string) string {
	s := strings.ToLower(title)
	s = strings.ReplaceAll(s, " ", "-")
	s = nonAlphaNumRegex.ReplaceAllString(s, "")
	return strings.Trim(s, "-")
}

// ===========================================================================
// Built-in plugin: Word Count
// ===========================================================================

type wordCountPlugin struct{}

func (p *wordCountPlugin) Name() string { return "word-count" }
func (p *wordCountPlugin) Description() string {
	return "Adds word and character count metadata as a hidden HTML element"
}

func (p *wordCountPlugin) ProcessMarkdown(content string) string { return content }

func (p *wordCountPlugin) ProcessHTML(html string) string {
	plain := stripTags(html)
	words := len(strings.Fields(plain))
	chars := utf8.RuneCountInString(plain)

	meta := fmt.Sprintf(
		`<div class="md-meta" data-words="%d" data-chars="%d" style="display:none;"></div>`,
		words, chars,
	)
	return html + "\n" + meta
}

// ===========================================================================
// Built-in plugin: Reading Time
// ===========================================================================

type readingTimePlugin struct{}

func (p *readingTimePlugin) Name() string { return "reading-time" }
func (p *readingTimePlugin) Description() string {
	return "Estimates reading time (200 wpm average) as a hidden HTML element"
}

func (p *readingTimePlugin) ProcessMarkdown(content string) string { return content }

func (p *readingTimePlugin) ProcessHTML(html string) string {
	plain := stripTags(html)
	words := len(strings.Fields(plain))

	minutes := int(math.Ceil(float64(words) / 200.0))
	if minutes < 1 {
		minutes = 1
	}

	meta := fmt.Sprintf(
		`<div class="md-reading-time" data-minutes="%d" style="display:none;"></div>`,
		minutes,
	)
	return html + "\n" + meta
}

// ===========================================================================
// Built-in plugin: Auto-Link Detector
// ===========================================================================

type autoLinkPlugin struct{}

func (p *autoLinkPlugin) Name() string { return "auto-link" }
func (p *autoLinkPlugin) Description() string {
	return "Wraps bare URLs in angle brackets so they become clickable links"
}

var bareURLRegex = regexp.MustCompile(`(?m)(^|[\s(])((https?://)[^\s)<>]+)`)

// ProcessMarkdown wraps bare URLs in <> for GFM autolink. Skips code blocks
// and lines that already contain markdown link syntax.
func (p *autoLinkPlugin) ProcessMarkdown(content string) string {
	lines := strings.Split(content, "\n")
	var result strings.Builder
	inCodeBlock := false

	for i, line := range lines {
		if i > 0 {
			result.WriteString("\n")
		}

		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "```") {
			inCodeBlock = !inCodeBlock
			result.WriteString(line)
			continue
		}
		if inCodeBlock || strings.HasPrefix(trimmed, "    ") || strings.Contains(line, "](") {
			result.WriteString(line)
			continue
		}

		processed := bareURLRegex.ReplaceAllStringFunc(line, func(match string) string {
			prefix := ""
			url := match
			if len(match) > 0 && (match[0] == ' ' || match[0] == '\t' || match[0] == '(') {
				prefix = string(match[0])
				url = match[1:]
			}
			return prefix + "<" + url + ">"
		})
		result.WriteString(processed)
	}
	return result.String()
}

func (p *autoLinkPlugin) ProcessHTML(html string) string { return html }

// ---- helpers ----

var htmlTagRegex = regexp.MustCompile(`<[^>]*>`)

// stripTags removes HTML tags and returns plain text.
func stripTags(html string) string {
	return strings.TrimSpace(htmlTagRegex.ReplaceAllString(html, ""))
}
