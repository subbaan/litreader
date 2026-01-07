package main

import (
	"fmt"
	"log"

	"github.com/subbass/litreader/internal/config"
)

func main() {
	configDir, err := config.GetConfigDir()
	if err != nil {
		log.Fatalf("Failed to get config dir: %v", err)
	}

	cacheDir, err := config.GetCacheDir()
	if err != nil {
		log.Fatalf("Failed to get cache dir: %v", err)
	}

	configPath, err := config.GetConfigPath()
	if err != nil {
		log.Fatalf("Failed to get config path: %v", err)
	}

	fmt.Printf("Config directory: %s\n", configDir)
	fmt.Printf("Cache directory:  %s\n", cacheDir)
	fmt.Printf("Config file path: %s\n", configPath)
}
