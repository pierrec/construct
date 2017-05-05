package constructs

import (
	"fmt"
	"reflect"

	"github.com/pierrec/construct"
	"github.com/pierrec/construct/internal/structs"
)

func marshal(store construct.Store, marshal func([]string, interface{}) (interface{}, error),
	keys []string, v interface{}) (interface{}, error) {
	switch t := reflect.TypeOf(v); t.Kind() {
	case reflect.Slice, reflect.Array:
		value := reflect.ValueOf(v)
		if n := value.Len(); n > 0 {
			// Create of slice of items.
			// First find out the type of the items by
			// marshaling the first one, then process the rest.
			w, err := marshal(keys, value.Index(0).Interface())
			if err != nil {
				return nil, err
			}

			t := reflect.TypeOf(w)
			st := reflect.SliceOf(t)
			lst := reflect.MakeSlice(st, n, n)

			lst.Index(0).Set(reflect.ValueOf(w))
			for i := 1; i < n; i++ {
				v := value.Index(i)
				w, err := marshal(keys, v.Interface())
				if err != nil {
					return nil, err
				}
				lst.Index(i).Set(reflect.ValueOf(w))
			}
			v = lst.Interface()
		}

	case reflect.Map:
		err := marshalMap(store, marshal, keys, v)
		return nil, err

	default:
		mv, err := structs.MarshalValue(v, nil)
		if err != nil {
			return nil, err
		}
		v = fmt.Sprintf("%v", mv)
	}
	return v, nil
}

// marshalMap makes use of TOML tables by setting them with the map keys.
// v must be a valid go map.
func marshalMap(store construct.Store, marshal func([]string, interface{}) (interface{}, error),
	keys []string, v interface{}) error {
	value := reflect.ValueOf(v)
	n := value.Len()
	if n == 0 {
		// Empty map, just keep the key.
		store.Set(nil, keys...)
		return nil
	}
	mkeys := value.MapKeys()
	for i := 0; i < n; i++ {
		key := mkeys[i]
		mkey, err := marshal(keys, key.Interface())
		if err != nil {
			return err
		}
		skey := fmt.Sprintf("%v", mkey)
		nkeys := append(keys, skey)
		el := value.MapIndex(key)
		mel, err := marshal(nkeys, el.Interface())
		if err != nil {
			return err
		}
		store.Set(mel, nkeys...)
	}
	return nil
}

// unmarshalMap remarshals generically unmarshalled data map[string]interface{} items
// of type []interface into their relevant type with structs.MarshalValue.
func unmarshalMap(data map[string]interface{}) error {
	for k, v := range data {
		w, err := unmarshal(v)
		if err != nil {
			return fmt.Errorf("%s: %v", k, err)
		}
		data[k] = w
	}
	return nil
}

func unmarshal(v interface{}) (interface{}, error) {
	var err error
	switch w := v.(type) {
	case map[string]interface{}:
		err = unmarshalMap(w)
	case []interface{}:
		if len(w) == 0 {
			return "", nil
		}
		v, err = structs.MarshalValue(v, nil)
	}
	return v, err
}
