package config

import (
	"os"
	"path/filepath"
)

// getAppName returns the name of the currently running binary
func getAppName() string {
	if len(os.Args) > 0 {
		return filepath.Base(os.Args[0])
	}
	return "txtreader" // Fallback
}

// GetConfigDir returns the XDG-compliant config directory for the app
func GetConfigDir() (string, error) {
	var configDir string
	appName := getAppName()

	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		configDir = filepath.Join(xdg, appName)
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		configDir = filepath.Join(home, ".config", appName)
	}

	// Ensure directory exists
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return "", err
	}

	return configDir, nil
}

// GetCacheDir returns the XDG-compliant cache directory for the app
func GetCacheDir() (string, error) {
	var cacheDir string
	appName := getAppName()

	if xdg := os.Getenv("XDG_CACHE_HOME"); xdg != "" {
		cacheDir = filepath.Join(xdg, appName)
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		cacheDir = filepath.Join(home, ".cache", appName)
	}

	// Ensure directory exists
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return "", err
	}

	return cacheDir, nil
}

// GetConfigPath returns the full path to the config file
// This is a variable to allow overriding in tests
var GetConfigPath = func() (string, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return "", err
	}
	appName := getAppName()
	return filepath.Join(configDir, appName+".conf"), nil
}
