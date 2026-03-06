package sp2md

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRootCommandExecutes(t *testing.T) {
	defer resetFlags()
	rootCmd.SetArgs([]string{"--help"})
	err := rootCmd.Execute()
	assert.NoError(t, err)
}

func TestRootCommandHasFlags(t *testing.T) {
	flags := rootCmd.PersistentFlags()

	for _, name := range []string{"file", "url", "output", "images-dir"} {
		f := flags.Lookup(name)
		assert.NotNil(t, f, "flag %q should exist", name)
		assert.NotEmpty(t, f.Usage, "flag %q should have a description", name)
	}
}

func TestApplyEnvDefaults_ClientID(t *testing.T) {
	resetFlags()
	defer resetFlags()

	t.Setenv("SP2MD_CLIENT_ID", "env-client-id")

	cmd := &cobra.Command{}
	cmd.Flags().AddFlagSet(rootCmd.PersistentFlags())

	// Simulate: no --client-id flag was passed.
	require.NoError(t, cmd.ParseFlags([]string{}))

	err := applyEnvDefaults(cmd, nil)
	require.NoError(t, err)

	assert.Equal(t, "env-client-id", flagClientID, "SP2MD_CLIENT_ID env var should set flagClientID when flag not provided")
}

func TestApplyEnvDefaults_TenantID(t *testing.T) {
	resetFlags()
	defer resetFlags()

	t.Setenv("SP2MD_TENANT_ID", "env-tenant-id")

	cmd := &cobra.Command{}
	cmd.Flags().AddFlagSet(rootCmd.PersistentFlags())

	require.NoError(t, cmd.ParseFlags([]string{}))

	err := applyEnvDefaults(cmd, nil)
	require.NoError(t, err)

	assert.Equal(t, "env-tenant-id", flagTenantID, "SP2MD_TENANT_ID env var should set flagTenantID when flag not provided")
}

func TestApplyEnvDefaults_FlagOverridesEnv_ClientID(t *testing.T) {
	resetFlags()
	defer resetFlags()

	t.Setenv("SP2MD_CLIENT_ID", "env-client-id")

	cmd := &cobra.Command{}
	cmd.Flags().AddFlagSet(rootCmd.PersistentFlags())

	require.NoError(t, cmd.ParseFlags([]string{"--client-id", "flag-client-id"}))

	err := applyEnvDefaults(cmd, nil)
	require.NoError(t, err)

	assert.Equal(t, "flag-client-id", flagClientID, "explicit --client-id flag should override SP2MD_CLIENT_ID env var")
}

func TestApplyEnvDefaults_FlagOverridesEnv_TenantID(t *testing.T) {
	resetFlags()
	defer resetFlags()

	t.Setenv("SP2MD_TENANT_ID", "env-tenant-id")

	cmd := &cobra.Command{}
	cmd.Flags().AddFlagSet(rootCmd.PersistentFlags())

	require.NoError(t, cmd.ParseFlags([]string{"--tenant-id", "flag-tenant-id"}))

	err := applyEnvDefaults(cmd, nil)
	require.NoError(t, err)

	assert.Equal(t, "flag-tenant-id", flagTenantID, "explicit --tenant-id flag should override SP2MD_TENANT_ID env var")
}

func TestApplyEnvDefaults_NoEnvNoFlag_DefaultsPreserved(t *testing.T) {
	resetFlags()
	defer resetFlags()

	// Ensure env vars are unset.
	t.Setenv("SP2MD_CLIENT_ID", "")
	t.Setenv("SP2MD_TENANT_ID", "")

	cmd := &cobra.Command{}
	cmd.Flags().AddFlagSet(rootCmd.PersistentFlags())

	require.NoError(t, cmd.ParseFlags([]string{}))

	err := applyEnvDefaults(cmd, nil)
	require.NoError(t, err)

	assert.Equal(t, "", flagClientID, "flagClientID should remain empty when no env var or flag")
	assert.Equal(t, "common", flagTenantID, "flagTenantID should remain 'common' when no env var or flag")
}
