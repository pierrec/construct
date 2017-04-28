package construct

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/pierrec/construct/internal/structs"
)

func (c *config) buildFlags(section string, root *structs.StructStruct) error {
	if c.fs == nil {
		c.fs = flag.NewFlagSet("", flag.ContinueOnError)
	}

	for _, field := range root.Fields() {
		if isConfig(field) {
			// Skip subcommand.
			continue
		}

		if emb := field.Embedded(); emb != nil {
			section := toSection(section, emb)
			err := c.buildFlags(section, emb)
			if err != nil {
				return err
			}
			continue
		}
		name := toName(section, field)
		lname := strings.ToLower(name)
		usage := c.usage(field.Name())

		v := field.Value()

		// Convert lower types.
		v, err := structs.MarshalValue(v)
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

	// Lazily set the usage message.
	c.fs.Usage = func() {
		usage := c.buildFlagsUsage()
		out := c.raw.(FromFlags).FlagsUsageConfig()
		usage(out)
	}

	return nil
}

func (c *config) buildFlagsUsage() func(io.Writer) error {
	var subcommands []*structs.StructField

	for _, field := range c.root.Fields() {
		if isConfig(field) {
			subcommands = append(subcommands, field)
		}
	}

	return func(out io.Writer) (err error) {
		if out == nil {
			out = os.Stderr
		}

		// Main usage.
		if usage := c.raw.UsageConfig(""); usage != "" {
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

			v := f.Value.(flag.Getter).Get()
			switch v.(type) {
			case bool:
				_, err = fmt.Fprintf(tabw, " -%s\t", f.Name)
			default:
				_, err = fmt.Fprintf(tabw, " -%s\t%T", f.Name, v)
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
			for _, field := range subcommands {
				root, conf := getConfig(field)
				if root == nil {
					continue
				}
				usage := conf.UsageConfig("")
				if usage == "" {
					// Hidden command.
					continue
				}
				cmd := strings.ToLower(root.Name())
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
		names := c.fromNameAll(f.Name, OptionSeparator)
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
