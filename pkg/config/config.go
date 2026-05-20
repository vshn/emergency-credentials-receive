package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"gopkg.in/yaml.v3"
)

const configFile = "config.yaml"

var configMux = sync.RWMutex{}

type Config struct {
	PassboltKey string `yaml:"passbolt_key" json:"passbolt_key"`
}

// ConfigFile returns the path to the config file.
// Also see ConfigDir().
func ConfigFile() (string, error) {
	d, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, configFile), nil
}

func RetrieveConfig() (Config, error) {
	configMux.RLock()
	defer configMux.RUnlock()

	configFile, err := ConfigFile()
	if err != nil {
		return Config{}, fmt.Errorf("error getting config file path: %w", err)
	}

	yamlFile, err := os.ReadFile(configFile)
	if err != nil {
		return Config{}, fmt.Errorf("error reading config file %q: %w", configFile, err)
	}

	var config Config
	if err := yaml.Unmarshal([]byte(yamlFile), &config); err != nil {
		return Config{}, fmt.Errorf("error parsing config file %q: %w", configFile, err)
	}

	return config, nil
}

func SaveConfig(config Config) error {
	configMux.Lock()
	defer configMux.Unlock()

	configDir, err := ConfigDir()
	if err != nil {
		return fmt.Errorf("error getting config dir: %w", err)
	}

	if err := os.MkdirAll(configDir, 0700); err != nil {
		return fmt.Errorf("error creating config dir %q: %w", configDir, err)
	}

	yamlFile, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("error marshalling config: %w", err)
	}

	configFile := filepath.Join(configDir, configFile)
	if err := os.WriteFile(configFile, yamlFile, 0600); err != nil {
		return fmt.Errorf("error writing config file %q: %w", configFile, err)
	}

	return nil
}
