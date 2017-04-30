package constructs

import (
	"fmt"
	"io"
	"reflect"
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

func (c *ConfigFileTOML) New() construct.ConfigIO {
	v, _ := toml.Load("")
	return &tomlIO{v}
}

var _ construct.ConfigIO = (*tomlIO)(nil)

// tomlIO wraps an toml.Toml instance to implement the construct.ConfigIO interface.
type tomlIO struct {
	toml *toml.TomlTree
}

func (cio *tomlIO) StructTag() string { return "toml" }

func (cio *tomlIO) Has(keys ...string) bool {
	return cio.toml.HasPath(keys)
}

func (cio *tomlIO) Get(keys ...string) (interface{}, error) {
	v := cio.toml.GetPath(keys)
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
//  - time.Duration -> string
//  - any map -> string
//  - any slice -> slice of marshaled items
func (cio *tomlIO) marshal(keys []string, v interface{}) (interface{}, error) {
	switch w := v.(type) {
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
	default:
		switch t := reflect.TypeOf(v); t.Kind() {
		case reflect.Slice, reflect.Array:
			value := reflect.ValueOf(v)
			if n := value.Len(); n > 0 {
				// Create of slice of items.
				// First find out the type of the items by
				// marshaling the first one, then process the rest.
				w, err := cio.marshal(keys, value.Index(0).Interface())
				if err != nil || w == nil {
					return nil, err
				}

				t := reflect.TypeOf(w)
				st := reflect.SliceOf(t)
				lst := reflect.MakeSlice(st, n, n)

				lst.Index(0).Set(reflect.ValueOf(w))
				for i := 1; i < n; i++ {
					v := value.Index(i)
					w, err := cio.marshal(keys, v.Interface())
					if err != nil || w == nil {
						return nil, err
					}
					lst.Index(i).Set(reflect.ValueOf(w))
				}
				v = lst.Interface()
			}

		case reflect.Map:
			err := cio.marshalMap(keys, v)
			return nil, err

		default:
			mv, err := structs.MarshalValue(v)
			if err != nil {
				return nil, err
			}
			v = fmt.Sprintf("%v", mv)
		}
	}
	return v, nil
}

// marshalMap makes use of TOML tables by setting them with the map keys.
// v must be a valid go map.
func (cio *tomlIO) marshalMap(keys []string, v interface{}) error {
	value := reflect.ValueOf(v)
	n := value.Len()
	if n == 0 {
		// Empty map, just keep the key.
		cio.toml.SetPath(keys, nil)
		return nil
	}
	mkeys := value.MapKeys()
	for i := 0; i < n; i++ {
		key := mkeys[i]
		mkey, err := cio.marshal(keys, key.Interface())
		if err != nil {
			return err
		}
		skey := fmt.Sprintf("%v", mkey)
		nkeys := append(keys, skey)
		if mkey == nil {
			cio.toml.SetPath(nkeys, nil)
			continue
		}
		el := value.MapIndex(key)
		mel, err := cio.marshal(nkeys, el.Interface())
		if err != nil {
			return err
		}
		cio.toml.SetPath(nkeys, mel)
	}
	return nil
}

func (cio *tomlIO) Set(v interface{}, keys ...string) error {
	v, err := cio.marshal(keys, v)
	if err != nil || v == nil {
		return err
	}
	cio.toml.SetPath(keys, v)
	return nil
}

func (cio *tomlIO) ReadFrom(r io.Reader) (int64, error) {
	t, err := toml.LoadReader(r)
	if err != nil {
		return 0, err
	}
	cio.toml = t
	//TODO bytes read
	return 0, nil
}

func (cio *tomlIO) WriteTo(w io.Writer) (int64, error) {
	return cio.toml.WriteTo(w)
}
