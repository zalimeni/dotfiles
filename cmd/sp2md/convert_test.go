package sp2md

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	pflag "github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunConvert_NoFlags_ShowsHelp(t *testing.T) {
	// Reset flags for isolated test.
	flagFile = ""
	flagURL = ""
	flagOutput = ""
	flagImagesDir = ""
	defer resetFlags()

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{})

	err := rootCmd.Execute()
	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "sp2md")
}

func TestRunConvert_BothFileAndURL_ReturnsError(t *testing.T) {
	flagFile = ""
	flagURL = ""
	flagOutput = ""
	flagImagesDir = ""
	defer resetFlags()

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"--file", "doc.docx", "--url", "https://example.sharepoint.com/foo"})

	err := rootCmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "specify either --file or --url, not both")
}

func TestRunConvert_FileNotDocx_ReturnsConversionError(t *testing.T) {
	flagFile = ""
	flagURL = ""
	flagOutput = ""
	flagImagesDir = ""
	defer resetFlags()

	// Create a temp file that is not .docx
	tmpDir := t.TempDir()
	txtFile := filepath.Join(tmpDir, "test.txt")
	require.NoError(t, os.WriteFile(txtFile, []byte("hello"), 0644))

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"--file", txtFile})

	err := rootCmd.Execute()
	assert.Error(t, err)
	// pandoc.Convert rejects non-.docx files
	assert.Contains(t, err.Error(), ".docx")
}

func TestRunConvert_FileMissing_ReturnsError(t *testing.T) {
	flagFile = ""
	flagURL = ""
	flagOutput = ""
	flagImagesDir = ""
	defer resetFlags()

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"--file", "/nonexistent/path/doc.docx"})

	err := rootCmd.Execute()
	assert.Error(t, err)
}

func TestExitError_Code(t *testing.T) {
	e := &ExitError{code: 2, err: errors.New("auth failed")}
	assert.Equal(t, 2, e.Code())
	assert.Equal(t, "auth failed", e.Error())
	assert.Equal(t, errors.New("auth failed").Error(), errors.Unwrap(e).Error())
}

func TestExitError_Unwrap(t *testing.T) {
	inner := errors.New("inner error")
	e := &ExitError{code: 3, err: inner}
	assert.ErrorIs(t, e, inner)
}

func TestExitCodes(t *testing.T) {
	assert.Equal(t, 0, exitSuccess)
	assert.Equal(t, 1, exitConversionErr)
	assert.Equal(t, 2, exitAuthErr)
	assert.Equal(t, 3, exitNetworkErr)
}

func TestRunConvert_URLWithoutClientID_ReturnsAuthError(t *testing.T) {
	flagFile = ""
	flagURL = ""
	flagOutput = ""
	flagImagesDir = ""
	flagClientID = ""
	flagTenantID = "common"
	flagTokenPath = filepath.Join(t.TempDir(), "nonexistent-token.json")
	defer resetFlags()

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"--url", "https://contoso.sharepoint.com/:w:/s/Site/doc"})

	err := rootCmd.Execute()
	assert.Error(t, err)

	var exitErr *ExitError
	assert.True(t, errors.As(err, &exitErr))
	assert.Equal(t, exitAuthErr, exitErr.Code())
}

func resetFlags() {
	flagFile = ""
	flagURL = ""
	flagOutput = ""
	flagImagesDir = ""
	flagClientID = ""
	flagTenantID = "common"
	flagTokenPath = ""
	flagNoClean = false

	// Reset cobra's internal Changed state for each persistent flag so that
	// subsequent tests using cmd.Flags().Changed() get accurate results.
	rootCmd.PersistentFlags().VisitAll(func(f *pflag.Flag) {
		f.Changed = false
	})
}
