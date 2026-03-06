package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newMockServer creates a test server that simulates Azure AD device code
// and token endpoints. pollsBeforeSuccess controls how many
// "authorization_pending" responses the token endpoint returns before
// issuing a successful token.
func newMockServer(t *testing.T, pollsBeforeSuccess int) *httptest.Server {
	t.Helper()
	var polls atomic.Int32

	mux := http.NewServeMux()

	mux.HandleFunc("/test-tenant/oauth2/v2.0/devicecode", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "application/x-www-form-urlencoded", r.Header.Get("Content-Type"))

		err := r.ParseForm()
		require.NoError(t, err)
		assert.Equal(t, "test-client-id", r.FormValue("client_id"))

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(deviceCodeResponse{
			DeviceCode:      "test-device-code",
			UserCode:        "ABCD-1234",
			VerificationURI: "https://microsoft.com/devicelogin",
			ExpiresIn:       300,
			Interval:        1,
			Message:         "Visit https://microsoft.com/devicelogin and enter code ABCD-1234",
		})
	})

	mux.HandleFunc("/test-tenant/oauth2/v2.0/token", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)

		err := r.ParseForm()
		require.NoError(t, err)

		grantType := r.FormValue("grant_type")
		w.Header().Set("Content-Type", "application/json")

		switch grantType {
		case "urn:ietf:params:oauth:grant-type:device_code":
			count := int(polls.Add(1))
			if count <= pollsBeforeSuccess {
				_ = json.NewEncoder(w).Encode(tokenResponse{
					Error: "authorization_pending",
				})
				return
			}
			_ = json.NewEncoder(w).Encode(tokenResponse{
				AccessToken:  "test-access-token",
				RefreshToken: "test-refresh-token",
				ExpiresIn:    3600,
				Scope:        DefaultScope,
			})

		case "refresh_token":
			assert.Equal(t, "test-refresh-token", r.FormValue("refresh_token"))
			_ = json.NewEncoder(w).Encode(tokenResponse{
				AccessToken:  "refreshed-access-token",
				RefreshToken: "new-refresh-token",
				ExpiresIn:    3600,
				Scope:        DefaultScope,
			})

		default:
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(tokenResponse{
				Error:     "unsupported_grant_type",
				ErrorDesc: "unsupported grant type: " + grantType,
			})
		}
	})

	return httptest.NewServer(mux)
}

func testConfig(t *testing.T, serverURL string) *Config {
	t.Helper()
	tokenDir := t.TempDir()
	var prompted []string
	return &Config{
		ClientID:      "test-client-id",
		TenantID:      "test-tenant",
		TokenPath:     filepath.Join(tokenDir, "token.json"),
		AuthorityHost: serverURL,
		HTTPClient:    &http.Client{Timeout: 5 * time.Second},
		PromptFunc: func(message string) {
			prompted = append(prompted, message)
		},
	}
}

func TestDeviceCodeFlow(t *testing.T) {
	server := newMockServer(t, 2) // 2 pending polls, then success
	defer server.Close()

	cfg := testConfig(t, server.URL)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tok, err := Authenticate(ctx, cfg)
	require.NoError(t, err)
	assert.Equal(t, "test-access-token", tok.AccessToken)
	assert.Equal(t, "test-refresh-token", tok.RefreshToken)
	assert.True(t, tok.Valid())
}

func TestTokenCaching(t *testing.T) {
	server := newMockServer(t, 0)
	defer server.Close()

	cfg := testConfig(t, server.URL)

	ctx := context.Background()

	// First call should go through device code flow.
	tok1, err := Authenticate(ctx, cfg)
	require.NoError(t, err)
	assert.Equal(t, "test-access-token", tok1.AccessToken)

	// Second call should return cached token without hitting server.
	tok2, err := Authenticate(ctx, cfg)
	require.NoError(t, err)
	assert.Equal(t, "test-access-token", tok2.AccessToken)
}

func TestTokenRefresh(t *testing.T) {
	server := newMockServer(t, 0)
	defer server.Close()

	cfg := testConfig(t, server.URL)

	// Write an expired token with a refresh token.
	expired := &Token{
		AccessToken:  "expired-access-token",
		RefreshToken: "test-refresh-token",
		ExpiresAt:    time.Now().Add(-1 * time.Hour),
		Scope:        DefaultScope,
	}
	require.NoError(t, SaveToken(cfg, expired))

	ctx := context.Background()
	tok, err := Authenticate(ctx, cfg)
	require.NoError(t, err)
	assert.Equal(t, "refreshed-access-token", tok.AccessToken)
	assert.Equal(t, "new-refresh-token", tok.RefreshToken)
	assert.True(t, tok.Valid())
}

func TestTokenSaveAndLoad(t *testing.T) {
	cfg := &Config{
		TokenPath: filepath.Join(t.TempDir(), "token.json"),
	}

	original := &Token{
		AccessToken:  "save-test-token",
		RefreshToken: "save-test-refresh",
		ExpiresAt:    time.Now().Add(1 * time.Hour).Truncate(time.Second),
		Scope:        "Files.Read.All",
	}

	require.NoError(t, SaveToken(cfg, original))

	loaded, err := LoadToken(cfg)
	require.NoError(t, err)
	assert.Equal(t, original.AccessToken, loaded.AccessToken)
	assert.Equal(t, original.RefreshToken, loaded.RefreshToken)
	assert.Equal(t, original.Scope, loaded.Scope)
	// Time comparison with tolerance for JSON round-trip.
	assert.WithinDuration(t, original.ExpiresAt, loaded.ExpiresAt, time.Second)
}

func TestTokenFilePermissions(t *testing.T) {
	dir := t.TempDir()
	cfg := &Config{
		TokenPath: filepath.Join(dir, "subdir", "token.json"),
	}

	tok := &Token{
		AccessToken: "perm-test",
		ExpiresAt:   time.Now().Add(1 * time.Hour),
	}

	require.NoError(t, SaveToken(cfg, tok))

	info, err := os.Stat(cfg.TokenPath)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0600), info.Mode().Perm(),
		"token file should have 0600 permissions")
}

func TestMissingClientID(t *testing.T) {
	cfg := &Config{}
	_, err := Authenticate(context.Background(), cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "client ID is required")
}

func TestTokenValid(t *testing.T) {
	tests := []struct {
		name  string
		token *Token
		want  bool
	}{
		{
			name:  "nil token",
			token: nil,
			want:  false,
		},
		{
			name:  "empty access token",
			token: &Token{ExpiresAt: time.Now().Add(1 * time.Hour)},
			want:  false,
		},
		{
			name:  "expired token",
			token: &Token{AccessToken: "tok", ExpiresAt: time.Now().Add(-1 * time.Minute)},
			want:  false,
		},
		{
			name:  "valid token",
			token: &Token{AccessToken: "tok", ExpiresAt: time.Now().Add(1 * time.Hour)},
			want:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.token.Valid())
		})
	}
}

func TestLoadTokenNotFound(t *testing.T) {
	cfg := &Config{
		TokenPath: filepath.Join(t.TempDir(), "nonexistent", "token.json"),
	}
	_, err := LoadToken(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "reading token cache")
}

func TestAuthFailureError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/test-tenant/oauth2/v2.0/devicecode", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"invalid_client","error_description":"client not found"}`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	cfg := testConfig(t, server.URL)
	_, err := Authenticate(context.Background(), cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "device code request failed")
}

func TestExpiredDeviceCode(t *testing.T) {
	// Server always returns authorization_pending. We set ExpiresIn to 1
	// so the device code expires almost immediately.
	mux := http.NewServeMux()
	mux.HandleFunc("/test-tenant/oauth2/v2.0/devicecode", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(deviceCodeResponse{
			DeviceCode:      "dc",
			UserCode:        "CODE",
			VerificationURI: "https://example.com",
			ExpiresIn:       1,
			Interval:        1,
			Message:         "Enter CODE",
		})
	})
	mux.HandleFunc("/test-tenant/oauth2/v2.0/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(tokenResponse{Error: "authorization_pending"})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	cfg := testConfig(t, server.URL)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := Authenticate(ctx, cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "device code expired")
}
