package config

import (
	"os"
	"path/filepath"
	"runtime"
)

const (
	EnvConfigDir = "EMR_CONFIG_DIR"

	appData       = "AppData"
	localAppData  = "LocalAppData"
	xdgConfigHome = "XDG_CONFIG_HOME"

	configDirName = "emergency-credentials-receive"
)

// ConfigDir returns the path to the config directory.
// Config path precedence: EMR_CONFIG_DIR, XDG_CONFIG_HOME, AppData (windows only), HOME.
func ConfigDir() string {
	var path string
	if a := os.Getenv(EnvConfigDir); a != "" {
		path = a
	} else if b := os.Getenv(xdgConfigHome); b != "" {
		path = filepath.Join(b, configDirName)
	} else if c := os.Getenv(appData); runtime.GOOS == "windows" && c != "" {
		path = filepath.Join(c, configDirName)
	} else {
		d, _ := os.UserHomeDir()
		path = filepath.Join(d, ".config", configDirName)
	}
	return path
}
