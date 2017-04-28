package construct

import (
	"fmt"
	"io"

	"github.com/pierrec/construct/internal/structs"
)

// configIO defines the interface for retrieving options stored in
// various data formats (ini, toml...).
type configIO interface {
	Has(...string) bool
	Get(...string) (interface{}, error)
	Set(value interface{}, keys ...string) error
	//TODO SetComments(string, ...string)

	io.ReaderFrom
	io.WriterTo

	StructTag() string
}

func ioLoad(from FromIO) (configIO, error) {
	if from == nil {
		return nil, nil
	}
	src, err := from.LoadConfig()
	if err != nil {
		return nil, err
	}
	if src == nil {
		return nil, nil
	}
	defer src.Close()

	cio := from.new()
	if _, err := cio.ReadFrom(src); err != nil {
		return nil, err
	}
	return cio, nil
}

func (c *config) ioSave(cio configIO, from FromIO) error {
	dest, err := from.WriteConfig()
	if err != nil || dest == nil {
		return err
	}
	defer dest.Close()
	if cio == nil {
		cio = from.new()
	}
	if err := ioEncode(cio, nil, c.root); err != nil {
		return err
	}
	_, err = cio.WriteTo(dest)

	return err
}

// ioEncode encodes root into the configIO storage format.
func ioEncode(cio configIO, keys []string, root *structs.StructStruct) error {
	tag := cio.StructTag()

	for _, field := range root.Fields() {
		if field.Tag().Get(tag) == "-" {
			// Skip discarded fields.
			continue
		}

		ks := append(keys, field.Name())
		if emb := field.Embedded(); emb != nil {
			if err := ioEncode(cio, ks, emb); err != nil {
				return err
			}
			continue
		}

		v := field.Value()
		if err := cio.Set(v, ks...); err != nil {
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
