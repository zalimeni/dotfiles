package sp2md

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRootCommandExecutes(t *testing.T) {
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
