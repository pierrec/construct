package construct

import (
	"io"
	"os"
)

// configFile is embedded into ConfigFile types to provide
// common behaviour.
// It does not implement the Config interface as it is not a subcommand.
type configFile struct{}

func (*configFile) DoFlagsConfig() {}

func (*configFile) InitConfig() error { return nil }

func (c *configFile) usageConfig(name, bak string) string {
	switch name {
	case "Name":
		return "config file name (default=stdout)"
	case "Save":
		return "save config to file"
	case "Backup":
		return "backup config file extension (default=" + bak + ")"
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

func (c *configFile) writeConfig(name string, bakExtension string, save bool) (io.WriteCloser, error) {
	if !save {
		return nil, nil
	}

	if name == "" {
		return &nopCloser{os.Stdout}, nil
	}
	if bakExtension != "" {
		bname := name + bakExtension
		if err := os.Rename(name, bname); err != nil {
			if !os.IsNotExist(err) {
				return nil, err
			}
		}
	}
	return os.Create(name)
}

// Wrap the given Writer with a no-op Close method.
type nopCloser struct{ io.Writer }

func (*nopCloser) Close() error { return nil }
