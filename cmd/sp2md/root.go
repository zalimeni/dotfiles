package sp2md

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/zalimeni/sp2md/internal/auth"
	mdclean "github.com/zalimeni/sp2md/internal/cleanup"
	"github.com/zalimeni/sp2md/internal/pandoc"
	"github.com/zalimeni/sp2md/internal/sharepoint"
)

var (
	flagFile      string
	flagURL       string
	flagOutput    string
	flagImagesDir string
	flagClientID  string
	flagTenantID  string
	flagTokenPath string
	flagNoClean   bool
)

// Exit codes for structured error reporting.
const (
	exitSuccess        = 0
	exitConversionErr  = 1
	exitAuthErr        = 2
	exitNetworkErr     = 3
)

var rootCmd = &cobra.Command{
	Use:   "sp2md",
	Short: "Convert SharePoint documents to Markdown",
	Long:  "sp2md converts SharePoint .aspx pages and documents into clean Markdown files.",
	PersistentPreRunE: applyEnvDefaults,
	RunE:              runConvert,
}

func runConvert(cmd *cobra.Command, _ []string) error {
	if flagFile == "" && flagURL == "" {
		return cmd.Help()
	}
	if flagFile != "" && flagURL != "" {
		return errors.New("specify either --file or --url, not both")
	}

	var docxPath string
	var cleanup func()

	if flagFile != "" {
		docxPath = flagFile
	} else {
		// URL path: authenticate and download.
		path, cleanFn, err := acquireFromURL(cmd.Context())
		if err != nil {
			return err
		}
		docxPath = path
		cleanup = cleanFn
	}

	if cleanup != nil {
		defer cleanup()
	}

	md, err := pandoc.Convert(cmd.Context(), &pandoc.Options{
		InputPath: docxPath,
		ImagesDir: flagImagesDir,
	})
	if err != nil {
		if errors.Is(err, pandoc.ErrNotInstalled) {
			return &ExitError{code: exitConversionErr, err: err}
		}
		return &ExitError{code: exitConversionErr, err: fmt.Errorf("conversion failed: %w", err)}
	}

	if !flagNoClean {
		md = mdclean.Clean(md, flagImagesDir)
	}

	if flagOutput != "" {
		if err := os.WriteFile(flagOutput, []byte(md), 0644); err != nil {
			return fmt.Errorf("writing output file: %w", err)
		}
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Wrote %s\n", flagOutput)
	} else {
		_, _ = fmt.Fprint(cmd.OutOrStdout(), md)
	}

	return nil
}

// acquireFromURL authenticates with Microsoft Graph and downloads the document.
func acquireFromURL(ctx context.Context) (docxPath string, cleanup func(), err error) {
	cfg := &auth.Config{
		ClientID:  flagClientID,
		TenantID:  flagTenantID,
		TokenPath: flagTokenPath,
	}

	authCtx, authCancel := context.WithTimeout(ctx, 5*time.Minute)
	defer authCancel()

	tok, err := auth.Authenticate(authCtx, cfg)
	if err != nil {
		return "", nil, &ExitError{code: exitAuthErr, err: fmt.Errorf("authentication failed: %w", err)}
	}

	client := &sharepoint.Client{
		AccessToken: tok.AccessToken,
	}

	dlCtx, dlCancel := context.WithTimeout(ctx, 5*time.Minute)
	defer dlCancel()

	localPath, err := client.Download(dlCtx, flagURL)
	if err != nil {
		if errors.Is(err, sharepoint.ErrPermissionDenied) {
			return "", nil, &ExitError{code: exitAuthErr, err: err}
		}
		if errors.Is(err, sharepoint.ErrNotFound) {
			return "", nil, &ExitError{code: exitNetworkErr, err: err}
		}
		return "", nil, &ExitError{code: exitNetworkErr, err: fmt.Errorf("download failed: %w", err)}
	}

	cleanup = func() {
		_ = os.Remove(localPath)
	}

	return localPath, cleanup, nil
}

// ExitError wraps an error with an exit code for structured CLI reporting.
type ExitError struct {
	code int
	err  error
}

// Code returns the exit code.
func (e *ExitError) Code() int {
	return e.code
}

func (e *ExitError) Error() string {
	return e.err.Error()
}

func (e *ExitError) Unwrap() error {
	return e.err
}

func init() {
	rootCmd.PersistentFlags().StringVar(&flagFile, "file", "", "path to a local .docx file to convert")
	rootCmd.PersistentFlags().StringVar(&flagURL, "url", "", "URL of a SharePoint page to convert")
	rootCmd.PersistentFlags().StringVar(&flagOutput, "output", "", "output file path for the Markdown result")
	rootCmd.PersistentFlags().StringVar(&flagImagesDir, "images-dir", "", "directory to save extracted images")
	rootCmd.PersistentFlags().StringVar(&flagClientID, "client-id", "", "Azure AD application (client) ID (env: SP2MD_CLIENT_ID)")
	rootCmd.PersistentFlags().StringVar(&flagTenantID, "tenant-id", "common", "Azure AD tenant ID (env: SP2MD_TENANT_ID)")
	rootCmd.PersistentFlags().StringVar(&flagTokenPath, "token-path", "", "path to token cache file (default: ~/.config/sp2md/token.json)")
	rootCmd.PersistentFlags().BoolVar(&flagNoClean, "no-clean", false, "disable markdown post-processing cleanup")
}

// applyEnvDefaults applies environment variable fallbacks for auth flags.
// It runs after cobra has parsed flags, so cmd.Flags().Changed() accurately
// reflects whether the user explicitly provided a value on the command line.
func applyEnvDefaults(cmd *cobra.Command, _ []string) error {
	if !cmd.Flags().Changed("client-id") {
		if v := os.Getenv("SP2MD_CLIENT_ID"); v != "" {
			flagClientID = v
		}
	}
	if !cmd.Flags().Changed("tenant-id") {
		if v := os.Getenv("SP2MD_TENANT_ID"); v != "" {
			flagTenantID = v
		}
	}
	return nil
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}
