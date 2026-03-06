package sharepoint

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// driveItemJSON builds a mock driveItem response with the given download URL
// and filename.
func driveItemJSON(t *testing.T, downloadURL, name string) []byte {
	t.Helper()
	m := map[string]interface{}{
		"name":                           name,
		"@microsoft.graph.downloadUrl":   downloadURL,
		"id":                             "item-id-123",
		"size":                           12345,
	}
	b, err := json.Marshal(m)
	require.NoError(t, err)
	return b
}

// siteJSON builds a mock site response.
func siteJSON(t *testing.T, siteID string) []byte {
	t.Helper()
	m := map[string]interface{}{
		"id":          siteID,
		"displayName": "Test Site",
		"webUrl":      "https://contoso.sharepoint.com/sites/TestSite",
	}
	b, err := json.Marshal(m)
	require.NoError(t, err)
	return b
}

// newGraphServer creates a test server that simulates Graph API endpoints.
func newGraphServer(t *testing.T, fileContent []byte) *httptest.Server {
	t.Helper()

	mux := http.NewServeMux()

	// Sharing link resolution endpoint.
	mux.HandleFunc("/shares/", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.True(t, strings.HasPrefix(r.Header.Get("Authorization"), "Bearer "))

		// The path should end with /driveItem.
		if !strings.HasSuffix(r.URL.Path, "/driveItem") {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		// Use the server's own URL for the download endpoint.
		_, _ = w.Write(driveItemJSON(t, "http://"+r.Host+"/download/report.docx", "report.docx"))
	})

	// Site resolution endpoint.
	mux.HandleFunc("/sites/", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)

		w.Header().Set("Content-Type", "application/json")

		// If the path contains /drive/root:/, it's a drive item request.
		if strings.Contains(r.URL.Path, "/drive/root:/") {
			_, _ = w.Write(driveItemJSON(t, "http://"+r.Host+"/download/report.docx", "report.docx"))
			return
		}

		// Otherwise it's a site resolution request.
		_, _ = w.Write(siteJSON(t, "contoso.sharepoint.com,site-guid-123,web-guid-456"))
	})

	// Download endpoint.
	mux.HandleFunc("/download/", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.wordprocessingml.document")
		_, _ = w.Write(fileContent)
	})

	return httptest.NewServer(mux)
}

func testClient(t *testing.T, serverURL string) *Client {
	t.Helper()
	return &Client{
		AccessToken: "test-token",
		GraphBase:   serverURL,
		HTTPClient:  &http.Client{Timeout: 5 * time.Second},
		TempDir:     t.TempDir(),
	}
}

func TestDownloadViaSharingLink(t *testing.T) {
	content := []byte("PK\x03\x04fake-docx-content-for-testing")
	server := newGraphServer(t, content)
	defer server.Close()

	client := testClient(t, server.URL)

	ctx := context.Background()
	localPath, err := client.Download(ctx, "https://contoso.sharepoint.com/:w:/s/TestSite/EaBcDeFgHiJ")

	require.NoError(t, err)
	defer func() { _ = os.Remove(localPath) }()

	assert.FileExists(t, localPath)
	assert.Contains(t, filepath.Base(localPath), "report.docx")

	got, err := os.ReadFile(localPath)
	require.NoError(t, err)
	assert.Equal(t, content, got)
}

func TestDownloadViaDirectURL(t *testing.T) {
	content := []byte("PK\x03\x04direct-doc-content")
	server := newGraphServer(t, content)
	defer server.Close()

	client := testClient(t, server.URL)

	ctx := context.Background()
	localPath, err := client.Download(ctx, "https://contoso.sharepoint.com/sites/TestSite/Shared Documents/report.docx")

	require.NoError(t, err)
	defer func() { _ = os.Remove(localPath) }()

	assert.FileExists(t, localPath)

	got, err := os.ReadFile(localPath)
	require.NoError(t, err)
	assert.Equal(t, content, got)
}

func TestDownloadPermissionDenied(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/shares/", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":{"code":"accessDenied","message":"Access denied"}}`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client := testClient(t, server.URL)

	_, err := client.Download(context.Background(), "https://contoso.sharepoint.com/:w:/s/Private/EaBcDeFgHiJ")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrPermissionDenied)
}

func TestDownloadNotFound(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/shares/", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":{"code":"itemNotFound","message":"Item not found"}}`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client := testClient(t, server.URL)

	_, err := client.Download(context.Background(), "https://contoso.sharepoint.com/:w:/s/Gone/EaBcDeFgHiJ")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestDownloadMissingToken(t *testing.T) {
	client := &Client{}
	_, err := client.Download(context.Background(), "https://contoso.sharepoint.com/:w:/s/Site/EaBcDeFgHiJ")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "access token is required")
}

func TestDownloadEmptyURL(t *testing.T) {
	client := &Client{AccessToken: "tok"}
	_, err := client.Download(context.Background(), "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "SharePoint URL is required")
}

func TestDownloadStreaming(t *testing.T) {
	// Verify large files are streamed without buffering entirely in memory.
	// We use a 1MB payload as a proxy.
	content := make([]byte, 1024*1024)
	for i := range content {
		content[i] = byte(i % 256)
	}

	server := newGraphServer(t, content)
	defer server.Close()

	client := testClient(t, server.URL)

	ctx := context.Background()
	localPath, err := client.Download(ctx, "https://contoso.sharepoint.com/:w:/s/Big/EaBcDeFgHiJ")
	require.NoError(t, err)
	defer func() { _ = os.Remove(localPath) }()

	info, err := os.Stat(localPath)
	require.NoError(t, err)
	assert.Equal(t, int64(len(content)), info.Size())
}

// --- URL Parsing Tests ---

func TestParseURL_SharingLinks(t *testing.T) {
	tests := []struct {
		name string
		url  string
	}{
		{
			name: "word sharing link",
			url:  "https://contoso.sharepoint.com/:w:/s/TeamSite/EaBcDeFgHiJkLmNoPqRsTu",
		},
		{
			name: "excel sharing link",
			url:  "https://contoso.sharepoint.com/:x:/s/TeamSite/EaBcDeFgHiJkLmNoPqRsTu",
		},
		{
			name: "powerpoint sharing link",
			url:  "https://contoso.sharepoint.com/:p:/s/TeamSite/EaBcDeFgHiJkLmNoPqRsTu",
		},
		{
			name: "sharing link with /r/ prefix",
			url:  "https://contoso.sharepoint.com/:w:/r/sites/TeamSite/_layouts/15/Doc.aspx?sourcedoc=abc",
		},
		{
			name: "personal OneDrive link",
			url:  "https://contoso-my.sharepoint.com/personal/user_contoso_com/_layouts/15/onedrive.aspx",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ref, err := ParseURL(tt.url)
			require.NoError(t, err)
			assert.Equal(t, RefTypeSharingLink, ref.Type)
			assert.Equal(t, tt.url, ref.SharingURL)
		})
	}
}

func TestParseURL_DirectDocumentURLs(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		siteHost string
		sitePath string
		itemPath string
	}{
		{
			name:     "shared documents",
			url:      "https://contoso.sharepoint.com/sites/TeamSite/Shared Documents/report.docx",
			siteHost: "contoso.sharepoint.com",
			sitePath: "sites/TeamSite",
			itemPath: "Shared Documents/report.docx",
		},
		{
			name:     "nested folder",
			url:      "https://contoso.sharepoint.com/sites/Project/Shared Documents/2024/Q1/budget.docx",
			siteHost: "contoso.sharepoint.com",
			sitePath: "sites/Project",
			itemPath: "Shared Documents/2024/Q1/budget.docx",
		},
		{
			name:     "with query parameters",
			url:      "https://contoso.sharepoint.com/sites/Team/Shared Documents/file.docx?csf=1&web=1",
			siteHost: "contoso.sharepoint.com",
			sitePath: "sites/Team",
			itemPath: "Shared Documents/file.docx",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ref, err := ParseURL(tt.url)
			require.NoError(t, err)
			assert.Equal(t, RefTypeDriveItem, ref.Type)
			assert.Equal(t, tt.siteHost, ref.SiteHost)
			assert.Equal(t, tt.sitePath, ref.SitePath)
			assert.Equal(t, tt.itemPath, ref.ItemPath)
		})
	}
}

func TestParseURL_NonSharePointURL(t *testing.T) {
	_, err := ParseURL("https://example.com/doc.docx")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not a SharePoint URL")
}

func TestParseURL_EmptyURL(t *testing.T) {
	_, err := ParseURL("")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "URL is required")
}

func TestParseURL_FallbackToSharingLink(t *testing.T) {
	// A SharePoint URL that doesn't match specific patterns should fall back
	// to sharing link resolution.
	ref, err := ParseURL("https://contoso.sharepoint.com/sites/Team/SitePages/Home.aspx")
	require.NoError(t, err)
	assert.Equal(t, RefTypeSharingLink, ref.Type)
}

// --- Encoding Tests ---

func TestEncodeSharingURL(t *testing.T) {
	// Verify the encoding matches the Microsoft Graph API spec.
	tests := []struct {
		name     string
		input    string
		wantPfx  string
	}{
		{
			name:    "basic URL",
			input:   "https://contoso.sharepoint.com/:w:/s/Site/EaBc",
			wantPfx: "u!",
		},
		{
			name:    "URL with special characters",
			input:   "https://contoso.sharepoint.com/sites/Team/Shared Documents/report.docx",
			wantPfx: "u!",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoded := EncodeSharingURL(tt.input)
			assert.True(t, strings.HasPrefix(encoded, tt.wantPfx))
			// Should not contain base64 padding.
			assert.False(t, strings.Contains(encoded, "="))
			// Should use URL-safe alphabet.
			assert.False(t, strings.Contains(encoded[2:], "+"))
			assert.False(t, strings.Contains(encoded[2:], "/"))
		})
	}
}

// --- Helper Function Tests ---

func TestExtractDriveItemInfo(t *testing.T) {
	t.Run("valid response", func(t *testing.T) {
		body := driveItemJSON(t, "https://download.example.com/file.docx", "report.docx")
		dlURL, name, err := extractDriveItemInfo(body)
		require.NoError(t, err)
		assert.Equal(t, "https://download.example.com/file.docx", dlURL)
		assert.Equal(t, "report.docx", name)
	})

	t.Run("missing download URL", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{"name": "file.docx"})
		_, _, err := extractDriveItemInfo(body)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing download URL")
	})

	t.Run("invalid JSON", func(t *testing.T) {
		_, _, err := extractDriveItemInfo([]byte("not json"))
		require.Error(t, err)
	})
}

func TestExtractJSONString(t *testing.T) {
	body := []byte(`{"id":"site-123","name":"Test"}`)

	val, err := extractJSONString(body, "id")
	require.NoError(t, err)
	assert.Equal(t, "site-123", val)

	_, err = extractJSONString(body, "missing")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"report.docx", "report.docx"},
		{"my file (1).docx", "my_file__1_.docx"},
		{"../../etc/passwd", ".._.._etc_passwd"},
		{"normal-file_v2.docx", "normal-file_v2.docx"},
		{"..", "file"},
		{"", "file"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.want, sanitizeFilename(tt.input))
		})
	}
}

func TestCheckHTTPError(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       []byte
		wantErr    error
		wantNil    bool
		wantMsg    string
	}{
		{
			name:       "200 returns nil",
			statusCode: http.StatusOK,
			body:       nil,
			wantNil:    true,
		},
		{
			name:       "204 returns nil",
			statusCode: http.StatusNoContent,
			body:       nil,
			wantNil:    true,
		},
		{
			name:       "401 returns ErrPermissionDenied",
			statusCode: http.StatusUnauthorized,
			body:       []byte(`{"error":"unauthorized"}`),
			wantErr:    ErrPermissionDenied,
		},
		{
			name:       "403 returns ErrPermissionDenied",
			statusCode: http.StatusForbidden,
			body:       []byte(`{"error":"forbidden"}`),
			wantErr:    ErrPermissionDenied,
		},
		{
			name:       "404 returns ErrNotFound",
			statusCode: http.StatusNotFound,
			body:       []byte(`{"error":"not found"}`),
			wantErr:    ErrNotFound,
		},
		{
			name:       "500 returns error with body content",
			statusCode: http.StatusInternalServerError,
			body:       []byte(`{"error":"internal server error"}`),
			wantMsg:    `graph API error (HTTP 500): {"error":"internal server error"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkHTTPError(tt.statusCode, tt.body)
			if tt.wantNil {
				assert.NoError(t, err)
				return
			}
			require.Error(t, err)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			}
			if tt.wantMsg != "" {
				assert.EqualError(t, err, tt.wantMsg)
			}
		})
	}
}

func TestDownloadContextCancellation(t *testing.T) {
	server := newGraphServer(t, []byte("content"))
	defer server.Close()

	client := testClient(t, server.URL)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.

	_, err := client.Download(ctx, "https://contoso.sharepoint.com/:w:/s/Site/EaBc")
	require.Error(t, err)
}
