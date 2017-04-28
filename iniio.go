package construct

import (
	"fmt"
	"io"

	"github.com/pierrec/construct/internal/structs"
	ini "github.com/pierrec/go-ini"
)

// ConfigFileINI implements the FromIO interface for INI formatted files.
type ConfigFileINI struct {
	// Name of the config file.
	// If no name is specified, the file is not loaded by LoadConfig()
	// and stdout is used if Save is true.
	Name   string `ini:"-"`
	Backup string `ini:"-"`
	// Save the config file once the whole config has been loaded.
	Save bool `ini:"-"`

	configFile
}

var (
	_ FromFlags = (*ConfigFileINI)(nil)
	_ FromIO    = (*ConfigFileINI)(nil)
)

func (c *ConfigFileINI) UsageConfig(name string) string {
	return c.usageConfig(name, c.Backup)
}

func (c *ConfigFileINI) LoadConfig() (io.ReadCloser, error) { return c.loadConfig(c.Name, c.Save) }

func (c *ConfigFileINI) WriteConfig() (io.WriteCloser, error) {
	return c.writeConfig(c.Name, c.Backup, c.Save)
}

func (c *ConfigFileINI) new() configIO {
	v, _ := ini.New(ini.Comment("# "))
	return &iniIO{v}
}

var _ configIO = (*iniIO)(nil)

// iniIO wraps an ini.INI instance to implement the configIO interface.
type iniIO struct {
	*ini.INI
}

func (cio *iniIO) StructTag() string { return "ini" }

func (cio *iniIO) keys(keys []string) (section, key string) {
	switch len(keys) {
	case 0:
	case 1:
		key = keys[0]
	default:
		section = keys[0]
		key = keys[1]
	}
	return
}

func (cio *iniIO) Has(keys ...string) bool {
	return cio.INI.Has(cio.keys(keys))
}

func (cio *iniIO) Get(keys ...string) (interface{}, error) {
	return cio.INI.Get(cio.keys(keys)), nil
}

func (cio *iniIO) Set(v interface{}, keys ...string) error {
	section, key := cio.keys(keys)
	mv, err := structs.MarshalValue(v)
	if err != nil {
		return err
	}
	s := fmt.Sprintf("%v", mv)
	cio.INI.Set(section, key, s)
	return nil
}
