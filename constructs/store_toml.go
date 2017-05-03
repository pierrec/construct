package constructs

import (
	"io"
	"time"

	toml "github.com/pelletier/go-toml"
	"github.com/pierrec/construct"
	"github.com/pierrec/construct/internal/structs"
)

var _ construct.Config = (*ConfigFileTOML)(nil)

// ConfigFileTOML implements the FromIO interface for TOML formatted files.
type ConfigFileTOML struct {
	ConfigFile `cfg:",inline"`
}

var _ construct.FromIO = (*ConfigFileTOML)(nil)

func (c *ConfigFileTOML) New() construct.Store {
	v, _ := toml.Load("")
	return &tomlStore{v}
}

var _ construct.Store = (*tomlStore)(nil)

// tomlStore wraps an toml.Toml instance to implement the construct.ConfigIO interface.
type tomlStore struct {
	toml *toml.TomlTree
}

func (store *tomlStore) StructTag() string { return "toml" }

func (store *tomlStore) Has(keys ...string) bool {
	return store.toml.HasPath(keys)
}

func (store *tomlStore) Get(keys ...string) (interface{}, error) {
	v := store.toml.GetPath(keys)
	switch w := v.(type) {
	case int64, float64, string, bool, time.Time:
	case *toml.TomlTree:
		m := w.ToMap()
		return structs.MarshalValue(m)
	default:
		// Convert the value to make sure it can be Set properly.
		return structs.MarshalValue(v)
	}
	return v, nil
}

// TOML supported types:
// string, int, bool, float, datetime, array, table
//
// Strategy for marshaling:
//  - leave string, int64, bool, float64, time.Time unchanged
//  - int, int8, int16, int32 -> int64
//  - uint, uint8, uint16, uint32 -> int64
//  - float32 -> float64
//  - time.Duration -> string
//  - any map -> string
//  - any slice -> slice of marshaled items
func (store *tomlStore) marshal(keys []string, v interface{}) (interface{}, error) {
	switch w := v.(type) {
	case toml.Marshaler:
		bts, err := w.MarshalTOML()
		if err != nil {
			return nil, err
		}
		return string(bts), nil
	case int64, float64, string, bool, time.Time:
	case int:
		v = int64(w)
	case int8:
		v = int64(w)
	case int16:
		v = int64(w)
	case int32:
		v = int64(w)
	case uint:
		v = int64(w)
	case uint8:
		v = int64(w)
	case uint16:
		v = int64(w)
	case uint32:
		v = int64(w)
	case uint64:
		v = int64(w)
	case float32:
		v = float64(w)
	default:
		return marshal(store, store.marshal, keys, v)
	}
	return v, nil
}

func (store *tomlStore) Set(v interface{}, keys ...string) error {
	v, err := store.marshal(keys, v)
	if err != nil || v == nil {
		return err
	}
	store.toml.SetPath(keys, v)
	return nil
}

func (store *tomlStore) ReadFrom(r io.Reader) (int64, error) {
	t, err := toml.LoadReader(r)
	if err != nil {
		return 0, err
	}
	store.toml = t
	//TODO bytes read
	return 0, nil
}

func (store *tomlStore) WriteTo(w io.Writer) (int64, error) {
	return store.toml.WriteTo(w)
}
