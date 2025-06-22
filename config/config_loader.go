package config

import (
	"fmt"
	"log"
	"os"
	"strings"
)

// LoadConfigurationWithFallback loads configuration from TOML file with environment variable fallback
func LoadConfigurationWithFallback(configPath string) (*Config, error) {
	cfg, err := Load(configPath)
	if err != nil {
		// If config file doesn't exist but it's the default path, that's OK
		if os.IsNotExist(err) && configPath == "nectar.toml" {
			log.Println("No config file found, using defaults and environment variables")
			cfg, _ = Load("") // This will use defaults + env vars
		} else {
			return nil, fmt.Errorf("failed to load configuration: %w", err)
		}
	}
	return cfg, nil
}

// HandleInitCommand handles the 'nectar init' subcommand
func HandleInitCommand() {
	configPath := "nectar.toml"
	interactive := true
	
	// Parse init command flags
	for i := 2; i < len(os.Args); i++ {
		if os.Args[i] == "--no-interactive" || os.Args[i] == "-n" {
			interactive = false
		} else if !strings.HasPrefix(os.Args[i], "-") {
			configPath = os.Args[i]
		}
	}
	
	var err error
	if interactive {
		err = InteractiveInit(configPath)
	} else {
		err = InitCommand(configPath)
	}
	
	if err != nil {
		log.Fatalf("Failed to initialize config: %v", err)
	}
	os.Exit(0)
}

// HandleMigrateEnvCommand handles the 'nectar migrate-env' subcommand
func HandleMigrateEnvCommand() {
	configPath := "nectar.toml"
	if len(os.Args) > 2 {
		configPath = os.Args[2]
	}
	
	if err := MigrateFromEnv(configPath); err != nil {
		log.Fatalf("Failed to migrate environment config: %v", err)
	}
	os.Exit(0)
}