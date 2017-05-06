package constructs

import (
	"bytes"
	"io"
	"time"

	"github.com/pierrec/construct"
	"github.com/pierrec/construct/internal/structs"
	yaml "gopkg.in/yaml.v2"
)

var _ construct.Config = (*ConfigFileYAML)(nil)

// ConfigFileYAML implements the FromIO interface for JSON formatted files.
type ConfigFileYAML struct {
	ConfigFile `cfg:",inline"`
}

var _ construct.FromIO = (*ConfigFileYAML)(nil)

// New returns the Store for a YAML formatted file.
func (c *ConfigFileYAML) New(lookup func(key ...string) []rune) construct.Store {
	m := make(map[string]interface{})
	return &yamlStore{lookup, m}
}

var _ construct.Store = (*yamlStore)(nil)

// yamlStore wraps json instances to implement the construct.ConfigIO interface.
type yamlStore struct {
	lookup func(key ...string) []rune
	data   map[string]interface{}
}

func (store *yamlStore) StructTag() string { return "json" }

func (store *yamlStore) Has(keys ...string) bool {
	if len(keys) == 0 {
		return false
	}
	return store.has(store.data, keys)
}

func (store *yamlStore) has(data map[string]interface{}, keys []string) bool {
	key := keys[0]
	v, ok := data[key]
	if len(keys) == 1 || !ok {
		return ok
	}
	if data, ok := v.(map[string]interface{}); ok {
		return store.has(data, keys[1:])
	}
	return false
}

func (store *yamlStore) Get(keys ...string) (interface{}, error) {
	if len(keys) == 0 {
		return nil, nil
	}
	return store.get(store.data, keys)
}

func (store *yamlStore) get(data map[string]interface{}, keys []string) (interface{}, error) {
	key := keys[0]
	v, ok := data[key]
	if len(keys) == 1 || !ok {
		return v, nil
	}
	if data, ok := v.(map[string]interface{}); ok {
		return store.get(data, keys[1:])
	}
	return nil, nil
}

func (store *yamlStore) Set(v interface{}, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}
	v, err := store.marshal(keys, v)
	if err != nil || v == nil {
		return err
	}
	return store.set(store.data, v, keys)
}

func (store *yamlStore) marshal(keys []string, v interface{}) (interface{}, error) {
	switch w := v.(type) {
	case yaml.Marshaler:
		return w.MarshalYAML()
	case string, bool,
		int, int8, int16, int32,
		uint, uint8, uint16, uint32, uint64,
		float32, float64:
	case time.Time, time.Duration:
		return structs.MarshalValue(v, nil)
	default:
		seps := store.lookup(keys...)
		return marshal(store, store.marshal, keys, v, seps)
	}
	return v, nil
}

func (store *yamlStore) set(data map[string]interface{}, v interface{}, keys []string) error {
	key := keys[0]
	if len(keys) == 1 {
		data[key] = v
		return nil
	}
	val := data[key]
	if data, ok := val.(map[string]interface{}); ok {
		return store.set(data, v, keys[1:])
	}
	m := make(map[string]interface{})
	data[key] = m
	return store.set(m, v, keys[1:])
}

func (store *yamlStore) ReadFrom(r io.Reader) (n int64, err error) {
	buf := new(bytes.Buffer)
	n, err = io.Copy(buf, r)
	if err != nil {
		return
	}
	err = yaml.Unmarshal(buf.Bytes(), store.data)
	if err != nil {
		return
	}
	return
}

func (store *yamlStore) WriteTo(w io.Writer) (int64, error) {
	bts, err := yaml.Marshal(store.data)
	if err != nil {
		return 0, err
	}
	r := bytes.NewReader(bts)
	return io.Copy(w, r)
}

func (store *yamlStore) SetComment(comment string, keys ...string) error {
	return nil
}
