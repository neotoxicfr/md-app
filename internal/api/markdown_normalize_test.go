package api

import (
	"strings"
	"testing"
)

func TestPreprocessMarkdown_InlineHeadings(t *testing.T) {
	in := "Intro paragraph. ## 4. Go-To-Market\nMore text"
	out := preprocessMarkdown(in)

	if !strings.Contains(out, "\n\n## 4. Go-To-Market") {
		t.Fatalf("inline heading not normalized:\n%s", out)
	}
}

func TestPreprocessMarkdown_UnicodeBullets(t *testing.T) {
	in := "• Parent item\n  ◦ Child A\n  ◦ Child B\nText • Another top item"
	out := preprocessMarkdown(in)

	if !strings.Contains(out, "- Parent item") {
		t.Fatalf("top bullet not normalized:\n%s", out)
	}
	if !strings.Contains(out, "  - Child A") || !strings.Contains(out, "  - Child B") {
		t.Fatalf("sub-bullets not normalized:\n%s", out)
	}
	if !strings.Contains(out, "\n- Another top item") {
		t.Fatalf("inline bullet not normalized:\n%s", out)
	}
}

func TestPreprocessMarkdown_CodeFenceUntouched(t *testing.T) {
	in := "```md\n## not-a-heading\n• not-a-list\n```\nOutside ## real-heading"
	out := preprocessMarkdown(in)

	if !strings.Contains(out, "```md\n## not-a-heading\n• not-a-list\n```") {
		t.Fatalf("code fence content was altered:\n%s", out)
	}
	if !strings.Contains(out, "\n\n## real-heading") {
		t.Fatalf("outside heading was not normalized:\n%s", out)
	}
}

func TestRenderMarkdown_AppliesNormalization(t *testing.T) {
	in := "Text before ## Section\n• Item 1\n• Item 2"
	html, err := renderMarkdown(in)
	if err != nil {
		t.Fatalf("renderMarkdown error: %v", err)
	}

	if !strings.Contains(html, "<h2") {
		t.Fatalf("expected h2 in rendered HTML, got:\n%s", html)
	}
	if strings.Contains(html, "## Section") {
		t.Fatalf("raw markdown heading leaked into HTML:\n%s", html)
	}
	if strings.Count(html, "<li>") < 2 {
		t.Fatalf("expected list items in rendered HTML, got:\n%s", html)
	}
}

func TestPreprocessMarkdown_FlattenedInlineTable(t *testing.T) {
	in := "Hypothèses : Churn faible. | Métrique | Année 1 | Année 2 || —|—|— || CA | 125 k€ | 491 k€ || EBITDA | 63 k€ | 101 k€"
	out := preprocessMarkdown(in)

	if !strings.Contains(out, "| Métrique | Année 1 | Année 2 |") {
		t.Fatalf("table header not normalized:\n%s", out)
	}
	if !strings.Contains(out, "| --- | --- | --- |") {
		t.Fatalf("table separator not normalized:\n%s", out)
	}
	if !strings.Contains(out, "| CA | 125 k€ | 491 k€ |") {
		t.Fatalf("table row CA missing:\n%s", out)
	}
	if !strings.Contains(out, "| EBITDA | 63 k€ | 101 k€ |") {
		t.Fatalf("table row EBITDA missing:\n%s", out)
	}
}

func TestPreprocessMarkdown_FlattenedInlineTable_ScreenshotPattern(t *testing.T) {
	in := "Hypothèses : Churn B2B très faible (<3%). Les achats de packs augmentent avec la maturité de l’étude (1 pack en Y1, 2 en Y2, 3 en Y3, etc.). | Métrique | Année 1 (2026) | Année 2 (2027) | Année 3 (2028) | Année 4 (2029) | Année 5 (2030) || —|—|—|—|—|— || Parc Clients “Pro” | 50 | 200 | 600 | 1 500 | 3 000 || EBITDA (Résultat Brut) | ~63 k€ | ~101 k€ | ~391 k€ | ~1 391 k€ | ~4 005 k€"
	out := preprocessMarkdown(in)

	if !strings.Contains(out, "| Métrique | Année 1 (2026) | Année 2 (2027) | Année 3 (2028) | Année 4 (2029) | Année 5 (2030) |") {
		t.Fatalf("header row not reconstructed:\n%s", out)
	}
	if !strings.Contains(out, "| --- | --- | --- | --- | --- | --- |") {
		t.Fatalf("separator row not reconstructed:\n%s", out)
	}
	if !strings.Contains(out, "| Parc Clients “Pro” | 50 | 200 | 600 | 1 500 | 3 000 |") {
		t.Fatalf("data row not reconstructed:\n%s", out)
	}
}
