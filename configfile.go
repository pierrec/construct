package iniconfig

import (
	"io"
	"os"
)

// ConfigINIFile can be embedded for automatically dealing with ini config files.
type ConfigINIFile struct {
	// Name of the config file.
	// If no name is specified, the file is not loaded by LoadConfig()
	// and stdout is used if Save is true.
	Name string `ini:"-"`
	// Save the config file once the whole config has been loaded.
	Save bool `ini:"-"`
}

var (
	_ Config    = (*ConfigINIFile)(nil)
	_ FromFlags = (*ConfigINIFile)(nil)
	_ FromIni   = (*ConfigINIFile)(nil)
)

// FlagsConfig makes ConfigINIFile implement FromFlags.
func (*ConfigINIFile) FlagsConfig() {}

// InitConfig makes ConfigINIFile implement Config.
func (*ConfigINIFile) InitConfig() error { return nil }

// UsageConfig provides the command line flags usage.
func (c *ConfigINIFile) UsageConfig(name string) string {
	switch name {
	case "configfile-name":
		return "config file name (default=stdout)"
	case "configfile-save":
		return "save config to file"
	}
	return ""
}

// LoadConfig opens the config file for loading if the name is not empty.
func (c *ConfigINIFile) LoadConfig() (io.ReadCloser, error) {
	if c.Name == "" {
		return nil, nil
	}
	f, err := os.Open(c.Name)
	if err != nil {
		if os.IsNotExist(err) && c.Save {
			return nil, nil
		}
		return nil, err
	}
	return f, nil
}

// WriteConfig opens the config file for saving if the save flag is active.
// If the name is empty, the config file is written to stdout.
func (c *ConfigINIFile) WriteConfig() (io.WriteCloser, error) {
	if !c.Save {
		return nil, nil
	}

	if c.Name == "" {
		return &nopCloser{os.Stdout}, nil
	}
	return os.Create(c.Name)
}

// Wrap the given Writer with a no-op Close method.
type nopCloser struct{ io.Writer }

func (*nopCloser) Close() error { return nil }
