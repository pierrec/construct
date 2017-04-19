package iniconfig

import (
	"flag"
	"fmt"
	"strings"
	"time"

	"github.com/pierrec/go-iniconfig/internal/structs"
)

func (c *config) buildFlags(section string, fields []*structs.StructField) (err error) {
	for _, field := range fields {
		name := toName(section, field.Name())
		if emb := field.Embedded(); emb != nil {
			err := c.buildFlags(name, emb.Fields())
			if err != nil {
				return err
			}
			continue
		}
		lname := strings.ToLower(name)
		usage := c.usage(name)

		v := field.Value()

		// Convert lower types.
		v, err = structs.MarshalValue(v)
		if err != nil {
			return fmt.Errorf("field %s: %v", name, err)
		}

		switch w := v.(type) {
		case bool:
			c.fs.Bool(lname, w, usage)
		case time.Duration:
			c.fs.Duration(lname, w, usage)
		case float64:
			c.fs.Float64(lname, w, usage)
		case int:
			c.fs.Int(lname, w, usage)
		case int64:
			c.fs.Int64(lname, w, usage)
		case string:
			c.fs.String(lname, w, usage)
		case uint:
			c.fs.Uint(lname, w, usage)
		case uint64:
			c.fs.Uint64(lname, w, usage)
		}
	}
	return nil
}

// The flags that have been updated are removed from the map.
func (c *config) updateFlags() (err error) {
	c.fs.Visit(func(f *flag.Flag) {
		if err != nil {
			return
		}
		names := c.fromNameAll(f.Name)
		field := c.root.Lookup(names...)

		v := f.Value.(flag.Getter).Get()
		err = field.Set(v)
		if err != nil {
			err = fmt.Errorf("flag %s: %v", f.Name, err)
		}
		delete(c.trans, f.Name)
	})
	return
}
