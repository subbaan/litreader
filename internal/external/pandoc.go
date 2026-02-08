package external

import (
	"context"
	"os/exec"
	"strings"
	"time"
)

const (
	// Maximum size for pandoc rendering (2MB)
	maxPandocSize = 2 * 1024 * 1024
	// Maximum time to wait for pandoc (2 seconds)
	pandocTimeout = 2 * time.Second
)

// RenderMarkdown converts markdown content to plain text using pandoc
func RenderMarkdown(content string) (string, error) {
	// Check if pandoc is available
	if _, err := exec.LookPath("pandoc"); err != nil {
		// If pandoc not available, return content as-is
		return content, nil
	}

	// Skip pandoc for very large files to avoid hanging
	if len(content) > maxPandocSize {
		return content, nil
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), pandocTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "pandoc", "-f", "markdown", "-t", "plain", "--wrap=preserve")
	cmd.Stdin = strings.NewReader(content)

	output, err := cmd.Output()
	if err != nil {
		// If pandoc fails or times out, return original content
		return content, nil
	}

	return string(output), nil
}

// IsPandocAvailable checks if pandoc is installed
func IsPandocAvailable() bool {
	_, err := exec.LookPath("pandoc")
	return err == nil
}
