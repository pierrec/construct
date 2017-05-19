package construct

import (
	"fmt"
	"io"
	"io/ioutil"
	"reflect"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/pierrec/construct/internal/structs"
	"github.com/pkg/errors"
	flag "github.com/spf13/pflag"
)

func (c *config) buildFlags(section string, root *structs.StructStruct) error {
	if c.fs == nil {
		c.fs = flag.NewFlagSet("", flag.ContinueOnError)
		// Disable the output on error.
		c.fs.SetOutput(ioutil.Discard)
		// Make sure the parsing stops when a command is found.
		c.fs.SetInterspersed(false)
		c.refs = make(map[string]interface{})
	}

	config, ok := root.Interface().(Config)
	if !ok {
		// Skip non Config structs.
		return nil
	}
	from, isFlags := root.Interface().(FromFlags)

	for _, field := range root.Fields() {
		if c, _ := getCommand(field); c != nil {
			// Skip subcommand.
			continue
		}

		if emb := field.Embedded(); emb != nil {
			section := c.toSection(section, emb)
			err := c.buildFlags(section, emb)
			if err != nil {
				return err
			}
			continue
		}
		name := c.toName(section, field)

		// Convert lower types.
		v, err := field.MarshalValue()
		if err != nil {
			return errors.Errorf("field %s: %v", name, err)
		}
		lname := strings.ToLower(name)
		usage := config.Usage(field.Name())
		var short string
		if isFlags {
			short = from.FlagsShort(field.Name())
			short = strings.ToLower(short)
		}

		// Assign flags and keep track of the pointers of the set value.
		var ref interface{}
		switch w := v.(type) {
		case bool:
			ref = c.fs.BoolP(lname, short, w, usage)
		case time.Duration:
			ref = c.fs.DurationP(lname, short, w, usage)
		case float64:
			ref = c.fs.Float64P(lname, short, w, usage)
		case int:
			ref = c.fs.IntP(lname, short, w, usage)
		case int64:
			ref = c.fs.Int64P(lname, short, w, usage)
		case string:
			ref = c.fs.StringP(lname, short, w, usage)
		case uint:
			ref = c.fs.UintP(lname, short, w, usage)
		case uint64:
			ref = c.fs.Uint64P(lname, short, w, usage)
		}
		c.refs[lname] = ref
	}

	return nil
}

func (c *config) buildFlagsUsage() func(io.Writer) error {
	type subcommand struct {
		s *structs.StructStruct
		c Config
	}
	var subcommands []subcommand

	for _, field := range c.root.Fields() {
		s, c := getCommand(field)
		if s != nil {
			subcommands = append(subcommands, subcommand{s, c})
		}
	}

	return func(out io.Writer) (err error) {
		// Main usage.
		if usage := c.raw.Usage(""); usage != "" {
			_, err = fmt.Fprintf(out, "%s\n\n", usage)
			if err != nil {
				return err
			}
		}
		_, err = fmt.Fprintf(out, "Options:\n")
		if err != nil {
			return err
		}

		tabw := tabwriter.NewWriter(out, 8, 0, 1, ' ', 0)
		c.fs.VisitAll(func(f *flag.Flag) {
			if err != nil {
				return
			}
			if f.Usage == "" {
				// Hidden flag.
				return
			}

			refv := c.refs[f.Name]
			v := reflect.ValueOf(refv).Elem().Interface()
			short := f.Shorthand
			if short != "" {
				short = "-" + short + ", "
			}
			switch v.(type) {
			case bool:
				_, err = fmt.Fprintf(tabw, " %s\t--%s\t", short, f.Name)
			default:
				_, err = fmt.Fprintf(tabw, " %s\t--%s\t%T", short, f.Name, v)
			}
			if err == nil {
				_, err = fmt.Fprintf(tabw, "\t%s\n", f.Usage)
			}
		})
		if err != nil {
			return err
		}
		if err = tabw.Flush(); err != nil {
			return err
		}

		// Subcommands.
		if len(subcommands) > 0 {
			_, err = fmt.Fprintf(out, "\nCommands:\n")
			if err != nil {
				return err
			}
			for _, sub := range subcommands {
				usage := sub.c.Usage("")
				if usage == "" {
					// Hidden command.
					continue
				}
				cmd := strings.ToLower(sub.s.Name())
				_, err = fmt.Fprintf(tabw, "\t%s\t%s\n", cmd, usage)
				if err != nil {
					return err
				}
			}
		}

		return tabw.Flush()
	}
}

// The flags that have been updated are removed from the map.
func (c *config) updateFlags() (err error) {
	c.fs.Visit(func(f *flag.Flag) {
		if err != nil {
			return
		}
		names := c.fromNameAll(f.Name, c.options.gsep)
		field := c.root.Lookup(names...)

		// Cached references are pointers to the flag set value.
		refv := c.refs[f.Name]
		v := reflect.ValueOf(refv).Elem().Interface()
		err = field.Set(v)
		if err != nil {
			err = errors.Errorf("flag %s: %v", f.Name, err)
		}
		delete(c.trans, f.Name)
	})
	return
}
