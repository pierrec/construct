package construct

import (
	"fmt"
	"io"

	"github.com/pierrec/construct/internal/structs"
)

// Store defines the interface for retrieving config items stored in
// various data formats.
//
// Check the constructs package for implementations.
type Store interface {
	// Has check the existence of the key.
	Has(keys ...string) bool

	// Get retrieves the value of the given key.
	Get(keys ...string) (interface{}, error)

	// Set changes the value of the given key.
	Set(value interface{}, keys ...string) error
	//TODO SetComments(string, ...string)

	// Used when deserializing config items.
	io.ReaderFrom

	// Used when serializing config items.
	io.WriterTo

	// StructTag returns the tag id used in struct field tags for the data format.
	// Field tags set to "-" are ignored.
	StructTag() string
}

func ioLoad(from FromIO, lookup func(key ...string) []rune) (Store, error) {
	if from == nil {
		return nil, nil
	}
	src, err := from.Load()
	if err != nil {
		return nil, err
	}
	if src == nil {
		return nil, nil
	}
	defer src.Close()

	store := from.New(lookup)
	if _, err := store.ReadFrom(src); err != nil {
		return nil, err
	}
	return store, nil
}

func (c *config) ioSave(store Store, from FromIO, lookup func(key ...string) []rune) error {
	dest, err := from.Write()
	if err != nil || dest == nil {
		return err
	}
	defer dest.Close()
	if store == nil {
		store = from.New(lookup)
	}
	if err := ioEncode(store, nil, c.root); err != nil {
		return err
	}
	_, err = store.WriteTo(dest)

	return err
}

// ioEncode encodes root into the Store storage format.
func ioEncode(store Store, keys []string, root *structs.StructStruct) error {
	tag := store.StructTag()

	for _, field := range root.Fields() {
		if key := field.Tag().Get(tag); len(key) > 0 && key[0] == '-' {
			// Skip discarded fields.
			continue
		}

		ks := append(keys, field.Name())
		if emb := field.Embedded(); emb != nil {
			if emb.Inlined() {
				ks = ks[:len(ks)-1]
			}
			if err := ioEncode(store, ks, emb); err != nil {
				return err
			}
			continue
		}

		v := field.Interface()
		if err := store.Set(v, ks...); err != nil {
			return fmt.Errorf("value %v: %v", v, err)
		}
	}

	return nil
}

//TODO func (c *config) iniAddComments(ini *ini.INI) {
// 	ini.SetComments("", "", c.raw.UsageConfig(""))

// 	for _, section := range append(ini.Sections(), "") {
// 		if section != "" {
// 			usage := c.usage(section)
// 			ini.SetComments(section, "", usage)
// 		}

// 		for _, key := range ini.Keys(section) {
// 			name := toName(section, key)
// 			usage := c.usage(name)
// 			ini.SetComments(section, key, usage)
// 		}
// 	}
// }
