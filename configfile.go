package iniconfig

import (
	"io"
	"os"
)

type configFile struct{}

func (*configFile) DoFlagsConfig() {}

func (*configFile) InitConfig() error { return nil }

func (c *configFile) UsageConfig(name string) string {
	switch name {
	case "Name":
		return "config file name (default=stdout)"
	case "Save":
		return "save config to file"
	}
	return ""
}

func (c *configFile) loadConfig(name string, save bool) (io.ReadCloser, error) {
	if name == "" {
		return nil, nil
	}
	f, err := os.Open(name)
	if err != nil {
		if os.IsNotExist(err) && save {
			return nil, nil
		}
		return nil, err
	}
	return f, nil
}

func (c *configFile) writeConfig(name string, save bool) (io.WriteCloser, error) {
	if !save {
		return nil, nil
	}

	if name == "" {
		return &nopCloser{os.Stdout}, nil
	}
	return os.Create(name)
}

// Wrap the given Writer with a no-op Close method.
type nopCloser struct{ io.Writer }

func (*nopCloser) Close() error { return nil }
