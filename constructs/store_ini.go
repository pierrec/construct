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

func (c *ConfigFileINI) New() construct.Store {
	v, _ := ini.New(ini.Comment("# "))
	return &iniStore{v}
}

var _ construct.Store = (*iniStore)(nil)

// iniStore wraps an ini.INI instance to implement the construct.ConfigIO interface.
type iniStore struct {
	*ini.INI
}

func (store *iniStore) StructTag() string { return "ini" }

func (store *iniStore) keys(keys []string) (section, key string) {
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

func (store *iniStore) Has(keys ...string) bool {
	return store.INI.Has(store.keys(keys))
}

func (store *iniStore) Get(keys ...string) (interface{}, error) {
	return store.INI.Get(store.keys(keys)), nil
}

func (store *iniStore) Set(v interface{}, keys ...string) error {
	section, key := store.keys(keys)
	mv, err := structs.MarshalValue(v)
	if err != nil {
		return err
	}
	s := fmt.Sprintf("%v", mv)
	store.INI.Set(section, key, s)
	return nil
}
