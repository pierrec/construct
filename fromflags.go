package iniconfig

import (
	"encoding"
	"flag"
	"fmt"
	"strconv"
	"strings"
	"time"
)

func (c *config) buildFlags(section string, fields []*structfield) error {
	for _, field := range fields {
		name := toName(section, field.field.Name)
		if field.embedded != nil {
			err := c.buildFlags(name, field.embedded.data)
			if err != nil {
				return err
			}
			continue
		}
		lname := strings.ToLower(name)
		usage := strings.Join(c.usage(name), "\n")

		v := field.Value()
		switch v := v.(type) {
		case encoding.TextMarshaler:
			bts, err := v.MarshalText()
			if err != nil {
				return err
			}
			c.fs.String(lname, string(bts), usage)
		case bool:
			c.fs.Bool(lname, v, usage)
		case time.Duration:
			c.fs.Duration(lname, v, usage)
		case float64:
			c.fs.Float64(lname, v, usage)
		case int:
			c.fs.Int(lname, v, usage)
		case int64:
			c.fs.Int64(lname, v, usage)
		case string:
			c.fs.String(lname, v, usage)
		case uint:
			c.fs.Uint(lname, v, usage)
		case uint64:
			c.fs.Uint64(lname, v, usage)

		// Extra types.
		case float32:
			c.fs.Float64(lname, float64(v), usage)
		case int8:
			c.fs.Int64(lname, int64(v), usage)
		case int16:
			c.fs.Int64(lname, int64(v), usage)
		case int32:
			c.fs.Int64(lname, int64(v), usage)
		case uint8:
			c.fs.Uint64(lname, uint64(v), usage)
		case uint16:
			c.fs.Uint64(lname, uint64(v), usage)
		case uint32:
			c.fs.Uint64(lname, uint64(v), usage)

		// Slice types.
		case []bool:
			lst := make([]string, len(v))
			for i, v := range v {
				lst[i] = fmt.Sprintf("%v", v)
			}
			c.fs.String(lname, strings.Join(lst, sliceSeparator), usage)
		case []time.Duration:
			lst := make([]string, len(v))
			for i, v := range v {
				lst[i] = fmt.Sprintf("%v", v)
			}
			c.fs.String(lname, strings.Join(lst, sliceSeparator), usage)
		case []float64:
			lst := make([]string, len(v))
			for i, v := range v {
				lst[i] = fmt.Sprintf("%v", v)
			}
			c.fs.String(lname, strings.Join(lst, sliceSeparator), usage)
		case []int:
			lst := make([]string, len(v))
			for i, v := range v {
				lst[i] = fmt.Sprintf("%v", v)
			}
			c.fs.String(lname, strings.Join(lst, sliceSeparator), usage)
		case []int64:
			lst := make([]string, len(v))
			for i, v := range v {
				lst[i] = fmt.Sprintf("%v", v)
			}
			c.fs.String(lname, strings.Join(lst, sliceSeparator), usage)
		case []string:
			lst := make([]string, len(v))
			for i, v := range v {
				lst[i] = strconv.Quote(v)
			}
			c.fs.String(lname, strings.Join(lst, sliceSeparator), usage)
		case []uint:
			lst := make([]string, len(v))
			for i, v := range v {
				lst[i] = fmt.Sprintf("%v", v)
			}
			c.fs.String(lname, strings.Join(lst, sliceSeparator), usage)
		case []uint64:
			lst := make([]string, len(v))
			for i, v := range v {
				lst[i] = fmt.Sprintf("%v", v)
			}
			c.fs.String(lname, strings.Join(lst, sliceSeparator), usage)

		// Extra types slices.
		case []float32:
			lst := make([]string, len(v))
			for i, v := range v {
				lst[i] = fmt.Sprintf("%v", v)
			}
			c.fs.String(lname, strings.Join(lst, sliceSeparator), usage)
		case []int8:
			lst := make([]string, len(v))
			for i, v := range v {
				lst[i] = fmt.Sprintf("%v", v)
			}
			c.fs.String(lname, strings.Join(lst, sliceSeparator), usage)
		case []int16:
			lst := make([]string, len(v))
			for i, v := range v {
				lst[i] = fmt.Sprintf("%v", v)
			}
			c.fs.String(lname, strings.Join(lst, sliceSeparator), usage)
		case []int32:
			lst := make([]string, len(v))
			for i, v := range v {
				lst[i] = fmt.Sprintf("%v", v)
			}
			c.fs.String(lname, strings.Join(lst, sliceSeparator), usage)
		case []uint8:
			lst := make([]string, len(v))
			for i, v := range v {
				lst[i] = fmt.Sprintf("%v", v)
			}
			c.fs.String(lname, strings.Join(lst, sliceSeparator), usage)
		case []uint16:
			lst := make([]string, len(v))
			for i, v := range v {
				lst[i] = fmt.Sprintf("%v", v)
			}
			c.fs.String(lname, strings.Join(lst, sliceSeparator), usage)
		case []uint32:
			lst := make([]string, len(v))
			for i, v := range v {
				lst[i] = fmt.Sprintf("%v", v)
			}
			c.fs.String(lname, strings.Join(lst, sliceSeparator), usage)

		default:
			return fmt.Errorf("unsupported type %T for %s", v, name)
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
		field := c.root.Get(names...)

		v := f.Value.(flag.Getter).Get()
		err = field.Set(v)
		if err != nil {
			err = fmt.Errorf("flag %s: %v", f.Name, err)
		}
		delete(c.trans, f.Name)
	})
	return
}
