#!/bin/bash
set -uo pipefail

BASE="https://md.cybergraphe.fr"
PASS=0
FAIL=0

check() {
  local name="$1" 
  local result="$2"
  if [[ "$result" == "PASS" ]]; then
    echo "  ✓ $name"
    PASS=$((PASS + 1))
  else
    echo "  ✗ $name — $result"
    FAIL=$((FAIL + 1))
  fi
}

echo "═══════════════════════════════════"
echo "  MD — E2E Test Suite"
echo "  $BASE"
echo "═══════════════════════════════════"
echo ""

# 1. Health
H=$(curl -sf "$BASE/health" 2>/dev/null || echo "")
[[ "$H" == *'"status":"ok"'* ]] && check "Health endpoint" "PASS" || check "Health endpoint" "FAIL"

# 2. SPA
PAGE=$(curl -sf "$BASE/" 2>/dev/null || echo "")
[[ "$PAGE" == *'id="app"'* ]] && check "SPA HTML loads" "PASS" || check "SPA HTML loads" "FAIL"

# 3. CSS asset
CSS_URL=$(echo "$PAGE" | grep -oP 'href="(/assets/index-[^"]+\.css)"' | head -1 | grep -oP '/assets/[^"]+' || echo "")
if [[ -n "$CSS_URL" ]]; then
  CSS_CODE=$(curl -sf -o /dev/null -w "%{http_code}" "$BASE$CSS_URL" 2>/dev/null || echo "000")
  [[ "$CSS_CODE" == "200" ]] && check "CSS asset ($CSS_URL)" "PASS" || check "CSS asset" "HTTP $CSS_CODE"
else
  check "CSS asset" "URL not found"
fi

# 4. JS asset
JS_URL=$(echo "$PAGE" | grep -oP 'src="(/assets/index-[^"]+\.js)"' | head -1 | grep -oP '/assets/[^"]+' || echo "")
if [[ -n "$JS_URL" ]]; then
  JS_CODE=$(curl -sf -o /dev/null -w "%{http_code}" "$BASE$JS_URL" 2>/dev/null || echo "000")
  [[ "$JS_CODE" == "200" ]] && check "JS asset ($JS_URL)" "PASS" || check "JS asset" "HTTP $JS_CODE"
else
  check "JS asset" "URL not found"
fi

# 5. Create file
CREATE=$(curl -sf -X POST "$BASE/api/files" \
  -H "Content-Type: application/json" \
  -d '{"name":"e2e-test.md","content":"# Hello E2E\n\nBold **test** and `code`."}' 2>/dev/null || echo "")
FILE_ID=$(echo "$CREATE" | python3 -c "import sys,json; print(json.load(sys.stdin)['id'])" 2>/dev/null || echo "")
[[ -n "$FILE_ID" ]] && check "Create file (id=${FILE_ID:0:8}…)" "PASS" || check "Create file" "FAIL: $CREATE"

# 6. List files
LIST=$(curl -sf "$BASE/api/files" 2>/dev/null || echo "")
[[ "$LIST" == *"e2e-test.md"* ]] && check "List files" "PASS" || check "List files" "FAIL"

# 7. Get file
GET=$(curl -sf "$BASE/api/files/$FILE_ID" 2>/dev/null || echo "")
[[ "$GET" == *"Hello E2E"* ]] && check "Get file by ID" "PASS" || check "Get file by ID" "FAIL"

# 8. Update file
UPD=$(curl -sf -X PUT "$BASE/api/files/$FILE_ID" \
  -H "Content-Type: application/json" \
  -d '{"name":"e2e-updated.md","content":"# Updated\nNew version."}' 2>/dev/null || echo "")
[[ "$UPD" == *"e2e-updated"* ]] && check "Update file" "PASS" || check "Update file" "FAIL"

# 9. Render markdown
RENDER=$(curl -sf -X POST "$BASE/api/files/render" \
  -H "Content-Type: application/json" \
  -d '{"content":"**bold** and _italic_"}' 2>/dev/null || echo "")
[[ "$RENDER" == *"<strong>bold</strong>"* ]] && check "Render markdown" "PASS" || check "Render markdown" "FAIL"

# 9b. Render malformed markdown normalization regression
MALFORMED='• Parent item\n  ◦ Child item\nText before ## 4. Section Title\n• Last item'
RENDER_MAL=$(curl -sf -X POST "$BASE/api/files/render" \
  -H "Content-Type: application/json" \
  -d "{\"content\":\"$MALFORMED\"}" 2>/dev/null || echo "")
if [[ "$RENDER_MAL" == *"<h2"* && "$RENDER_MAL" == *"<li>"* && "$RENDER_MAL" != *"## 4. Section Title"* ]]; then
  check "Render malformed markdown normalization" "PASS"
else
  check "Render malformed markdown normalization" "FAIL"
fi

# 10. Export HTML
EXP_HTML=$(curl -sf -o /dev/null -w "%{http_code}" "$BASE/api/files/$FILE_ID/export/html" 2>/dev/null || echo "000")
[[ "$EXP_HTML" == "200" ]] && check "Export HTML" "PASS" || check "Export HTML" "HTTP $EXP_HTML"

# 11. Export PDF
EXP_PDF=$(curl -sf -o /dev/null -w "%{http_code}" -X POST "$BASE/api/files/$FILE_ID/export/pdf" 2>/dev/null || echo "000")
[[ "$EXP_PDF" == "200" ]] && check "Export PDF" "PASS" || check "Export PDF" "HTTP $EXP_PDF"

# 12. Export DOCX
EXP_DOCX=$(curl -sf -o /dev/null -w "%{http_code}" -X POST "$BASE/api/files/$FILE_ID/export/docx" 2>/dev/null || echo "000")
[[ "$EXP_DOCX" == "200" ]] && check "Export DOCX" "PASS" || check "Export DOCX" "HTTP $EXP_DOCX"

# 13. Export EPUB
EXP_EPUB=$(curl -sf -o /dev/null -w "%{http_code}" -X POST "$BASE/api/files/$FILE_ID/export/epub" 2>/dev/null || echo "000")
[[ "$EXP_EPUB" == "200" ]] && check "Export EPUB" "PASS" || check "Export EPUB" "HTTP $EXP_EPUB"

# 14. Security headers
HDRS=$(curl -sfI "$BASE/" 2>/dev/null || echo "")
[[ "$HDRS" == *"x-content-type-options"* || "$HDRS" == *"X-Content-Type-Options"* ]] && check "Security headers" "PASS" || check "Security headers" "FAIL"

# 15. New design tokens in CSS
if [[ -n "$CSS_URL" ]]; then
  CSS_BODY=$(curl -sf "$BASE$CSS_URL" 2>/dev/null || echo "")
  [[ "$CSS_BODY" == *"--accent"* ]] && check "Design tokens (--accent)" "PASS" || check "Design tokens" "FAIL"
  [[ "$CSS_BODY" == *"backdrop-filter"* ]] && check "Glass effect (backdrop-filter)" "PASS" || check "Glass effect" "FAIL"
  [[ "$CSS_BODY" == *"#09090b"* || "$CSS_BODY" == *"09090b"* ]] && check "Dark zinc-950 bg" "PASS" || check "Dark zinc-950 bg" "FAIL"
  [[ "$CSS_BODY" == *"8b5cf6"* ]] && check "Violet accent color" "PASS" || check "Violet accent" "FAIL"
  [[ "$CSS_BODY" == *"orb"* ]] && check "Background orbs animation" "PASS" || check "Background orbs" "FAIL"
fi

# 16. Particles in JS bundle
if [[ -n "$JS_URL" ]]; then
  JS_BODY=$(curl -sf "$BASE$JS_URL" 2>/dev/null || echo "")
  [[ "$JS_BODY" == *"particle"* || "$JS_BODY" == *"Particle"* ]] && check "Particles component in bundle" "PASS" || check "Particles component" "FAIL"
fi

# 16b. Raw export (no save) — DOCX
RAW_DOCX=$(curl -sf -o /dev/null -w "%{http_code}" -X POST "$BASE/api/export/raw/docx" \
  -H "Content-Type: application/json" \
  -d '{"content":"# Raw Export Test\nHello world","name":"raw-test"}' 2>/dev/null || echo "000")
[[ "$RAW_DOCX" == "200" ]] && check "Raw export DOCX (no save)" "PASS" || check "Raw export DOCX" "HTTP $RAW_DOCX"

# 16c. Raw export (no save) — PDF
RAW_PDF=$(curl -sf -o /dev/null -w "%{http_code}" -X POST "$BASE/api/export/raw/pdf" \
  -H "Content-Type: application/json" \
  -d '{"content":"# Raw PDF\nNo file needed","name":"raw-pdf-test"}' 2>/dev/null || echo "000")
[[ "$RAW_PDF" == "200" ]] && check "Raw export PDF (no save)" "PASS" || check "Raw export PDF" "HTTP $RAW_PDF"

# 16e. Raw export malformed markdown — RST (assert normalized heading/list structure)
RAW_RST=$(curl -sf -X POST "$BASE/api/export/raw/rst" \
  -H "Content-Type: application/json" \
  -d "{\"content\":\"$MALFORMED\",\"name\":\"raw-rst-mal\"}" 2>/dev/null || echo "")
if [[ "$RAW_RST" == *"Section Title"* && "$RAW_RST" == *"===="* && "$RAW_RST" == *"-  Parent item"* ]]; then
  check "Raw export malformed markdown (RST)" "PASS"
else
  check "Raw export malformed markdown (RST)" "FAIL"
fi

# 16d. Font picker in bundle
if [[ -n "$JS_URL" ]]; then
  [[ "$JS_BODY" == *"font"* || "$JS_BODY" == *"Font"* ]] && check "Font picker in bundle" "PASS" || check "Font picker" "FAIL"
fi

# 17. SPA fallback
SPA_FALLBACK=$(curl -sf -o /dev/null -w "%{http_code}" "$BASE/some/deep/route" 2>/dev/null || echo "000")
[[ "$SPA_FALLBACK" == "200" ]] && check "SPA fallback routing" "PASS" || check "SPA fallback" "HTTP $SPA_FALLBACK"

# ── New roadmap feature tests ──

# 20. Templates — list
TPL_LIST=$(curl -sf "$BASE/api/templates" 2>/dev/null || echo "")
[[ "$TPL_LIST" == *'"count":8'* ]] && check "Templates list (8 templates)" "PASS" || check "Templates list" "FAIL: $TPL_LIST"

# 21. Templates — get single
TPL_GET=$(curl -sf "$BASE/api/templates/blog-post" 2>/dev/null || echo "")
[[ "$TPL_GET" == *'"id":"blog-post"'* && "$TPL_GET" == *'"content":'* ]] && check "Template get (blog-post)" "PASS" || check "Template get" "FAIL"

# 22. Templates — 404
TPL_404=$(curl -s -o /dev/null -w "%{http_code}" "$BASE/api/templates/nonexistent" 2>/dev/null || echo "000")
[[ "$TPL_404" == "404" ]] && check "Template 404 for unknown" "PASS" || check "Template 404" "HTTP $TPL_404"

# 23. Search endpoint
SEARCH=$(curl -sf "$BASE/api/search?q=test" 2>/dev/null || echo "")
[[ "$SEARCH" == *'"results":'* ]] && check "Search endpoint" "PASS" || check "Search endpoint" "FAIL: $SEARCH"

# 24. Plugins list
PLUG=$(curl -sf "$BASE/api/plugins" 2>/dev/null || echo "")
[[ "$PLUG" == *'"toc"'* && "$PLUG" == *'"word-count"'* && "$PLUG" == *'"reading-time"'* ]] && check "Plugins list (toc, word-count, reading-time)" "PASS" || check "Plugins list" "FAIL"

# 25. Webhooks CRUD
WH_LIST=$(curl -sf "$BASE/api/webhooks" 2>/dev/null || echo "")
[[ "$WH_LIST" == *'"webhooks":'* ]] && check "Webhooks list" "PASS" || check "Webhooks list" "FAIL"

# 26. Webhook create
WH_CREATE=$(curl -sf -X POST "$BASE/api/webhooks" \
  -H "Content-Type: application/json" \
  -d '{"url":"https://example.com/hook","events":["file.created"],"secret":"test123"}' 2>/dev/null || echo "")
WH_ID=$(echo "$WH_CREATE" | python3 -c "import sys,json; print(json.load(sys.stdin)['id'])" 2>/dev/null || echo "")
[[ -n "$WH_ID" ]] && check "Webhook create (id=${WH_ID:0:8}…)" "PASS" || check "Webhook create" "FAIL: $WH_CREATE"

# 27. Webhook delete
if [[ -n "$WH_ID" ]]; then
  WH_DEL=$(curl -sf -o /dev/null -w "%{http_code}" -X DELETE "$BASE/api/webhooks/$WH_ID" 2>/dev/null || echo "000")
  [[ "$WH_DEL" == "200" || "$WH_DEL" == "204" ]] && check "Webhook delete" "PASS" || check "Webhook delete" "HTTP $WH_DEL"
fi

# 28. Export formats list
FMT=$(curl -sf "$BASE/api/export/formats" 2>/dev/null || echo "")
[[ "$FMT" == *'"pdf"'* && "$FMT" == *'"docx"'* && "$FMT" == *'"epub"'* ]] && check "Export formats list" "PASS" || check "Export formats list" "FAIL"

# 29. Version history — create file, update, check versions
VER_CREATE=$(curl -sf -X POST "$BASE/api/files" \
  -H "Content-Type: application/json" \
  -d '{"name":"version-test.md","content":"# V1\nOriginal."}' 2>/dev/null || echo "")
VER_ID=$(echo "$VER_CREATE" | python3 -c "import sys,json; print(json.load(sys.stdin)['id'])" 2>/dev/null || echo "")
if [[ -n "$VER_ID" ]]; then
  # Update to create a version
  curl -sf -X PUT "$BASE/api/files/$VER_ID" \
    -H "Content-Type: application/json" \
    -d '{"name":"version-test.md","content":"# V2\nUpdated."}' >/dev/null 2>&1
  VER_LIST=$(curl -sf "$BASE/api/files/$VER_ID/versions" 2>/dev/null || echo "")
  [[ "$VER_LIST" == *'"versions":'* ]] && check "Version history list" "PASS" || check "Version history list" "FAIL: $VER_LIST"
  # Cleanup
  curl -sf -X DELETE "$BASE/api/files/$VER_ID" >/dev/null 2>&1
else
  check "Version history" "FAIL: could not create test file"
fi

# 30. PWA manifest
MANIFEST_CODE=$(curl -sf -o /dev/null -w "%{http_code}" "$BASE/manifest.json" 2>/dev/null || echo "000")
[[ "$MANIFEST_CODE" == "200" ]] && check "PWA manifest.json" "PASS" || check "PWA manifest.json" "HTTP $MANIFEST_CODE"

# 31. Service worker
SW_CODE=$(curl -sf -o /dev/null -w "%{http_code}" "$BASE/sw.js" 2>/dev/null || echo "000")
[[ "$SW_CODE" == "200" ]] && check "Service worker (sw.js)" "PASS" || check "Service worker" "HTTP $SW_CODE"

# 32. Mermaid in JS bundle
if [[ -n "$JS_URL" ]]; then
  [[ "$JS_BODY" == *"mermaid"* || "$JS_BODY" == *"Mermaid"* ]] && check "Mermaid in bundle" "PASS" || check "Mermaid in bundle" "FAIL"
fi

# 33. KaTeX in bundle
KATEX_CHUNKS=$(curl -sf "$BASE/" 2>/dev/null | grep -oP '/assets/katex[^"]+' || echo "")
if [[ -n "$KATEX_CHUNKS" ]]; then
  check "KaTeX chunk in HTML" "PASS"
elif [[ -n "$JS_URL" && "$JS_BODY" == *"katex"* ]]; then
  check "KaTeX in JS bundle" "PASS"
else
  # Check for katex CSS link in index.html
  KATEX_CSS=$(curl -sf "$BASE/" 2>/dev/null | grep -i "katex" || echo "")
  [[ -n "$KATEX_CSS" ]] && check "KaTeX CSS reference" "PASS" || check "KaTeX in bundle" "FAIL"
fi

# 34. Collaborative editing — SSE endpoint (just check it responds with 200)
COLLAB_CODE=$(curl -s -o /dev/null -w "%{http_code}" --max-time 2 "$BASE/api/files/fake-id/events" 2>/dev/null; true)
[[ "$COLLAB_CODE" == "200" ]] && check "Collab SSE endpoint" "PASS" || check "Collab SSE endpoint" "HTTP $COLLAB_CODE"

# ── End new tests ──

# 18. Delete file
DEL_CODE=$(curl -sf -o /dev/null -w "%{http_code}" -X DELETE "$BASE/api/files/$FILE_ID" 2>/dev/null || echo "000")
[[ "$DEL_CODE" == "200" || "$DEL_CODE" == "204" ]] && check "Delete file" "PASS" || check "Delete file" "HTTP $DEL_CODE"

# 19. Verify deletion
GONE=$(curl -s -o /dev/null -w "%{http_code}" "$BASE/api/files/$FILE_ID" 2>/dev/null || echo "000")
[[ "$GONE" == "404" ]] && check "Verify deletion (404)" "PASS" || check "Verify deletion" "HTTP $GONE"

echo ""
echo "═══════════════════════════════════"
echo "  Results: $PASS passed, $FAIL failed"
echo "═══════════════════════════════════"

[[ $FAIL -eq 0 ]] && exit 0 || exit 1
