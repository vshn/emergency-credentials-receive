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
func ConfigFile() string {
	return filepath.Join(ConfigDir(), configFile)
}

func RetrieveConfig() (Config, error) {
	configMux.RLock()
	defer configMux.RUnlock()

	configFile := ConfigFile()

	yamlFile, err := os.ReadFile(configFile)
	if err != nil {
		return Config{}, fmt.Errorf("error reading config file %q: %w", configFile, err)
	}

	var config Config
	yaml.Unmarshal([]byte(yamlFile), &config)
	if err != nil {
		return Config{}, fmt.Errorf("error parsing config file %q: %w", configFile, err)
	}

	return config, nil
}

func SaveConfig(config Config) error {
	configMux.Lock()
	defer configMux.Unlock()

	if err := os.MkdirAll(ConfigDir(), 0700); err != nil {
		return fmt.Errorf("error creating config dir %q: %w", ConfigDir(), err)
	}

	configFile := filepath.Join(ConfigDir(), configFile)

	yamlFile, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("error marshalling config: %w", err)
	}

	if err := os.WriteFile(configFile, yamlFile, 0600); err != nil {
		return fmt.Errorf("error writing config file %q: %w", configFile, err)
	}

	return nil
}
