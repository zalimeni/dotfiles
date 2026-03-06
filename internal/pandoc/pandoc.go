// Package pandoc converts .docx files to GitHub Flavored Markdown by
// shelling out to the pandoc binary. It validates that pandoc is installed,
// builds the appropriate argument list, and captures output and errors.
package pandoc

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// DefaultTimeout is the maximum time a pandoc conversion is allowed to run.
const DefaultTimeout = 5 * time.Minute

// ErrNotInstalled is returned when the pandoc binary cannot be found on PATH.
var ErrNotInstalled = errors.New("pandoc is not installed or not found on PATH; install it from https://pandoc.org")

// Options configures a single conversion run.
type Options struct {
	// InputPath is the path to the .docx file to convert.
	InputPath string

	// ImagesDir, when set, tells pandoc to extract embedded media to this
	// directory via the --extract-media flag.
	ImagesDir string

	// ExtraArgs are additional pandoc flags passed through verbatim.
	ExtraArgs []string

	// Timeout overrides DefaultTimeout for this conversion.
	Timeout time.Duration

	// LookPath overrides exec.LookPath for testing.
	LookPath func(file string) (string, error)

	// CommandContext overrides exec.CommandContext for testing.
	CommandContext func(ctx context.Context, name string, arg ...string) *exec.Cmd
}

func (o *Options) timeout() time.Duration {
	if o.Timeout > 0 {
		return o.Timeout
	}
	return DefaultTimeout
}

func (o *Options) lookPath(file string) (string, error) {
	if o.LookPath != nil {
		return o.LookPath(file)
	}
	return exec.LookPath(file)
}

func (o *Options) commandContext(ctx context.Context, name string, arg ...string) *exec.Cmd {
	if o.CommandContext != nil {
		return o.CommandContext(ctx, name, arg...)
	}
	return exec.CommandContext(ctx, name, arg...)
}

// Convert runs pandoc to convert the .docx file described by opts into GFM
// markdown. It returns the markdown content as a string.
func Convert(ctx context.Context, opts *Options) (string, error) {
	if opts == nil {
		return "", errors.New("options must not be nil")
	}
	if opts.InputPath == "" {
		return "", errors.New("input path is required")
	}
	if !strings.HasSuffix(strings.ToLower(opts.InputPath), ".docx") {
		return "", fmt.Errorf("input file must be a .docx file: %s", opts.InputPath)
	}

	// Verify pandoc is available.
	pandocBin, err := opts.lookPath("pandoc")
	if err != nil {
		return "", ErrNotInstalled
	}

	args := buildArgs(opts)

	ctx, cancel := context.WithTimeout(ctx, opts.timeout())
	defer cancel()

	cmd := opts.commandContext(ctx, pandocBin, args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return "", fmt.Errorf("pandoc timed out after %s", opts.timeout())
		}
		stderrStr := strings.TrimSpace(stderr.String())
		if stderrStr != "" {
			return "", fmt.Errorf("pandoc failed: %s", stderrStr)
		}
		return "", fmt.Errorf("pandoc failed: %w", err)
	}

	return stdout.String(), nil
}

// CheckInstalled returns nil if pandoc is found on PATH, or ErrNotInstalled.
func CheckInstalled() error {
	_, err := exec.LookPath("pandoc")
	if err != nil {
		return ErrNotInstalled
	}
	return nil
}

func buildArgs(opts *Options) []string {
	args := []string{
		"-f", "docx",
		"-t", "gfm",
	}

	if opts.ImagesDir != "" {
		args = append(args, "--extract-media", opts.ImagesDir)
	}

	args = append(args, opts.ExtraArgs...)
	args = append(args, opts.InputPath)

	return args
}
