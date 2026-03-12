package api

import (
	"regexp"
	"strings"
)

var reInlineHeading = regexp.MustCompile(`\s+(#{1,6}\s+)`)
var reInlineBullet = regexp.MustCompile(`\s+•\s+`)
var reInlineSubBullet = regexp.MustCompile(`\s+◦\s+`)
var reIndentedHeading = regexp.MustCompile(`^\s{4,}(#{1,6}\s+)`)
var reTableSep = regexp.MustCompile(`^[\s|:\-]+$`)

func normalizeInlineTableLine(line string) ([]string, bool) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || !strings.Contains(trimmed, "||") || strings.Count(trimmed, "|") < 4 {
		return nil, false
	}

	line = strings.ReplaceAll(line, "—", "-")
	line = strings.ReplaceAll(line, "–", "-")

	out := make([]string, 0, 8)
	hadPrefix := false
	firstPipe := strings.Index(line, "|")
	if firstPipe > 0 {
		prefix := strings.TrimSpace(line[:firstPipe])
		if prefix != "" {
			out = append(out, prefix)
			hadPrefix = true
		}
		line = line[firstPipe:]
	}

	if hadPrefix {
		out = append(out, "")
	}

	tableRows := 0
	for _, chunk := range strings.Split(line, "||") {
		row := strings.TrimSpace(chunk)
		if row == "" {
			continue
		}
		if strings.Count(row, "|") < 2 {
			out = append(out, row)
			continue
		}

		if !strings.HasPrefix(row, "|") {
			row = "| " + row
		}
		if !strings.HasSuffix(row, "|") {
			row = row + " |"
		}

		inner := strings.TrimSpace(strings.Trim(row, "|"))
		if reTableSep.MatchString(inner) {
			cells := strings.Split(strings.Trim(row, "|"), "|")
			for i, c := range cells {
				c = strings.TrimSpace(c)
				left := strings.HasPrefix(c, ":")
				right := strings.HasSuffix(c, ":")
				norm := "---"
				if left {
					norm = ":" + norm
				}
				if right {
					norm += ":"
				}
				cells[i] = norm
			}
			row = "| " + strings.Join(cells, " | ") + " |"
		}

		out = append(out, row)
		tableRows++
	}

	if tableRows < 2 {
		return nil, false
	}
	out = append(out, "")
	return out, true
}

func preprocessMarkdown(content string) string {
	if content == "" {
		return content
	}

	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.ReplaceAll(content, "\r", "\n")
	content = strings.ReplaceAll(content, "\u00a0", " ")

	lines := strings.Split(content, "\n")
	out := make([]string, 0, len(lines)+16)
	inFence := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			inFence = !inFence
			out = append(out, line)
			continue
		}

		if inFence {
			out = append(out, line)
			continue
		}

		line = reIndentedHeading.ReplaceAllString(line, "$1")

		trimmed = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(trimmed, "• "):
			line = "- " + strings.TrimPrefix(trimmed, "• ")
		case strings.HasPrefix(trimmed, "◦ "):
			line = "  - " + strings.TrimPrefix(trimmed, "◦ ")
		}

		if !strings.HasPrefix(strings.TrimLeft(line, " \t"), "#") && reInlineHeading.MatchString(line) {
			line = reInlineHeading.ReplaceAllString(line, "\n\n$1")
		}

		if !strings.HasPrefix(strings.TrimLeft(line, " \t"), "-") &&
			!strings.HasPrefix(strings.TrimLeft(line, " \t"), "*") &&
			!strings.HasPrefix(strings.TrimLeft(line, " \t"), "+") &&
			reInlineBullet.MatchString(line) {
			line = reInlineBullet.ReplaceAllString(line, "\n- ")
		}

		if reInlineSubBullet.MatchString(line) {
			line = reInlineSubBullet.ReplaceAllString(line, "\n  - ")
		}

		for _, segment := range strings.Split(line, "\n") {
			if tableLines, ok := normalizeInlineTableLine(segment); ok {
				out = append(out, tableLines...)
			} else {
				out = append(out, segment)
			}
		}
	}

	return strings.Join(out, "\n")
}
