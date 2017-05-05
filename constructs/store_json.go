package constructs

import (
	"encoding/json"
	"io"
	"time"

	"github.com/pierrec/construct"
	"github.com/pierrec/construct/internal/structs"
)

var _ construct.Config = (*ConfigFileJSON)(nil)

// ConfigFileJSON implements the FromIO interface for JSON formatted files.
type ConfigFileJSON struct {
	ConfigFile `cfg:",inline"`
}

var _ construct.FromIO = (*ConfigFileJSON)(nil)

func (c *ConfigFileJSON) New(lookup func(key ...string) []rune) construct.Store {
	m := make(map[string]interface{})
	return &jsonStore{lookup, m}
}

var _ construct.Store = (*jsonStore)(nil)

// jsonStore wraps json instances to implement the construct.ConfigIO interface.
type jsonStore struct {
	lookup func(key ...string) []rune
	data   map[string]interface{}
}

func (store *jsonStore) StructTag() string { return "json" }

func (store *jsonStore) Has(keys ...string) bool {
	if len(keys) == 0 {
		return false
	}
	return store.has(store.data, keys)
}

func (store *jsonStore) has(data map[string]interface{}, keys []string) bool {
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

func (store *jsonStore) Get(keys ...string) (interface{}, error) {
	if len(keys) == 0 {
		return nil, nil
	}
	return store.get(store.data, keys)
}

func (store *jsonStore) get(data map[string]interface{}, keys []string) (interface{}, error) {
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

func (store *jsonStore) Set(v interface{}, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}
	v, err := store.marshal(keys, v)
	if err != nil || v == nil {
		return err
	}
	return store.set(store.data, v, keys)
}

func (store *jsonStore) marshal(keys []string, v interface{}) (interface{}, error) {
	switch w := v.(type) {
	case json.Marshaler:
		bts, err := w.MarshalJSON()
		if err != nil {
			return nil, err
		}
		return string(bts), nil
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

func (store *jsonStore) set(data map[string]interface{}, v interface{}, keys []string) error {
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

func (store *jsonStore) ReadFrom(r io.Reader) (int64, error) {
	nr := &reader{Reader: r}
	dec := json.NewDecoder(nr)
	err := dec.Decode(&store.data)
	return nr.read(), err
}

func (store *jsonStore) WriteTo(w io.Writer) (int64, error) {
	enc := json.NewEncoder(w)
	enc.SetIndent("", " ")
	if err := enc.Encode(store.data); err != nil {
		return 0, err
	}
	return 0, nil
}
