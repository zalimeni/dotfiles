package sp2md

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	flagFile      string
	flagURL       string
	flagOutput    string
	flagImagesDir string
	flagClientID  string
	flagTenantID  string
	flagTokenPath string
)

var rootCmd = &cobra.Command{
	Use:   "sp2md",
	Short: "Convert SharePoint documents to Markdown",
	Long:  "sp2md converts SharePoint .aspx pages and documents into clean Markdown files.",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("sp2md: use --help for usage information")
		return nil
	},
}

func init() {
	rootCmd.PersistentFlags().StringVar(&flagFile, "file", "", "path to a local SharePoint .aspx file")
	rootCmd.PersistentFlags().StringVar(&flagURL, "url", "", "URL of a SharePoint page to convert")
	rootCmd.PersistentFlags().StringVar(&flagOutput, "output", "", "output file path for the Markdown result")
	rootCmd.PersistentFlags().StringVar(&flagImagesDir, "images-dir", "", "directory to save extracted images")
	rootCmd.PersistentFlags().StringVar(&flagClientID, "client-id", "", "Azure AD application (client) ID (env: SP2MD_CLIENT_ID)")
	rootCmd.PersistentFlags().StringVar(&flagTenantID, "tenant-id", "common", "Azure AD tenant ID (env: SP2MD_TENANT_ID)")
	rootCmd.PersistentFlags().StringVar(&flagTokenPath, "token-path", "", "path to token cache file (default: ~/.config/sp2md/token.json)")
}

// Execute runs the root command.
func Execute() error {
	// Allow environment variables to set defaults for auth flags.
	if v := os.Getenv("SP2MD_CLIENT_ID"); v != "" && flagClientID == "" {
		flagClientID = v
	}
	if v := os.Getenv("SP2MD_TENANT_ID"); v != "" && flagTenantID == "" {
		flagTenantID = v
	}
	return rootCmd.Execute()
}
