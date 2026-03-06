// Package sharepoint downloads .docx files from SharePoint Online using
// the Microsoft Graph API. It handles URL parsing, sharing link resolution,
// and streaming downloads to a local temp directory.
package sharepoint

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const (
	// DefaultGraphBase is the Microsoft Graph API base URL.
	DefaultGraphBase = "https://graph.microsoft.com/v1.0"

	defaultHTTPTimeout = 2 * time.Minute
)

// ErrPermissionDenied indicates the caller lacks access to the requested resource.
var ErrPermissionDenied = errors.New("permission denied: insufficient access to the SharePoint resource")

// ErrNotFound indicates the requested resource does not exist.
var ErrNotFound = errors.New("not found: the SharePoint resource does not exist or has been deleted")

// Client downloads documents from SharePoint via the Graph API.
type Client struct {
	// AccessToken is the Bearer token for Graph API requests.
	AccessToken string

	// GraphBase overrides the Graph API base URL. Useful for testing.
	GraphBase string

	// HTTPClient overrides the default HTTP client.
	HTTPClient *http.Client

	// TempDir overrides the directory for downloaded files.
	// Defaults to os.TempDir().
	TempDir string
}

func (c *Client) graphBase() string {
	if c.GraphBase != "" {
		return c.GraphBase
	}
	return DefaultGraphBase
}

func (c *Client) httpClient() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	return &http.Client{Timeout: defaultHTTPTimeout}
}

func (c *Client) tempDir() string {
	if c.TempDir != "" {
		return c.TempDir
	}
	return os.TempDir()
}

// Download fetches a .docx file from SharePoint given a URL and returns the
// local file path where the document was saved. The caller is responsible
// for removing the file when done.
func (c *Client) Download(ctx context.Context, sharePointURL string) (string, error) {
	if c.AccessToken == "" {
		return "", errors.New("access token is required")
	}
	if sharePointURL == "" {
		return "", errors.New("SharePoint URL is required")
	}

	ref, err := ParseURL(sharePointURL)
	if err != nil {
		return "", fmt.Errorf("parsing SharePoint URL: %w", err)
	}

	downloadURL, filename, err := c.resolveDownloadURL(ctx, ref)
	if err != nil {
		return "", err
	}

	localPath, err := c.downloadFile(ctx, downloadURL, filename)
	if err != nil {
		return "", err
	}

	return localPath, nil
}

// resolveDownloadURL determines the Graph API download URL based on the
// parsed reference type.
func (c *Client) resolveDownloadURL(ctx context.Context, ref *ResourceRef) (downloadURL, filename string, err error) {
	switch ref.Type {
	case RefTypeSharingLink:
		return c.resolveSharingLink(ctx, ref.SharingURL)
	case RefTypeDriveItem:
		return c.resolveDriveItem(ctx, ref.SiteHost, ref.SitePath, ref.ItemPath)
	default:
		return "", "", fmt.Errorf("unsupported reference type: %s", ref.Type)
	}
}

// resolveSharingLink uses the /shares endpoint to resolve a sharing URL to
// a download URL.
func (c *Client) resolveSharingLink(ctx context.Context, sharingURL string) (downloadURL, filename string, err error) {
	encoded := EncodeSharingURL(sharingURL)
	endpoint := c.graphBase() + "/shares/" + encoded + "/driveItem"

	body, err := c.graphGet(ctx, endpoint)
	if err != nil {
		return "", "", fmt.Errorf("resolving sharing link: %w", err)
	}

	downloadURL, filename, err = extractDriveItemInfo(body)
	if err != nil {
		return "", "", fmt.Errorf("parsing sharing link response: %w", err)
	}

	return downloadURL, filename, nil
}

// resolveDriveItem resolves a site-relative path to a download URL.
func (c *Client) resolveDriveItem(ctx context.Context, siteHost, sitePath, itemPath string) (downloadURL, filename string, err error) {
	// Step 1: Resolve site ID.
	siteEndpoint := c.graphBase() + "/sites/" + siteHost + ":/" + sitePath
	siteBody, err := c.graphGet(ctx, siteEndpoint)
	if err != nil {
		return "", "", fmt.Errorf("resolving site: %w", err)
	}

	siteID, err := extractJSONString(siteBody, "id")
	if err != nil {
		return "", "", fmt.Errorf("extracting site ID: %w", err)
	}

	// Step 2: Get the drive item by path on the default drive.
	itemEndpoint := c.graphBase() + "/sites/" + siteID + "/drive/root:/" + itemPath
	itemBody, err := c.graphGet(ctx, itemEndpoint)
	if err != nil {
		return "", "", fmt.Errorf("resolving drive item: %w", err)
	}

	downloadURL, filename, err = extractDriveItemInfo(itemBody)
	if err != nil {
		return "", "", fmt.Errorf("parsing drive item response: %w", err)
	}

	return downloadURL, filename, nil
}

// downloadFile streams a file from downloadURL to a temp file on disk.
func (c *Client) downloadFile(ctx context.Context, downloadURL, filename string) (string, error) {
	if filename == "" {
		filename = "document.docx"
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return "", fmt.Errorf("creating download request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.AccessToken)

	resp, err := c.httpClient().Do(req)
	if err != nil {
		return "", fmt.Errorf("downloading file: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("downloading file: %w", checkHTTPError(resp.StatusCode, body))
	}

	// Create temp file with the original filename for clarity.
	dir := filepath.Join(c.tempDir(), "sp2md")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", fmt.Errorf("creating temp directory: %w", err)
	}

	f, err := os.CreateTemp(dir, "dl-*-"+sanitizeFilename(filename))
	if err != nil {
		return "", fmt.Errorf("creating temp file: %w", err)
	}

	localPath := f.Name()

	// Stream to disk — io.Copy uses a 32KB buffer internally, so large
	// files are not loaded entirely into memory.
	if _, err := io.Copy(f, resp.Body); err != nil {
		_ = f.Close()
		_ = os.Remove(localPath)
		return "", fmt.Errorf("writing downloaded file: %w", err)
	}

	if err := f.Close(); err != nil {
		_ = os.Remove(localPath)
		return "", fmt.Errorf("closing temp file: %w", err)
	}

	return localPath, nil
}

// graphGet performs an authenticated GET request to the Graph API and returns
// the response body.
func (c *Client) graphGet(ctx context.Context, endpoint string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.AccessToken)

	resp, err := c.httpClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("sending request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if err := checkHTTPError(resp.StatusCode, body); err != nil {
		return nil, err
	}

	return body, nil
}

// checkHTTPError maps Graph API error status codes to typed errors.
// Callers provide the status code and response body so that this function
// does not consume resp.Body as a side effect.
func checkHTTPError(statusCode int, body []byte) error {
	switch {
	case statusCode >= 200 && statusCode < 300:
		return nil
	case statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden:
		return ErrPermissionDenied
	case statusCode == http.StatusNotFound:
		return ErrNotFound
	default:
		return fmt.Errorf("graph API error (HTTP %d): %s", statusCode, string(body))
	}
}

// EncodeSharingURL encodes a sharing URL for the /shares endpoint using
// the base64url encoding scheme described in the Graph API docs:
// base64-encode the URL, convert +/ to -_, strip trailing =, prepend "u!".
func EncodeSharingURL(sharingURL string) string {
	encoded := base64.StdEncoding.EncodeToString([]byte(sharingURL))
	encoded = strings.NewReplacer("+", "-", "/", "_").Replace(encoded)
	encoded = strings.TrimRight(encoded, "=")
	return "u!" + encoded
}

// extractDriveItemInfo extracts the download URL and filename from a Graph
// API driveItem JSON response.
func extractDriveItemInfo(body []byte) (downloadURL, filename string, err error) {
	var item struct {
		Name                 string `json:"name"`
		MicrosoftGraphDownloadURL string `json:"@microsoft.graph.downloadUrl"`
	}
	if err := json.Unmarshal(body, &item); err != nil {
		return "", "", fmt.Errorf("parsing drive item: %w", err)
	}

	if item.MicrosoftGraphDownloadURL == "" {
		return "", "", errors.New("drive item response missing download URL")
	}

	return item.MicrosoftGraphDownloadURL, item.Name, nil
}

// extractJSONString extracts a top-level string field from JSON.
func extractJSONString(body []byte, key string) (string, error) {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(body, &m); err != nil {
		return "", fmt.Errorf("parsing JSON: %w", err)
	}
	raw, ok := m[key]
	if !ok {
		return "", fmt.Errorf("key %q not found in response", key)
	}
	var val string
	if err := json.Unmarshal(raw, &val); err != nil {
		return "", fmt.Errorf("key %q is not a string: %w", key, err)
	}
	return val, nil
}

// sanitizeFilename removes characters that are unsafe in filenames.
func sanitizeFilename(name string) string {
	re := regexp.MustCompile(`[^a-zA-Z0-9._-]`)
	return re.ReplaceAllString(name, "_")
}

// RefType classifies the kind of SharePoint URL.
type RefType string

const (
	// RefTypeSharingLink is a sharing link (e.g., /:w:/s/... or /personal/...).
	RefTypeSharingLink RefType = "sharing_link"

	// RefTypeDriveItem is a direct site-relative path to a document.
	RefTypeDriveItem RefType = "drive_item"
)

// ResourceRef holds the parsed components of a SharePoint URL.
type ResourceRef struct {
	// Type indicates how to resolve this reference.
	Type RefType

	// SharingURL is the original URL for sharing links.
	SharingURL string

	// SiteHost is the SharePoint site hostname (e.g., "contoso.sharepoint.com").
	SiteHost string

	// SitePath is the site-relative path (e.g., "sites/team-site").
	SitePath string

	// ItemPath is the document path relative to the site's default drive.
	ItemPath string
}

// sharingLinkPattern matches SharePoint sharing links like:
//
//	https://contoso.sharepoint.com/:w:/s/SiteName/EaBC123...
//	https://contoso.sharepoint.com/:w:/r/sites/SiteName/_layouts/...
//	https://contoso-my.sharepoint.com/personal/user_contoso_com/_layouts/...
var sharingLinkPattern = regexp.MustCompile(
	`^https?://[^/]+\.sharepoint\.com/(?::[a-z]:/[a-z]/|personal/)`,
)

// directDocPattern matches direct document URLs like:
//
//	https://contoso.sharepoint.com/sites/SiteName/Shared Documents/file.docx
var directDocPattern = regexp.MustCompile(
	`^https?://([^/]+\.sharepoint\.com)/sites/([^/]+)/(.+\.docx)(?:\?.*)?$`,
)

// ParseURL parses a SharePoint URL into a ResourceRef.
func ParseURL(rawURL string) (*ResourceRef, error) {
	if rawURL == "" {
		return nil, errors.New("URL is required")
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	if !strings.Contains(parsed.Host, "sharepoint.com") {
		return nil, fmt.Errorf("not a SharePoint URL: %s", parsed.Host)
	}

	// Check for sharing links first — they are the most common.
	if sharingLinkPattern.MatchString(rawURL) {
		return &ResourceRef{
			Type:       RefTypeSharingLink,
			SharingURL: rawURL,
			SiteHost:   parsed.Host,
		}, nil
	}

	// Check for direct document URLs — match against path without query.
	urlWithoutQuery := parsed.Scheme + "://" + parsed.Host + parsed.Path
	if m := directDocPattern.FindStringSubmatch(urlWithoutQuery); m != nil {
		host := m[1]
		siteName := m[2]
		docPath := m[3]

		return &ResourceRef{
			Type:     RefTypeDriveItem,
			SiteHost: host,
			SitePath: "sites/" + siteName,
			ItemPath: docPath,
		}, nil
	}

	// Fallback: treat any sharepoint.com URL as a sharing link.
	return &ResourceRef{
		Type:       RefTypeSharingLink,
		SharingURL: rawURL,
		SiteHost:   parsed.Host,
	}, nil
}
