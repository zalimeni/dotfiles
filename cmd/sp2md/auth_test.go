package sp2md

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAuthCommandRegistered(t *testing.T) {
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "auth" {
			found = true
			break
		}
	}
	assert.True(t, found, "auth subcommand should be registered")
}

func TestAuthFlags(t *testing.T) {
	flags := rootCmd.PersistentFlags()

	for _, name := range []string{"client-id", "tenant-id", "token-path"} {
		f := flags.Lookup(name)
		assert.NotNil(t, f, "flag %q should exist", name)
		assert.NotEmpty(t, f.Usage, "flag %q should have a description", name)
	}

	// tenant-id should default to "common"
	f := flags.Lookup("tenant-id")
	assert.Equal(t, "common", f.DefValue)
}
