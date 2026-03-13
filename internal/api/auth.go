package api

import (
	"context"
	"crypto"
	"crypto/hmac"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math/big"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ---- configuration ----

// OIDCConfig holds OpenID Connect configuration.
// If Issuer is empty the entire OIDC layer is disabled (backward-compatible).
type OIDCConfig struct {
	Issuer       string
	ClientID     string
	ClientSecret string
	RedirectURL  string
	Scopes       []string
	SessionKey   string // HMAC key for signing session cookies
}

// LoadOIDCConfig reads OIDC settings from environment variables.
// Returns nil when MD_OIDC_ISSUER is not set.
func LoadOIDCConfig() *OIDCConfig {
	issuer := os.Getenv("MD_OIDC_ISSUER")
	if issuer == "" {
		return nil
	}

	scopes := strings.Split(envFallback("MD_OIDC_SCOPES", "openid,profile,email"), ",")
	for i := range scopes {
		scopes[i] = strings.TrimSpace(scopes[i])
	}

	sessionKey := secretOrEnv("oidc_session_key", "MD_OIDC_SESSION_KEY", "")
	if sessionKey == "" {
		slog.Error("OIDC enabled but MD_OIDC_SESSION_KEY is not set — OIDC disabled for security")
		return nil
	}

	return &OIDCConfig{
		Issuer:       issuer,
		ClientID:     os.Getenv("MD_OIDC_CLIENT_ID"),
		ClientSecret: secretOrEnv("oidc_client_secret", "MD_OIDC_CLIENT_SECRET", ""),
		RedirectURL:  os.Getenv("MD_OIDC_REDIRECT_URL"),
		Scopes:       scopes,
		SessionKey:   sessionKey,
	}
}

// secretOrEnv reads a Docker secret (/run/secrets/<name>) first, then falls back to env var.
func secretOrEnv(secretName, envName, def string) string {
	path := "/run/secrets/" + secretName
	if data, err := os.ReadFile(path); err == nil {
		v := strings.TrimSpace(string(data))
		if v != "" {
			return v
		}
	}
	if v := os.Getenv(envName); v != "" {
		return v
	}
	return def
}

func envFallback(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// ---- user / session types ----

type contextKey string

const userContextKey contextKey = "md_user"

// UserInfo represents an authenticated user from the OIDC ID token.
type UserInfo struct {
	Sub   string `json:"sub"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

// UserFromContext extracts UserInfo stored by the OIDC middleware.
func UserFromContext(ctx context.Context) (UserInfo, bool) {
	u, ok := ctx.Value(userContextKey).(UserInfo)
	return u, ok
}

// sessionPayload is the signed data stored in the session cookie.
type sessionPayload struct {
	User    UserInfo  `json:"user"`
	Expires time.Time `json:"exp"`
	Nonce   string    `json:"nonce"`
}

const (
	sessionCookieName = "md_session"
	sessionDuration   = 24 * time.Hour
)

// ---- OIDC discovery ----

type oidcDiscovery struct {
	Issuer                string `json:"issuer"`
	AuthorizationEndpoint string `json:"authorization_endpoint"`
	TokenEndpoint         string `json:"token_endpoint"`
	JwksURI               string `json:"jwks_uri"`
	UserinfoEndpoint      string `json:"userinfo_endpoint"`
}

// oidcProvider caches discovery metadata and JWKS keys.
type oidcProvider struct {
	mu        sync.RWMutex
	cfg       *OIDCConfig
	disc      *oidcDiscovery
	keys      map[string]*rsa.PublicKey // kid -> key
	fetchedAt time.Time
	client    *http.Client
}

func newOIDCProvider(cfg *OIDCConfig) *oidcProvider {
	return &oidcProvider{
		cfg:    cfg,
		keys:   make(map[string]*rsa.PublicKey),
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

// discover fetches the OpenID Configuration document (cached for 1 h).
func (p *oidcProvider) discover() (*oidcDiscovery, error) {
	p.mu.RLock()
	if p.disc != nil && time.Since(p.fetchedAt) < time.Hour {
		d := p.disc
		p.mu.RUnlock()
		return d, nil
	}
	p.mu.RUnlock()

	url := strings.TrimRight(p.cfg.Issuer, "/") + "/.well-known/openid-configuration"
	resp, err := p.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("oidc discovery: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("oidc discovery: status %d", resp.StatusCode)
	}

	var d oidcDiscovery
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&d); err != nil {
		return nil, fmt.Errorf("oidc discovery decode: %w", err)
	}

	p.mu.Lock()
	// Re-check after acquiring write lock (another goroutine may have fetched)
	if p.disc != nil && time.Since(p.fetchedAt) < time.Hour {
		existing := p.disc
		p.mu.Unlock()
		return existing, nil
	}
	p.disc = &d
	p.fetchedAt = time.Now()
	p.mu.Unlock()

	// Prefetch JWKS
	go func() {
		if err := p.fetchJWKS(d.JwksURI); err != nil {
			slog.Warn("auth: JWKS prefetch failed", "uri", d.JwksURI, "error", err)
		}
	}()

	return &d, nil
}

// ---- JWKS ----

type jwksDoc struct {
	Keys []jwkKey `json:"keys"`
}

type jwkKey struct {
	Kty string `json:"kty"`
	Kid string `json:"kid"`
	N   string `json:"n"`
	E   string `json:"e"`
	Alg string `json:"alg"`
	Use string `json:"use"`
}

func (p *oidcProvider) fetchJWKS(jwksURI string) error {
	resp, err := p.client.Get(jwksURI)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var doc jwksDoc
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&doc); err != nil {
		return err
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	for _, k := range doc.Keys {
		if k.Kty != "RSA" || k.Use != "sig" {
			continue
		}
		pub, err := jwkToRSA(k)
		if err != nil {
			slog.Warn("auth: skip invalid JWK", "kid", k.Kid, "error", err)
			continue
		}
		p.keys[k.Kid] = pub
	}
	return nil
}

func jwkToRSA(k jwkKey) (*rsa.PublicKey, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(k.N)
	if err != nil {
		return nil, fmt.Errorf("decode n: %w", err)
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(k.E)
	if err != nil {
		return nil, fmt.Errorf("decode e: %w", err)
	}
	n := new(big.Int).SetBytes(nBytes)
	e := new(big.Int).SetBytes(eBytes)
	if !e.IsInt64() {
		return nil, errors.New("exponent too large")
	}
	return &rsa.PublicKey{N: n, E: int(e.Int64())}, nil
}

// ---- JWT validation ----

type jwtHeader struct {
	Alg string `json:"alg"`
	Kid string `json:"kid"`
	Typ string `json:"typ"`
}

type jwtClaims struct {
	Sub   string `json:"sub"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Iss   string `json:"iss"`
	Aud   any    `json:"aud"` // string or []string
	Exp   int64  `json:"exp"`
	Iat   int64  `json:"iat"`
	Nonce string `json:"nonce"`
}

func (p *oidcProvider) validateIDToken(rawToken string) (*jwtClaims, error) {
	parts := strings.Split(rawToken, ".")
	if len(parts) != 3 {
		return nil, errors.New("invalid JWT format")
	}

	// Decode header
	hdrBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, fmt.Errorf("decode header: %w", err)
	}
	var hdr jwtHeader
	if err := json.Unmarshal(hdrBytes, &hdr); err != nil {
		return nil, fmt.Errorf("parse header: %w", err)
	}
	if hdr.Alg != "RS256" {
		return nil, fmt.Errorf("unsupported algorithm: %s", hdr.Alg)
	}

	// Look up signing key
	p.mu.RLock()
	key, ok := p.keys[hdr.Kid]
	p.mu.RUnlock()
	if !ok {
		// Try refreshing JWKS once
		if disc, err := p.discover(); err == nil {
			if err := p.fetchJWKS(disc.JwksURI); err == nil {
				p.mu.RLock()
				key, ok = p.keys[hdr.Kid]
				p.mu.RUnlock()
			}
		}
		if !ok {
			return nil, fmt.Errorf("unknown signing key: %s", hdr.Kid)
		}
	}

	// Verify signature: RS256 = RSASSA-PKCS1-v1_5 using SHA-256
	signedContent := parts[0] + "." + parts[1]
	sigBytes, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return nil, fmt.Errorf("decode signature: %w", err)
	}
	hash := sha256.Sum256([]byte(signedContent))
	if err := rsa.VerifyPKCS1v15(key, crypto.SHA256, hash[:], sigBytes); err != nil {
		return nil, fmt.Errorf("signature verification failed: %w", err)
	}

	// Decode claims
	claimsBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("decode claims: %w", err)
	}
	var claims jwtClaims
	if err := json.Unmarshal(claimsBytes, &claims); err != nil {
		return nil, fmt.Errorf("parse claims: %w", err)
	}

	// Validate standard claims
	now := time.Now().Unix()
	if claims.Exp > 0 && now > claims.Exp {
		return nil, errors.New("token expired")
	}
	if claims.Iss != p.cfg.Issuer {
		return nil, fmt.Errorf("issuer mismatch: got %q, want %q", claims.Iss, p.cfg.Issuer)
	}
	if !audienceContains(claims.Aud, p.cfg.ClientID) {
		return nil, errors.New("audience mismatch")
	}

	return &claims, nil
}

func audienceContains(aud any, clientID string) bool {
	switch v := aud.(type) {
	case string:
		return v == clientID
	case []any:
		for _, a := range v {
			if s, ok := a.(string); ok && s == clientID {
				return true
			}
		}
	}
	return false
}

// ---- session cookie helpers ----

func (p *oidcProvider) createSession(user UserInfo) (string, error) {
	payload := sessionPayload{
		User:    user,
		Expires: time.Now().Add(sessionDuration),
		Nonce:   uuid.New().String()[:8],
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	encoded := base64.RawURLEncoding.EncodeToString(data)
	sig := hmacSign(encoded, p.cfg.SessionKey)
	return encoded + "." + sig, nil
}

func (p *oidcProvider) validateSession(token string) (*sessionPayload, error) {
	parts := strings.SplitN(token, ".", 2)
	if len(parts) != 2 {
		return nil, errors.New("invalid session format")
	}
	expected := hmacSign(parts[0], p.cfg.SessionKey)
	if !hmac.Equal([]byte(parts[1]), []byte(expected)) {
		return nil, errors.New("invalid session signature")
	}
	data, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, err
	}
	var payload sessionPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, err
	}
	if time.Now().After(payload.Expires) {
		return nil, errors.New("session expired")
	}
	return &payload, nil
}

func hmacSign(data, key string) string {
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write([]byte(data))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

// ---- auth handler ----

// authHandler provides OIDC login/callback/me/logout endpoints.
type authHandler struct {
	provider *oidcProvider
	states   sync.Map // CSRF state -> expiry
	done     chan struct{}
}

func newAuthHandler(cfg *OIDCConfig) *authHandler {
	h := &authHandler{
		provider: newOIDCProvider(cfg),
		done:     make(chan struct{}),
	}
	go h.cleanupStates()
	return h
}

// Shutdown stops the state cleanup goroutine.
func (h *authHandler) Shutdown() {
	close(h.done)
}

func (h *authHandler) cleanupStates() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-h.done:
			return
		case <-ticker.C:
			now := time.Now()
			h.states.Range(func(key, value any) bool {
				if exp, ok := value.(time.Time); ok && now.After(exp) {
					h.states.Delete(key)
				}
				return true
			})
		}
	}
}

// GET /api/auth/login — redirect the user to the OIDC authorization endpoint.
func (h *authHandler) login(w http.ResponseWriter, r *http.Request) {
	disc, err := h.provider.discover()
	if err != nil {
		slog.Error("auth: discovery failed", "error", err)
		writeError(w, http.StatusInternalServerError, "OIDC discovery failed")
		return
	}

	state := uuid.New().String()
	h.states.Store(state, time.Now().Add(10*time.Minute))

	params := url.Values{
		"response_type": {"code"},
		"client_id":     {h.provider.cfg.ClientID},
		"redirect_uri":  {h.provider.cfg.RedirectURL},
		"scope":         {strings.Join(h.provider.cfg.Scopes, " ")},
		"state":         {state},
	}

	target := disc.AuthorizationEndpoint + "?" + params.Encode()
	http.Redirect(w, r, target, http.StatusFound)
}

// GET /api/auth/callback — handle OIDC authorization code callback.
func (h *authHandler) callback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")

	if code == "" || state == "" {
		writeError(w, http.StatusBadRequest, "missing code or state")
		return
	}

	// Validate CSRF state.
	val, ok := h.states.LoadAndDelete(state)
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid state parameter")
		return
	}
	if exp, ok := val.(time.Time); ok && time.Now().After(exp) {
		writeError(w, http.StatusBadRequest, "state expired")
		return
	}

	disc, err := h.provider.discover()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "OIDC discovery failed")
		return
	}

	// Exchange code for tokens.
	tokenResp, err := h.exchangeCode(disc.TokenEndpoint, code)
	if err != nil {
		slog.Error("auth: code exchange failed", "error", err)
		writeError(w, http.StatusInternalServerError, "code exchange failed")
		return
	}

	// Validate the ID token.
	claims, err := h.provider.validateIDToken(tokenResp.IDToken)
	if err != nil {
		slog.Error("auth: ID token validation failed", "error", err)
		writeError(w, http.StatusUnauthorized, "invalid ID token")
		return
	}

	user := UserInfo{
		Sub:   claims.Sub,
		Name:  claims.Name,
		Email: claims.Email,
	}

	// Create session cookie.
	sessionToken, err := h.provider.createSession(user)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "session creation failed")
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    sessionToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(sessionDuration.Seconds()),
	})

	// Redirect to app root after login.
	http.Redirect(w, r, "/", http.StatusFound)
}

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	IDToken      string `json:"id_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
}

func (h *authHandler) exchangeCode(tokenEndpoint, code string) (*tokenResponse, error) {
	data := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {h.provider.cfg.RedirectURL},
		"client_id":     {h.provider.cfg.ClientID},
		"client_secret": {h.provider.cfg.ClientSecret},
	}

	resp, err := h.provider.client.PostForm(tokenEndpoint, data)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token endpoint returned %d: %s", resp.StatusCode, string(body))
	}

	var tr tokenResponse
	if err := json.Unmarshal(body, &tr); err != nil {
		return nil, err
	}
	return &tr, nil
}

// GET /api/auth/me — return the current authenticated user.
func (h *authHandler) me(w http.ResponseWriter, r *http.Request) {
	user, ok := UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}
	writeJSON(w, http.StatusOK, user)
}

// GET /api/auth/logout — clear the session cookie.
func (h *authHandler) logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1, // delete
	})
	writeJSON(w, http.StatusOK, map[string]string{"message": "logged out"})
}

// ---- OIDC middleware ----

// OIDCMiddleware returns a middleware that validates sessions when OIDC is enabled.
// If cfg is nil, it returns a no-op middleware (all requests pass through).
func OIDCMiddleware(cfg *OIDCConfig) func(http.Handler) http.Handler {
	if cfg == nil {
		// Auth disabled — pass through.
		return func(next http.Handler) http.Handler { return next }
	}

	provider := newOIDCProvider(cfg)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Allow auth endpoints without a session.
			if strings.HasPrefix(r.URL.Path, "/api/auth/") {
				next.ServeHTTP(w, r)
				return
			}

			// Check session cookie.
			cookie, err := r.Cookie(sessionCookieName)
			if err != nil {
				writeError(w, http.StatusUnauthorized, "authentication required")
				return
			}

			session, err := provider.validateSession(cookie.Value)
			if err != nil {
				slog.Debug("auth: invalid session", "error", err)
				writeError(w, http.StatusUnauthorized, "invalid or expired session")
				return
			}

			// Inject user into context.
			ctx := context.WithValue(r.Context(), userContextKey, session.User)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
