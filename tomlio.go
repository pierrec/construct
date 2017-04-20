package construct

import (
	"fmt"
	"io"
	"reflect"
	"time"

	toml "github.com/pelletier/go-toml"
	"github.com/pierrec/construct/internal/structs"
)

// ConfigFileTOML implements the FromIO interface for TOML formatted files.
type ConfigFileTOML struct {
	// Name of the config file.
	// If no name is specified, the file is not loaded by LoadConfig()
	// and stdout is used if Save is true.
	Name string `toml:"-"`
	// Save the config file once the whole config has been loaded.
	Save bool `toml:"-"`

	configFile
}

var (
	_ FromFlags = (*ConfigFileTOML)(nil)
	_ FromIO    = (*ConfigFileTOML)(nil)
)

func (c *ConfigFileTOML) LoadConfig() (io.ReadCloser, error) { return c.loadConfig(c.Name, c.Save) }

func (c *ConfigFileTOML) WriteConfig() (io.WriteCloser, error) { return c.writeConfig(c.Name, c.Save) }

func (c *ConfigFileTOML) new() configIO {
	v, _ := toml.Load("")
	return &tomlIO{v}
}

var _ configIO = (*tomlIO)(nil)

// tomlIO wraps an toml.Toml instance to implement the configIO interface.
type tomlIO struct {
	toml *toml.TomlTree
}

func (cio *tomlIO) Keys() []string {
	return cio.toml.Keys()
}

func (cio *tomlIO) Has(keys ...string) bool {
	return cio.toml.HasPath(keys)
}

func (cio *tomlIO) Get(keys ...string) (string, error) {
	item := cio.toml.GetPath(keys)
	v, err := structs.MarshalValue(item)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%v", v), nil
}

// TOML supported types:
// string, int, bool, float, datetime, array, table
//
// Strategy for marshaling:
//  - leave string, int64, bool, float64, time.Time unchanged
//  - int, int8, int16, int32 -> int64
//  - uint, uint8, uint16, uint32, time.Duration -> int64
//  - any map -> string
//  - any slice -> slice of marshaled items
func (cio *tomlIO) marshal(v interface{}) (interface{}, error) {
	switch w := v.(type) {
	case int64, string, bool, time.Time:
	case int:
		v = int64(w)
	case int8:
		v = int64(w)
	case int16:
		v = int64(w)
	case int32:
		v = int64(w)
	case time.Duration:
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
		case reflect.Slice:
			value := reflect.ValueOf(v)
			n := value.Len()
			if n > 0 {
				// Create of slice of items.
				// First determine the type of the items by
				// marshaling the first one, then process the rest.
				w, err := cio.marshal(value.Index(0).Interface())
				if err != nil {
					return nil, err
				}

				t := reflect.TypeOf(w)
				st := reflect.SliceOf(t)
				lst := reflect.MakeSlice(st, n, n)

				lst.Index(0).Set(reflect.ValueOf(w))
				for i := 1; i < n; i++ {
					v := value.Index(i)
					w, err := cio.marshal(v.Interface())
					if err != nil {
						return nil, err
					}
					lst.Index(i).Set(reflect.ValueOf(w))
				}
				v = lst.Interface()
			}

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

func (cio *tomlIO) Set(v interface{}, keys ...string) error {
	v, err := cio.marshal(v)
	if err != nil {
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
