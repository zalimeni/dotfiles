package sp2md

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/zalimeni/sp2md/internal/auth"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Authenticate with Microsoft Graph API using device code flow",
	Long: `Initiates the OAuth2 device code flow to obtain an access token for
Microsoft Graph API. The token is cached locally and refreshed automatically
on subsequent runs.

Requires an Azure AD (Entra ID) app registration with the following API
permissions: Files.Read.All, Sites.Read.All.`,
	RunE: runAuth,
}

func init() {
	rootCmd.AddCommand(authCmd)
}

func runAuth(cmd *cobra.Command, _ []string) error {
	cfg := &auth.Config{
		ClientID:  flagClientID,
		TenantID:  flagTenantID,
		TokenPath: flagTokenPath,
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), 5*time.Minute)
	defer cancel()

	tok, err := auth.Authenticate(ctx, cfg)
	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Authenticated successfully. Token expires at %s.\n",
		tok.ExpiresAt.Format(time.RFC3339))
	return nil
}
