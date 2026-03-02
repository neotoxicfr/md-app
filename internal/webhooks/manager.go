package webhooks

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Event types dispatched by the system.
const (
	EventFileCreated = "file.created"
	EventFileUpdated = "file.updated"
	EventFileDeleted = "file.deleted"
)

// Webhook represents a registered webhook endpoint.
type Webhook struct {
	ID        string    `json:"id"`
	URL       string    `json:"url"`
	Events    []string  `json:"events"` // e.g. ["file.created","file.updated"]
	Secret    string    `json:"secret"` // used for HMAC-SHA256 signing
	Active    bool      `json:"active"`
	CreatedAt time.Time `json:"created_at"`
}

// Manager handles webhook storage, lifecycle and async dispatch.
type Manager struct {
	mu         sync.RWMutex
	webhooks   []Webhook
	configPath string
	client     *http.Client
}

// New creates a Manager, loading existing webhooks from configPath.
// configPath is typically {storagePath}/.webhooks.json.
func New(configPath string) *Manager {
	m := &Manager{
		configPath: configPath,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
	if err := m.load(); err != nil {
		slog.Warn("webhooks: could not load config, starting empty",
			"path", configPath,
			"error", err,
		)
	}
	return m
}

// ---- persistence ----

func (m *Manager) load() error {
	data, err := os.ReadFile(m.configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // no config yet
		}
		return err
	}
	var hooks []Webhook
	if err := json.Unmarshal(data, &hooks); err != nil {
		return fmt.Errorf("parse webhooks config: %w", err)
	}
	m.webhooks = hooks
	return nil
}

func (m *Manager) save() error {
	data, err := json.MarshalIndent(m.webhooks, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(m.configPath, data, 0644)
}

// ---- CRUD ----

// List returns all registered webhooks.
func (m *Manager) List() []Webhook {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]Webhook, len(m.webhooks))
	copy(out, m.webhooks)
	return out
}

// Create registers a new webhook endpoint.
func (m *Manager) Create(url string, events []string, secret string, active bool) (Webhook, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	w := Webhook{
		ID:        uuid.New().String(),
		URL:       url,
		Events:    events,
		Secret:    secret,
		Active:    active,
		CreatedAt: time.Now().UTC(),
	}
	m.webhooks = append(m.webhooks, w)
	if err := m.save(); err != nil {
		// Roll back
		m.webhooks = m.webhooks[:len(m.webhooks)-1]
		return Webhook{}, fmt.Errorf("save config: %w", err)
	}
	return w, nil
}

// Update modifies an existing webhook by ID.
func (m *Manager) Update(id, url string, events []string, secret string, active bool) (Webhook, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i := range m.webhooks {
		if m.webhooks[i].ID == id {
			m.webhooks[i].URL = url
			m.webhooks[i].Events = events
			m.webhooks[i].Secret = secret
			m.webhooks[i].Active = active
			if err := m.save(); err != nil {
				return Webhook{}, fmt.Errorf("save config: %w", err)
			}
			return m.webhooks[i], nil
		}
	}
	return Webhook{}, fmt.Errorf("webhook %s not found", id)
}

// Delete removes a webhook by ID.
func (m *Manager) Delete(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i := range m.webhooks {
		if m.webhooks[i].ID == id {
			m.webhooks = append(m.webhooks[:i], m.webhooks[i+1:]...)
			return m.save()
		}
	}
	return fmt.Errorf("webhook %s not found", id)
}

// ---- dispatch ----

// Dispatch sends the event to all active webhooks that subscribe to it.
// Delivery happens asynchronously in goroutines.
func (m *Manager) Dispatch(event string, payload any) {
	m.mu.RLock()
	var targets []Webhook
	for _, w := range m.webhooks {
		if !w.Active {
			continue
		}
		for _, e := range w.Events {
			if e == event {
				targets = append(targets, w)
				break
			}
		}
	}
	m.mu.RUnlock()

	if len(targets) == 0 {
		return
	}

	body, err := json.Marshal(map[string]any{
		"event":     event,
		"payload":   payload,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
	if err != nil {
		slog.Error("webhooks: marshal payload", "error", err)
		return
	}

	for _, w := range targets {
		go m.deliver(w, body)
	}
}

// deliver sends a single webhook request with up to 3 retries and
// exponential backoff (2s, 4s, 8s).
func (m *Manager) deliver(w Webhook, body []byte) {
	const maxRetries = 3

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(math.Pow(2, float64(attempt))) * time.Second
			time.Sleep(backoff)
		}

		req, err := http.NewRequest(http.MethodPost, w.URL, bytes.NewReader(body))
		if err != nil {
			slog.Error("webhooks: create request", "url", w.URL, "error", err)
			return // non-retryable
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "MD-Webhook/1.0")

		// HMAC-SHA256 signature
		if w.Secret != "" {
			sig := signPayload(body, w.Secret)
			req.Header.Set("X-Webhook-Signature", "sha256="+sig)
		}

		resp, err := m.client.Do(req)
		if err != nil {
			slog.Warn("webhooks: delivery failed",
				"url", w.URL,
				"attempt", attempt+1,
				"error", err,
			)
			continue
		}
		resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			slog.Info("webhooks: delivered",
				"webhook_id", w.ID,
				"url", w.URL,
				"status", resp.StatusCode,
			)
			return
		}

		slog.Warn("webhooks: non-2xx response",
			"url", w.URL,
			"status", resp.StatusCode,
			"attempt", attempt+1,
		)
	}

	slog.Error("webhooks: all retries exhausted",
		"webhook_id", w.ID,
		"url", w.URL,
	)
}

// signPayload computes HMAC-SHA256(body, secret) and returns the hex digest.
func signPayload(payload []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	return hex.EncodeToString(mac.Sum(nil))
}
