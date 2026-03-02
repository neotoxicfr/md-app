package api

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	"md/internal/config"
)

// loggingMiddleware logs each request.
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rw, r)
		slog.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rw.status,
			"duration_ms", time.Since(start).Milliseconds(),
			"remote", r.RemoteAddr,
		)
	})
}

// apiKeyMiddleware enforces an optional API key (X-API-Key header or ?api_key query param).
func apiKeyMiddleware(cfg *config.Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if cfg.APIKey == "" {
				next.ServeHTTP(w, r)
				return
			}
			key := r.Header.Get("X-API-Key")
			if key == "" {
				key = r.URL.Query().Get("api_key")
			}
			if !strings.EqualFold(key, cfg.APIKey) {
				writeError(w, http.StatusUnauthorized, "invalid or missing API key")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// securityHeaders adds hardened HTTP security headers.
func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "SAMEORIGIN")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		w.Header().Set("Content-Security-Policy",
			"default-src 'self'; "+
				"script-src 'self' 'unsafe-inline' 'unsafe-eval' https://cdn.jsdelivr.net; "+
				"style-src 'self' 'unsafe-inline' https://fonts.googleapis.com https://cdn.jsdelivr.net; "+
				"img-src 'self' data: blob: https:; "+
				"font-src 'self' data: https://fonts.gstatic.com https://cdn.jsdelivr.net; "+
				"connect-src 'self' wss: ws:;")
		next.ServeHTTP(w, r)
	})
}

// responseWriter wraps http.ResponseWriter to capture the status code.
type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(status int) {
	rw.status = status
	rw.ResponseWriter.WriteHeader(status)
}

// Flush implements http.Flusher so SSE streaming works through the logging middleware.
func (rw *responseWriter) Flush() {
	if f, ok := rw.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// Unwrap allows http.ResponseController to access the underlying ResponseWriter.
func (rw *responseWriter) Unwrap() http.ResponseWriter {
	return rw.ResponseWriter
}
