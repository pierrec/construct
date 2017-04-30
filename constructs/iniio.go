package constructs

import (
	"fmt"

	"github.com/pierrec/construct"
	"github.com/pierrec/construct/internal/structs"
	ini "github.com/pierrec/go-ini"
)

var _ construct.Config = (*ConfigFileINI)(nil)

// ConfigFileINI implements the FromIO interface for INI formatted files.
type ConfigFileINI struct {
	ConfigFile `cfg:",inline"`
}

var _ construct.FromIO = (*ConfigFileINI)(nil)

func (c *ConfigFileINI) New() construct.ConfigIO {
	v, _ := ini.New(ini.Comment("# "))
	return &iniIO{v}
}

var _ construct.ConfigIO = (*iniIO)(nil)

// iniIO wraps an ini.INI instance to implement the construct.ConfigIO interface.
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
