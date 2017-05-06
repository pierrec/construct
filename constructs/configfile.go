package constructs

import (
	"io"
	"os"

	"github.com/pierrec/construct"
)

var _ construct.Config = (*ConfigFile)(nil)

// ConfigFile implements most of FromIO except New and should be embedded
// into another type that provides it.
type ConfigFile struct {
	// Name of the config file.
	// If no name is specified, the file is not loaded by LoadConfig()
	// and stdout is used if Save is true.
	Name string `ini:"-" toml:"-" json:"-" yaml:"-"`
	// Backup file extension.
	// The config file is first copied before being overwritten using this value.
	// Leave empty to disable.
	Backup string `ini:"-" toml:"-" json:"-" yaml:"-"`
	// Save the config file once the whole config has been loaded.
	Save bool `ini:"-" toml:"-" json:"-" yaml:"-"`
}

// Init initializes the ConfigFile.
func (*ConfigFile) Init() error { return nil }

// Usage returns the ConfigFile usage for each of its options.
func (c *ConfigFile) Usage(name string) string {
	switch name {
	case "Name":
		return "Config file name (default=stdout)"
	case "Save":
		return "Save the config to file"
	case "Backup":
		return "Config file backup extension (default=" + c.Backup + ")"
	}
	return ""
}

// Load returns an io.ReadCloser if the Name is set and the file exists.
func (c *ConfigFile) Load() (io.ReadCloser, error) {
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

// Write returns an io.WriteCloser if the Save flag is set to true.
// If the Name is empty, it defaults to stdout.
// If the backup extension is set, the file is first renamed with it,
// then a new one is created and returned.
func (c *ConfigFile) Write() (io.WriteCloser, error) {
	if !c.Save {
		return nil, nil
	}

	if c.Name == "" {
		return &nopCloser{os.Stdout}, nil
	}
	if c.Backup != "" {
		bname := c.Name + c.Backup
		if err := os.Rename(c.Name, bname); err != nil {
			if !os.IsNotExist(err) {
				return nil, err
			}
		}
	}
	return os.Create(c.Name)
}

// Wrap the given Writer with a no-op Close method.
type nopCloser struct{ io.Writer }

func (*nopCloser) Close() error { return nil }
