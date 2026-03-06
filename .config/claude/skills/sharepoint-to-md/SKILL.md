---
name: sharepoint-to-md
description: Convert SharePoint documents and .docx files to clean Markdown using sp2md. Use when the user wants to convert a SharePoint page URL or a local .docx file into Markdown format, extract images, or authenticate with Microsoft Graph for SharePoint access.
user_invocable: true
---

# sharepoint-to-md

Convert SharePoint documents and .docx files to clean Markdown.

## Prerequisites

The `sp2md` binary and `pandoc` must be installed. If either is missing, guide the user through setup.

### Install pandoc

```bash
brew install pandoc
```

### Build and install sp2md

```bash
cd ~/workspace/sp2md && make build
# Optionally copy to a directory on PATH:
cp sp2md /usr/local/bin/sp2md
```

If `sp2md` is not on PATH, invoke it directly at `~/workspace/sp2md/sp2md`.

### Azure AD app registration (required for SharePoint URL mode only)

To convert documents via SharePoint URL, an Azure AD (Entra ID) app registration is required:

1. Go to [Azure Portal > App registrations](https://portal.azure.com/#view/Microsoft_AAD_RegisteredApps/ApplicationsListBlade)
2. Create a new registration (any name, single-tenant or multi-tenant)
3. Under **Authentication**, enable "Allow public client flows" (for device code flow)
4. Under **API permissions**, add Microsoft Graph delegated permissions:
   - `Files.Read.All`
   - `Sites.Read.All`
5. Note the **Application (client) ID** and **Directory (tenant) ID**
6. Set environment variables or pass as flags:
   ```bash
   export SP2MD_CLIENT_ID="<your-client-id>"
   export SP2MD_TENANT_ID="<your-tenant-id>"
   ```

## Usage

### Convert a local .docx file to Markdown

```bash
sp2md --file document.docx
```

Output goes to stdout. To write to a file:

```bash
sp2md --file document.docx --output document.md
```

### Extract images during conversion

```bash
sp2md --file document.docx --output document.md --images-dir ./images
```

### Convert a SharePoint page by URL

Requires authentication (see Azure AD setup above). On first use, run the auth subcommand:

```bash
sp2md auth --client-id "$SP2MD_CLIENT_ID" --tenant-id "$SP2MD_TENANT_ID"
```

Then convert:

```bash
sp2md --url "https://contoso.sharepoint.com/sites/team/SitePages/page.aspx" --output page.md
```

### Authenticate explicitly

```bash
sp2md auth --client-id "$SP2MD_CLIENT_ID" --tenant-id "$SP2MD_TENANT_ID"
```

The token is cached at `~/.config/sp2md/token.json` (override with `--token-path`).

## Flags reference

| Flag | Description | Default |
|------|-------------|---------|
| `--file` | Path to a local .docx file | |
| `--url` | SharePoint page URL to convert | |
| `--output` | Output file path (default: stdout) | |
| `--images-dir` | Directory for extracted images | |
| `--client-id` | Azure AD application (client) ID | `$SP2MD_CLIENT_ID` |
| `--tenant-id` | Azure AD tenant ID | `common` / `$SP2MD_TENANT_ID` |
| `--token-path` | Token cache file path | `~/.config/sp2md/token.json` |

## Exit codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Conversion error (pandoc not installed, bad input) |
| 2 | Authentication error |
| 3 | Network error (download failed, not found) |

## Error handling

- **pandoc not found**: Tell the user to run `brew install pandoc`
- **sp2md not found**: Tell the user to run `cd ~/workspace/sp2md && make build`
- **Authentication errors (exit 2)**: Guide user through `sp2md auth` or check Azure AD setup
- **Network errors (exit 3)**: Check URL validity and network connectivity

## Example invocations

Convert a local file:
```bash
sp2md --file ~/Downloads/report.docx --output ~/Documents/report.md --images-dir ~/Documents/report-images
```

Convert from SharePoint:
```bash
sp2md --url "https://myorg.sharepoint.com/sites/wiki/SitePages/Architecture.aspx" --output architecture.md --images-dir ./images
```
