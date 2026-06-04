package config

import (
	"os"
	"path/filepath"
)

// appName is the per-application directory under the OS config root.
const appName = "chiquito"

// FileName is the name of the configuration file within the config directory.
const FileName = "config.toml"

// Dir returns chiquito's configuration directory, creating it (mode 0700) if it
// does not exist. The location follows the OS convention via os.UserConfigDir:
//
//	Linux/Unix: $XDG_CONFIG_HOME/chiquito or ~/.config/chiquito
//	macOS:      ~/Library/Application Support/chiquito
//	Windows:    %AppData%\chiquito
func Dir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(base, appName)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	return dir, nil
}

// FilePath returns the full path to the configuration file, ensuring the
// containing directory exists.
func FilePath() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, FileName), nil
}

// resolvePath returns the config file path without creating any directories.
// Used by read-only callers (e.g. ModTime) so that merely constructing the
// editor does not create the config directory.
func resolvePath() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, appName, FileName), nil
}
