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

	// SetComment defines the comment for the given key.
	SetComment(comment string, keys ...string) error

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

func ioComment(conf Config, store Store, keys ...string) error {
	name := keys[len(keys)-1]
	if comment := conf.Usage(name); comment != "" {
		return store.SetComment(comment, keys...)
	}
	return nil
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

	// Global comment.
	if err := ioComment(c.raw, store, "", ""); err != nil {
		return err
	}

	if err := ioEncode(c.raw, store, nil, c.root); err != nil {
		return err
	}
	_, err = store.WriteTo(dest)

	return err
}

// ioEncode encodes root into the Store storage format.
func ioEncode(conf Config, store Store, keys []string, root *structs.StructStruct) error {
	tag := store.StructTag()

	for _, field := range root.Fields() {
		if key := field.Tag().Get(tag); len(key) > 0 && key[0] == '-' {
			// Skip discarded fields.
			continue
		}
		if c, _ := getCommand(field); c != nil {
			// Do not save subcommands.
			continue
		}

		key := field.Name()
		ks := append(keys, key)
		if emb := field.Embedded(); emb != nil {
			if emb.Inlined() {
				ks = ks[:len(ks)-1]
			}
			conf := emb.Interface().(Config)
			if err := ioEncode(conf, store, ks, emb); err != nil {
				return err
			}
			continue
		}

		v := field.Interface()
		if err := store.Set(v, ks...); err != nil {
			return fmt.Errorf("value %v: %v", v, err)
		}

		if err := ioComment(conf, store, ks...); err != nil {
			return err
		}
	}

	return nil
}

func (c *config) updateIO(store Store) error {
	if store == nil {
		return nil
	}

	for _, name := range c.trans {
		keys := c.fromNameAll(name, c.options.gsep)
		field := c.root.Lookup(keys...)
		if !store.Has(keys...) {
			// Add the config item to the store for saving.
			v := field.Interface()
			if err := store.Set(v, keys...); err != nil {
				return err
			}

			continue
		}
		v, err := store.Get(keys...)
		if err != nil {
			return fmt.Errorf("%s: %v", name, err)
		}

		if v != nil {
			// Convert the value to make sure it can be Set properly.
			v, err = structs.MarshalValue(v, field.Separators())
			if err != nil {
				return fmt.Errorf("%s: %v", name, err)
			}
		}

		if err := field.Set(v); err != nil {
			return err
		}
	}
	return nil
}
