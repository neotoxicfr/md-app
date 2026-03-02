package api

import (
	"encoding/json"
	"net/http"
	"runtime"
	"time"
)

var startTime = time.Now()

type healthResponse struct {
	Status    string `json:"status"`
	Version   string `json:"version"`
	Uptime    string `json:"uptime"`
	GoVersion string `json:"go_version"`
	OS        string `json:"os"`
	Arch      string `json:"arch"`
}

func handleHealth(version string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, healthResponse{
			Status:    "ok",
			Version:   version,
			Uptime:    time.Since(startTime).Round(time.Second).String(),
			GoVersion: runtime.Version(),
			OS:        runtime.GOOS,
			Arch:      runtime.GOARCH,
		})
	}
}

// ---- JSON helpers ----

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
