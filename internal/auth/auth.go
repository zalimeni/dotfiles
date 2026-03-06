// Package auth implements Microsoft Graph API authentication using the
// OAuth2 device code flow. It handles token acquisition, caching, and
// transparent refresh.
package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	// DefaultTenantID uses the "common" multi-tenant endpoint.
	DefaultTenantID = "common"

	// DefaultScope requests access to SharePoint files and sites.
	DefaultScope = "Files.Read.All Sites.Read.All offline_access"

	deviceCodePath = "/oauth2/v2.0/devicecode"
	tokenPath      = "/oauth2/v2.0/token"

	// maxJSONResponseBytes is the maximum size of JSON API responses we
	// will read into memory. This guards against a misbehaving server
	// sending an unbounded response that exhausts memory.
	maxJSONResponseBytes = 1 << 20 // 1 MB

	// slowDownIncrement is the interval increase (in seconds) applied
	// when the server returns a "slow_down" error per RFC 8628 §3.5.
	slowDownIncrement = 5 * time.Second
)

// errSlowDown signals that the authorization server requested the client
// to increase its polling interval (RFC 8628 §3.5).
var errSlowDown = errors.New("slow_down")

// Config holds the parameters needed to perform device code authentication.
type Config struct {
	// ClientID is the Azure AD application (client) ID.
	ClientID string

	// TenantID is the Azure AD tenant ID. Defaults to "common".
	TenantID string

	// Scope is the space-separated list of OAuth2 scopes.
	// Defaults to DefaultScope.
	Scope string

	// TokenPath overrides the default token cache file location.
	// Defaults to ~/.config/sp2md/token.json.
	TokenPath string

	// AuthorityHost overrides the base URL for Azure AD endpoints.
	// Defaults to "https://login.microsoftonline.com". Useful for testing.
	AuthorityHost string

	// HTTPClient overrides the default HTTP client. Useful for testing.
	HTTPClient *http.Client

	// PromptFunc is called with the user-facing device code message.
	// Defaults to printing to stderr.
	PromptFunc func(message string)
}

func (c *Config) tenantID() string {
	if c.TenantID != "" {
		return c.TenantID
	}
	return DefaultTenantID
}

func (c *Config) scope() string {
	if c.Scope != "" {
		return c.Scope
	}
	return DefaultScope
}

func (c *Config) authorityHost() string {
	if c.AuthorityHost != "" {
		return c.AuthorityHost
	}
	return "https://login.microsoftonline.com"
}

func (c *Config) tokenPath() (string, error) {
	if c.TokenPath != "" {
		return c.TokenPath, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("determining home directory: %w", err)
	}
	return filepath.Join(home, ".config", "sp2md", "token.json"), nil
}

func (c *Config) httpClient() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	return &http.Client{Timeout: 30 * time.Second}
}

func (c *Config) prompt(message string) {
	if c.PromptFunc != nil {
		c.PromptFunc(message)
		return
	}
	fmt.Fprintln(os.Stderr, message)
}

// deviceCodeResponse represents the Azure AD device code endpoint response.
type deviceCodeResponse struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
	Message         string `json:"message"`
}

// Token represents a cached OAuth2 token.
type Token struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
	Scope        string    `json:"scope"`
}

// Valid reports whether the access token is present and not expired.
func (t *Token) Valid() bool {
	return t != nil && t.AccessToken != "" && time.Now().Before(t.ExpiresAt)
}

// tokenResponse is the raw JSON from the /token endpoint.
type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	Scope        string `json:"scope"`
	Error        string `json:"error"`
	ErrorDesc    string `json:"error_description"`
}

// Authenticate obtains a valid access token using the device code flow.
// It first attempts to load a cached token and refresh it if expired.
// If no cached token exists or refresh fails, it initiates a new device
// code flow.
func Authenticate(ctx context.Context, cfg *Config) (*Token, error) {
	if cfg.ClientID == "" {
		return nil, errors.New("client ID is required; set --client-id or SP2MD_CLIENT_ID")
	}

	// Try cached token first.
	cached, err := LoadToken(cfg)
	if err == nil && cached != nil {
		if cached.Valid() {
			return cached, nil
		}
		// Attempt refresh.
		if cached.RefreshToken != "" {
			refreshed, refreshErr := refreshToken(ctx, cfg, cached.RefreshToken)
			if refreshErr == nil {
				if saveErr := SaveToken(cfg, refreshed); saveErr != nil {
					return nil, fmt.Errorf("saving refreshed token: %w", saveErr)
				}
				return refreshed, nil
			}
			// Refresh failed — fall through to device code flow.
		}
	}

	// Initiate device code flow.
	token, err := deviceCodeFlow(ctx, cfg)
	if err != nil {
		return nil, err
	}

	if saveErr := SaveToken(cfg, token); saveErr != nil {
		return nil, fmt.Errorf("saving token: %w", saveErr)
	}
	return token, nil
}

// LoadToken reads a cached token from disk.
func LoadToken(cfg *Config) (*Token, error) {
	p, err := cfg.tokenPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return nil, fmt.Errorf("reading token cache: %w", err)
	}
	var tok Token
	if err := json.Unmarshal(data, &tok); err != nil {
		return nil, fmt.Errorf("parsing token cache: %w", err)
	}
	return &tok, nil
}

// SaveToken writes a token to the cache file.
func SaveToken(cfg *Config, tok *Token) error {
	p, err := cfg.tokenPath()
	if err != nil {
		return err
	}
	dir := filepath.Dir(p)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("creating token cache directory: %w", err)
	}
	data, err := json.MarshalIndent(tok, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling token: %w", err)
	}
	if err := os.WriteFile(p, data, 0600); err != nil {
		return fmt.Errorf("writing token cache: %w", err)
	}
	return nil
}

func deviceCodeFlow(ctx context.Context, cfg *Config) (*Token, error) {
	dc, err := requestDeviceCode(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("requesting device code: %w", err)
	}

	cfg.prompt(dc.Message)

	return pollForToken(ctx, cfg, dc)
}

func requestDeviceCode(ctx context.Context, cfg *Config) (*deviceCodeResponse, error) {
	endpoint := cfg.authorityHost() + "/" + cfg.tenantID() + deviceCodePath

	form := url.Values{
		"client_id": {cfg.ClientID},
		"scope":     {cfg.scope()},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := cfg.httpClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("sending request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxJSONResponseBytes))
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("device code request failed (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var dc deviceCodeResponse
	if err := json.Unmarshal(body, &dc); err != nil {
		return nil, fmt.Errorf("parsing device code response: %w", err)
	}
	return &dc, nil
}

func pollForToken(ctx context.Context, cfg *Config, dc *deviceCodeResponse) (*Token, error) {
	endpoint := cfg.authorityHost() + "/" + cfg.tenantID() + tokenPath

	interval := time.Duration(dc.Interval) * time.Second
	if interval < 1*time.Second {
		interval = 5 * time.Second
	}

	deadline := time.Now().Add(time.Duration(dc.ExpiresIn) * time.Second)

	form := url.Values{
		"client_id":   {cfg.ClientID},
		"grant_type":  {"urn:ietf:params:oauth:grant-type:device_code"},
		"device_code": {dc.DeviceCode},
	}

	for {
		if time.Now().After(deadline) {
			return nil, errors.New("device code expired; please try again")
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(interval):
		}

		tok, err := requestToken(ctx, cfg, endpoint, form)
		if errors.Is(err, errSlowDown) {
			interval += slowDownIncrement
			continue
		}
		if err != nil {
			return nil, err
		}
		if tok != nil {
			return tok, nil
		}
		// tok == nil means "authorization_pending" — keep polling.
	}
}

func requestToken(ctx context.Context, cfg *Config, endpoint string, form url.Values) (*Token, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("creating token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := cfg.httpClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("sending token request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxJSONResponseBytes))
	if err != nil {
		return nil, fmt.Errorf("reading token response: %w", err)
	}

	var tr tokenResponse
	if err := json.Unmarshal(body, &tr); err != nil {
		return nil, fmt.Errorf("parsing token response: %w", err)
	}

	switch tr.Error {
	case "":
		// Success.
		return &Token{
			AccessToken:  tr.AccessToken,
			RefreshToken: tr.RefreshToken,
			ExpiresAt:    time.Now().Add(time.Duration(tr.ExpiresIn) * time.Second),
			Scope:        tr.Scope,
		}, nil
	case "authorization_pending":
		return nil, nil
	case "slow_down":
		// Server asked us to back off; caller must increase the interval.
		return nil, errSlowDown
	case "expired_token":
		return nil, errors.New("device code expired; please try again")
	default:
		desc := tr.ErrorDesc
		if desc == "" {
			desc = tr.Error
		}
		return nil, fmt.Errorf("authentication failed: %s", desc)
	}
}

func refreshToken(ctx context.Context, cfg *Config, refresh string) (*Token, error) {
	endpoint := cfg.authorityHost() + "/" + cfg.tenantID() + tokenPath

	form := url.Values{
		"client_id":     {cfg.ClientID},
		"grant_type":    {"refresh_token"},
		"refresh_token": {refresh},
		"scope":         {cfg.scope()},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("creating refresh request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := cfg.httpClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("sending refresh request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxJSONResponseBytes))
	if err != nil {
		return nil, fmt.Errorf("reading refresh response: %w", err)
	}

	var tr tokenResponse
	if err := json.Unmarshal(body, &tr); err != nil {
		return nil, fmt.Errorf("parsing refresh response: %w", err)
	}

	if tr.Error != "" {
		desc := tr.ErrorDesc
		if desc == "" {
			desc = tr.Error
		}
		return nil, fmt.Errorf("token refresh failed: %s", desc)
	}

	return &Token{
		AccessToken:  tr.AccessToken,
		RefreshToken: tr.RefreshToken,
		ExpiresAt:    time.Now().Add(time.Duration(tr.ExpiresIn) * time.Second),
		Scope:        tr.Scope,
	}, nil
}
