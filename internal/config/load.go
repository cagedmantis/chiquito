package config

import (
	"bytes"
	"fmt"
	"os"
	"time"

	"github.com/BurntSushi/toml"

	"argc.dev/chiquito/internal/fileio"
)

// Load reads and parses the configuration file, layering it over the built-in
// defaults so that any field absent from the file keeps its default. If the
// file does not exist, the defaults are written to it (best effort) and
// returned. The result is always validated.
//
// It returns the resolved configuration and the path it read from.
func Load() (Config, string, error) {
	path, err := FilePath()
	if err != nil {
		return Default(), "", err
	}

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		cfg := Default()
		// Best effort: seed a default file so the user has something to edit.
		if b, mErr := Marshal(cfg); mErr == nil {
			_ = fileio.WriteAtomic(path, b)
		}
		return cfg, path, nil
	}
	if err != nil {
		return Default(), path, err
	}

	cfg, err := Parse(data)
	if err != nil {
		return Default(), path, err
	}
	return cfg, path, nil
}

// Parse decodes TOML data over the defaults and validates the result. Scalar
// fields present in the data override the defaults; the keybinding map is
// merged, so a file may override individual bindings without restating them all.
func Parse(data []byte) (Config, error) {
	cfg := Default() // start from defaults; decode overwrites present fields
	if _, err := toml.Decode(string(data), &cfg); err != nil {
		return Default(), fmt.Errorf("config: parse: %w", err)
	}
	cfg.validate()
	return cfg, nil
}

// Marshal encodes a configuration as TOML.
func Marshal(cfg Config) ([]byte, error) {
	var buf bytes.Buffer
	if err := toml.NewEncoder(&buf).Encode(cfg); err != nil {
		return nil, fmt.Errorf("config: marshal: %w", err)
	}
	return buf.Bytes(), nil
}

// Save writes the configuration to the standard path atomically.
func Save(cfg Config) error {
	path, err := FilePath()
	if err != nil {
		return err
	}
	b, err := Marshal(cfg)
	if err != nil {
		return err
	}
	return fileio.WriteAtomic(path, b)
}

// ModTime returns the modification time of the config file, used by the UI to
// detect changes for hot-reload. A missing file yields the zero time.
func ModTime() (time.Time, error) {
	path, err := resolvePath()
	if err != nil {
		return time.Time{}, err
	}
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return time.Time{}, nil
	}
	if err != nil {
		return time.Time{}, err
	}
	return info.ModTime(), nil
}

// validate repairs out-of-range or empty fields so the rest of the program can
// trust the configuration.
func (c *Config) validate() {
	if c.Editor.TabWidth < 1 {
		c.Editor.TabWidth = 4
	}
	if c.Theme.Name == "" {
		c.Theme.Name = "default"
	}
	if c.Spell.Language == "" {
		c.Spell.Language = "en_US"
	}
	if c.Keys.Bindings == nil {
		c.Keys.Bindings = DefaultKeybindings()
	}
}
