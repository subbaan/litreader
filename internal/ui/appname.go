package ui

import (
	"os"
	"path/filepath"
)

// GetAppName returns the name of the currently running binary
func GetAppName() string {
	if len(os.Args) > 0 {
		return filepath.Base(os.Args[0])
	}
	return "txtreader" // Fallback
}
