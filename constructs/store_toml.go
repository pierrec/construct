package constructs

import (
	"io"
	"time"

	toml "github.com/pelletier/go-toml"
	"github.com/pierrec/construct"
)

var _ construct.Config = (*ConfigFileTOML)(nil)

// ConfigFileTOML implements the FromIO interface for TOML formatted files.
type ConfigFileTOML struct {
	ConfigFile `cfg:",inline"`
}

var _ construct.FromIO = (*ConfigFileTOML)(nil)

// New returns the Store for a TOML formatted file.
func (c *ConfigFileTOML) New(lookup construct.LookupFn) construct.Store {
	return NewStoreTOML(lookup)
}

// NewStoreTOML returns a Store based on the TOML format.
func NewStoreTOML(lookup construct.LookupFn) construct.Store {
	v, _ := toml.Load("")
	return &tomlStore{lookup, v}
}

var _ construct.Store = (*tomlStore)(nil)

// tomlStore wraps an toml.Toml instance to implement the construct.ConfigIO interface.
type tomlStore struct {
	lookup construct.LookupFn
	toml   *toml.TomlTree
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
		return w.ToMap(), nil
	case []*toml.TomlTree:
		l := make([]map[string]interface{}, len(w))
		for i, t := range w {
			l[i] = t.ToMap()
		}
		return l, nil
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
		seps := store.lookup(keys...)
		return marshal(store, store.marshal, keys, v, seps)
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
	nr := &reader{Reader: r}
	t, err := toml.LoadReader(nr)
	if err == nil {
		store.toml = t
	}
	return nr.read(), err
}

func (store *tomlStore) WriteTo(w io.Writer) (int64, error) {
	return store.toml.WriteTo(w)
}

func (store *tomlStore) SetComment(comment string, keys ...string) error {
	return nil
}
