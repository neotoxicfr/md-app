package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"md/internal/webhooks"
)

// webhookHandler exposes CRUD endpoints for webhook management.
type webhookHandler struct {
	mgr *webhooks.Manager
}

func newWebhookHandler(mgr *webhooks.Manager) *webhookHandler {
	return &webhookHandler{mgr: mgr}
}

// GET /api/webhooks — list all registered webhooks.
func (h *webhookHandler) list(w http.ResponseWriter, r *http.Request) {
	hooks := h.mgr.List()
	if hooks == nil {
		hooks = []webhooks.Webhook{}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"webhooks": hooks,
		"count":    len(hooks),
	})
}

// POST /api/webhooks — create a new webhook.
func (h *webhookHandler) create(w http.ResponseWriter, r *http.Request) {
	var body struct {
		URL    string   `json:"url"`
		Events []string `json:"events"`
		Secret string   `json:"secret"`
		Active bool     `json:"active"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if body.URL == "" {
		writeError(w, http.StatusBadRequest, "url is required")
		return
	}
	if len(body.Events) == 0 {
		writeError(w, http.StatusBadRequest, "at least one event is required")
		return
	}

	hook, err := h.mgr.Create(body.URL, body.Events, body.Secret, body.Active)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not create webhook")
		return
	}
	writeJSON(w, http.StatusCreated, hook)
}

// PUT /api/webhooks/{id} — update an existing webhook.
func (h *webhookHandler) update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var body struct {
		URL    string   `json:"url"`
		Events []string `json:"events"`
		Secret string   `json:"secret"`
		Active bool     `json:"active"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if body.URL == "" {
		writeError(w, http.StatusBadRequest, "url is required")
		return
	}

	hook, err := h.mgr.Update(id, body.URL, body.Events, body.Secret, body.Active)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, hook)
}

// DELETE /api/webhooks/{id} — remove a webhook.
func (h *webhookHandler) delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.mgr.Delete(id); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
