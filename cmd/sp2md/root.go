package sp2md

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	flagFile      string
	flagURL       string
	flagOutput    string
	flagImagesDir string
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
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}
