package pandoc

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvert_NilOptions(t *testing.T) {
	_, err := Convert(context.Background(), nil)
	assert.EqualError(t, err, "options must not be nil")
}

func TestConvert_EmptyInputPath(t *testing.T) {
	_, err := Convert(context.Background(), &Options{})
	assert.EqualError(t, err, "input path is required")
}

func TestConvert_NonDocxExtension(t *testing.T) {
	_, err := Convert(context.Background(), &Options{InputPath: "file.txt"})
	assert.ErrorContains(t, err, "must be a .docx file")
}

func TestConvert_NonDocxExtensionCaseInsensitive(t *testing.T) {
	// .DOCX should be accepted (but will fail at lookPath since pandoc stub not set).
	// We only test the extension check is case-insensitive here.
	_, err := Convert(context.Background(), &Options{
		InputPath: "file.DOCX",
		LookPath: func(string) (string, error) {
			return "", exec.ErrNotFound
		},
	})
	// Should fail at lookPath, not extension check.
	assert.ErrorIs(t, err, ErrNotInstalled)
}

func TestConvert_PandocNotInstalled(t *testing.T) {
	_, err := Convert(context.Background(), &Options{
		InputPath: "test.docx",
		LookPath: func(string) (string, error) {
			return "", exec.ErrNotFound
		},
	})
	assert.ErrorIs(t, err, ErrNotInstalled)
	assert.ErrorContains(t, err, "pandoc.org")
}

func TestConvert_Success(t *testing.T) {
	opts := &Options{
		InputPath: "input.docx",
		LookPath: func(string) (string, error) {
			return "/usr/bin/echo", nil
		},
		CommandContext: func(ctx context.Context, name string, arg ...string) *exec.Cmd {
			// Stub: echo markdown output instead of running pandoc.
			return exec.CommandContext(ctx, "echo", "# Hello World")
		},
	}

	result, err := Convert(context.Background(), opts)
	require.NoError(t, err)
	assert.Contains(t, result, "# Hello World")
}

func TestConvert_WithImagesDir(t *testing.T) {
	var capturedArgs []string
	opts := &Options{
		InputPath: "input.docx",
		ImagesDir: "/tmp/images",
		LookPath: func(string) (string, error) {
			return "/usr/bin/echo", nil
		},
		CommandContext: func(ctx context.Context, name string, arg ...string) *exec.Cmd {
			capturedArgs = arg
			return exec.CommandContext(ctx, "echo", "ok")
		},
	}

	_, err := Convert(context.Background(), opts)
	require.NoError(t, err)
	assert.Contains(t, capturedArgs, "--extract-media")
	assert.Contains(t, capturedArgs, "/tmp/images")
}

func TestConvert_WithExtraArgs(t *testing.T) {
	var capturedArgs []string
	opts := &Options{
		InputPath: "input.docx",
		ExtraArgs: []string{"--wrap=none", "--columns=80"},
		LookPath: func(string) (string, error) {
			return "/usr/bin/echo", nil
		},
		CommandContext: func(ctx context.Context, name string, arg ...string) *exec.Cmd {
			capturedArgs = arg
			return exec.CommandContext(ctx, "echo", "ok")
		},
	}

	_, err := Convert(context.Background(), opts)
	require.NoError(t, err)
	assert.Contains(t, capturedArgs, "--wrap=none")
	assert.Contains(t, capturedArgs, "--columns=80")
}

func TestConvert_PandocFailsWithStderr(t *testing.T) {
	opts := &Options{
		InputPath: "input.docx",
		LookPath: func(string) (string, error) {
			return "/bin/sh", nil
		},
		CommandContext: func(ctx context.Context, name string, arg ...string) *exec.Cmd {
			return exec.CommandContext(ctx, "sh", "-c", "echo 'some error' >&2; exit 1")
		},
	}

	_, err := Convert(context.Background(), opts)
	assert.ErrorContains(t, err, "some error")
}

func TestConvert_PandocFailsWithoutStderr(t *testing.T) {
	opts := &Options{
		InputPath: "input.docx",
		LookPath: func(string) (string, error) {
			return "/bin/sh", nil
		},
		CommandContext: func(ctx context.Context, name string, arg ...string) *exec.Cmd {
			return exec.CommandContext(ctx, "sh", "-c", "exit 1")
		},
	}

	_, err := Convert(context.Background(), opts)
	assert.ErrorContains(t, err, "pandoc failed")
}

func TestConvert_Timeout(t *testing.T) {
	opts := &Options{
		InputPath: "input.docx",
		Timeout:   50 * time.Millisecond,
		LookPath: func(string) (string, error) {
			return "/bin/sh", nil
		},
		CommandContext: func(ctx context.Context, name string, arg ...string) *exec.Cmd {
			return exec.CommandContext(ctx, "sleep", "10")
		},
	}

	_, err := Convert(context.Background(), opts)
	assert.ErrorContains(t, err, "timed out")
}

func TestBuildArgs_Basic(t *testing.T) {
	args := buildArgs(&Options{InputPath: "test.docx"})
	assert.Equal(t, []string{"-f", "docx", "-t", "gfm", "test.docx"}, args)
}

func TestBuildArgs_WithImagesAndExtra(t *testing.T) {
	args := buildArgs(&Options{
		InputPath: "test.docx",
		ImagesDir: "./media",
		ExtraArgs: []string{"--wrap=none"},
	})
	expected := []string{"-f", "docx", "-t", "gfm", "--extract-media", "./media", "--wrap=none", "test.docx"}
	assert.Equal(t, expected, args)
}

func TestCheckInstalled(t *testing.T) {
	// This test is environment-dependent; just verify the function runs.
	err := CheckInstalled()
	if err != nil {
		assert.ErrorIs(t, err, ErrNotInstalled)
	}
}

func TestOptions_DefaultTimeout(t *testing.T) {
	opts := &Options{}
	assert.Equal(t, DefaultTimeout, opts.timeout())
}

func TestOptions_CustomTimeout(t *testing.T) {
	opts := &Options{Timeout: 30 * time.Second}
	assert.Equal(t, 30*time.Second, opts.timeout())
}

// TestConvert_Integration runs a real pandoc conversion if pandoc is installed
// and a test fixture exists.
func TestConvert_Integration(t *testing.T) {
	if err := CheckInstalled(); err != nil {
		t.Skip("pandoc not installed, skipping integration test")
	}

	// Create a minimal .docx using pandoc itself (markdown -> docx -> markdown round-trip).
	tmpDir := t.TempDir()
	docxPath := filepath.Join(tmpDir, "test.docx")
	mdInput := "# Test Heading\n\nA paragraph with **bold** and *italic* text.\n\n- Item one\n- Item two\n\n| Col A | Col B |\n|-------|-------|\n| 1     | 2     |\n"

	// Create docx from markdown.
	createCmd := exec.Command("pandoc", "-f", "gfm", "-t", "docx", "-o", docxPath)
	createCmd.Stdin = strings.NewReader(mdInput)
	out, err := createCmd.CombinedOutput()
	require.NoError(t, err, "failed to create test docx: %s", string(out))

	// Convert back to markdown.
	result, err := Convert(context.Background(), &Options{
		InputPath: docxPath,
	})
	require.NoError(t, err)

	// Verify key elements survived round-trip.
	assert.Contains(t, result, "Test Heading")
	assert.Contains(t, result, "bold")
	assert.Contains(t, result, "italic")
	assert.Contains(t, result, "Item one")
	assert.Contains(t, result, "Item two")
	assert.Contains(t, result, "Col A")
}

// TestConvert_IntegrationExtractMedia verifies image extraction works.
func TestConvert_IntegrationExtractMedia(t *testing.T) {
	if err := CheckInstalled(); err != nil {
		t.Skip("pandoc not installed, skipping integration test")
	}

	tmpDir := t.TempDir()
	docxPath := filepath.Join(tmpDir, "test.docx")
	mediaDir := filepath.Join(tmpDir, "media")

	// Create a simple docx (no images, but verify the flag doesn't break anything).
	createCmd := exec.Command("pandoc", "-f", "gfm", "-t", "docx", "-o", docxPath)
	createCmd.Stdin = strings.NewReader("# Hello\n\nSome text.\n")
	out, err := createCmd.CombinedOutput()
	require.NoError(t, err, "failed to create test docx: %s", string(out))

	result, err := Convert(context.Background(), &Options{
		InputPath: docxPath,
		ImagesDir: mediaDir,
	})
	require.NoError(t, err)
	assert.Contains(t, result, "Hello")
}

// TestConvert_IntegrationLargeDocument verifies conversion doesn't hang on larger input.
func TestConvert_IntegrationLargeDocument(t *testing.T) {
	if err := CheckInstalled(); err != nil {
		t.Skip("pandoc not installed, skipping integration test")
	}

	tmpDir := t.TempDir()
	docxPath := filepath.Join(tmpDir, "large.docx")

	// Generate a large-ish markdown input.
	var sb strings.Builder
	for i := 0; i < 500; i++ {
		sb.WriteString("## Section ")
		sb.WriteString(strings.Repeat("x", 10))
		sb.WriteString("\n\n")
		sb.WriteString("Lorem ipsum dolor sit amet, consectetur adipiscing elit. ")
		sb.WriteString("Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua.\n\n")
	}

	createCmd := exec.Command("pandoc", "-f", "gfm", "-t", "docx", "-o", docxPath)
	createCmd.Stdin = strings.NewReader(sb.String())
	out, err := createCmd.CombinedOutput()
	require.NoError(t, err, "failed to create large test docx: %s", string(out))

	info, err := os.Stat(docxPath)
	require.NoError(t, err)
	t.Logf("large test docx size: %d bytes", info.Size())

	result, err := Convert(context.Background(), &Options{
		InputPath: docxPath,
		Timeout:   30 * time.Second,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, result)
	assert.Contains(t, result, "Section")
}
