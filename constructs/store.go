package constructs

import (
	"fmt"
	"io"
	"reflect"

	"github.com/pierrec/construct"
	"github.com/pierrec/construct/internal/structs"
)

// reader caches the number of bytes read.
type reader struct {
	n int64
	io.Reader
}

func (r *reader) read() int64 { return r.n }

func (r *reader) Read(b []byte) (int, error) {
	n, err := r.Reader.Read(b)
	r.n += int64(n)
	return n, err
}

// marshal makes sure the given value v is suitable for storage.
// It may update the Store directly in which case the returned value is nil.
func marshal(store construct.Store, marshal func([]string, interface{}) (interface{}, error),
	keys []string, v interface{}, seps []rune) (interface{}, error) {
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
		mv, err := structs.MarshalValue(v, seps)
		if err != nil {
			return nil, err
		}
		v = fmt.Sprintf("%v", mv)
	}
	return v, nil
}

// marshalMap populates the store with the map keys and marshaled values.
// v must be a valid go map.
func marshalMap(store construct.Store, marshal func([]string, interface{}) (interface{}, error),
	keys []string, v interface{}) error {
	value := reflect.ValueOf(v)
	n := value.Len()
	if n == 0 {
		// Empty map, just keep the key.
		zero := reflect.Zero(value.Type().Elem())
		store.Set(zero, keys...)
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
