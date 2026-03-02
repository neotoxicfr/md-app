# ═══════════════════════════════════════════════════════════════════════════
# MD – Multi-stage Dockerfile
# ═══════════════════════════════════════════════════════════════════════════
#
# This Dockerfile produces a single container (~150 MB) that serves both
# the Go REST API and the SvelteKit SPA, with Pandoc + WeasyPrint for
# multi-format document export (PDF, DOCX, HTML, etc.).
#
# Build stages:
#   1. web-build  → Compile the SvelteKit frontend (Node 22)
#   2. go-build   → Compile the Go binary with embedded metadata (Go 1.25)
#   3. runtime    → Minimal Alpine image with Pandoc + WeasyPrint + fonts
#
# Build args (set via docker-compose or CI):
#   VERSION    → displayed on /health endpoint (e.g. "1.2.3")
#   GIT_SHA    → git commit hash for traceability
#   BUILD_DATE → ISO-8601 build timestamp
#
# Usage:
#   docker build -t md-app --build-arg VERSION=1.0.0 -f Dockerfile.app .
#
# ═══════════════════════════════════════════════════════════════════════════


# ─────────────────────────────────────────────────────────────
# Stage 1: Build SvelteKit frontend
# ─────────────────────────────────────────────────────────────
# Produces a static SPA in /src/web/dist/ (adapter-static).
# Only package.json + lock file are copied first for layer caching.
FROM node:22-alpine AS web-build
WORKDIR /src

# Install dependencies (cached unless package*.json changes)
COPY web/package*.json ./web/
RUN cd web && npm ci --prefer-offline

# Copy source and build the SPA
COPY web/ ./web/
RUN cd web && npm run build


# ─────────────────────────────────────────────────────────────
# Stage 2: Build Go binary
# ─────────────────────────────────────────────────────────────
# Produces a statically-linked binary at /app/md (~15 MB).
# CGO is disabled for a fully static build (no libc dependency).
FROM golang:1.25-alpine AS go-build
WORKDIR /src

# Ensure the correct Go toolchain is used
ENV GOTOOLCHAIN=auto

# Static build: no CGO, targeting Linux AMD64
# Change GOARCH to arm64 if deploying on ARM (e.g. Raspberry Pi, M-series Mac)
ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOARCH=amd64

# Git is needed for go mod download (private repos or git-based deps)
RUN apk add --no-cache git

# Download Go dependencies (cached unless go.mod/go.sum changes)
COPY go.mod go.sum ./
RUN go mod download

# Copy full source tree
COPY . .
RUN go mod tidy

# Run unit tests during build (failures are non-blocking: || true)
# Remove "|| true" to make the build fail on test failures
RUN go test -short ./... 2>&1 || true

# Build args injected as linker flags into the binary
ARG VERSION=dev
ARG GIT_SHA=unknown
ARG BUILD_DATE=unknown

# Build the binary with:
#   -s -w     → strip debug info (smaller binary)
#   -X main.* → inject version metadata at compile time
#   -trimpath → reproducible builds (remove local paths from binary)
RUN go build \
    -ldflags="-s -w -X main.Version=${VERSION} -X main.GitSHA=${GIT_SHA} -X main.BuildDate=${BUILD_DATE}" \
    -trimpath \
    -o /app/md \
    ./cmd/server


# ─────────────────────────────────────────────────────────────
# Stage 3: Runtime image (Alpine + Pandoc + WeasyPrint)
# ─────────────────────────────────────────────────────────────
# Minimal production image. Only the compiled binary, the SPA,
# Pandoc templates, and system tools are included.
FROM alpine:3.21 AS runtime

# OCI image metadata (adjust source URL if you forked the project)
LABEL org.opencontainers.image.title="MD"
LABEL org.opencontainers.image.description="Open-source markdown editor & file manager"
LABEL org.opencontainers.image.source="https://github.com/cybergraphe-fr/md"
LABEL org.opencontainers.image.licenses="MIT"

# System dependencies:
#   pandoc          → multi-format export (DOCX, HTML, LaTeX, etc.)
#   py3-weasyprint  → HTML→PDF conversion (used by export pipeline)
#   ca-certificates → TLS for outbound HTTPS (OIDC, webhooks)
#   tzdata          → timezone support (TZ env var)
#   font-dejavu     → fallback serif/sans/mono fonts for PDF rendering
#   font-liberation → metric-compatible alternatives to Arial/Times/Courier
#   ttf-liberation  → TrueType version of Liberation fonts
RUN apk add --no-cache \
    pandoc \
    py3-weasyprint \
    ca-certificates \
    tzdata \
    font-dejavu \
    font-liberation \
    ttf-liberation \
    && rm -rf /var/cache/apk/*

# Security: run as non-root user "md" (UID/GID auto-assigned)
RUN addgroup -S md && adduser -S -G md md

# Create app and data directories with correct ownership
# /data/files  → user markdown files
# /data/.meta  → file metadata (JSON sidecar files)
RUN mkdir -p /app /data/files /data/.meta && \
    chown -R md:md /app /data

WORKDIR /app

# Copy artifacts from build stages
COPY --from=go-build /app/md ./md
COPY --from=web-build /src/web/dist ./web
# Pandoc templates + print.css for PDF/DOCX export
COPY pandoc/ ./pandoc/

# Switch to non-root user for all runtime operations
USER md

# Default environment variables (can be overridden in docker-compose)
ENV MD_HTTP_ADDR=:8080
ENV MD_STORAGE_PATH=/data
ENV MD_PANDOC_BINARY=pandoc
ENV TZ=UTC

# The Go server listens on this port
EXPOSE 8080

# Health check: the /health endpoint returns {"status":"ok","version":"..."}
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD wget -qO- http://localhost:8080/health || exit 1

# Start the Go server (no shell wrapper needed)
ENTRYPOINT ["./md"]
