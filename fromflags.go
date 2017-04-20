package construct

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/pierrec/construct/internal/structs"
)

func (c *config) buildFlags(section string, root *structs.StructStruct) (err error) {
	var subcommands []*structs.StructField

	for _, field := range root.Fields() {
		// Skip subcommand.
		if isConfig(field) {
			subcommands = append(subcommands, field)
			continue
		}

		name := toName(section, field.Name())
		if emb := field.Embedded(); emb != nil {
			err := c.buildFlags(name, emb)
			if err != nil {
				return err
			}
			continue
		}
		lname := strings.ToLower(name)
		usage := c.usage(field.Name())

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

	// Set the usage message.
	c.fs.Usage = func() {
		var out = os.Stderr

		// Main usage.
		usage := c.raw.UsageConfig("")
		fmt.Fprintf(out, usage)

		// Options for this command.
		if usage != "" {
			fmt.Fprintf(out, "\n\n")
		}
		fmt.Fprintf(out, "Options:\n")

		tabw := tabwriter.NewWriter(out, 8, 0, 1, ' ', 0)
		c.fs.VisitAll(func(f *flag.Flag) {
			if f.Usage == "" {
				// Hidden flag.
				return
			}

			v := f.Value.(flag.Getter).Get()
			switch v.(type) {
			case bool:
				fmt.Fprintf(tabw, " -%s\t\t", f.Name)
			default:
				fmt.Fprintf(tabw, " -%s\t%T\t", f.Name, v)
			}
			fmt.Fprintf(tabw, "%s\n", f.Usage)
		})
		tabw.Flush()

		// Subcommands.
		if len(subcommands) == 0 {
			return
		}
		fmt.Fprintf(out, "\nCommands:\n")
		for _, field := range subcommands {
			root, conf := getConfig(field)
			if root == nil {
				continue
			}
			usage := conf.UsageConfig("")
			if usage == "" {
				continue
			}
			cmd := strings.ToLower(root.Name())
			fmt.Fprintf(tabw, "\t%s\t%s\n", cmd, usage)
		}
		tabw.Flush()
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
