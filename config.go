// Package construct provides a simple way to load configuration into a struct
// from various sources in order of priority, overriding its default values:
//  - command line flags
//  - environment variables
//  - file in various formats
//
// The data sources are defined by implementing the relevant interfaces on the struct:
//  - FromFlags interface for command line flags
//  - FromEnv interface for environment variables
//  - FromIO interface for files
//
// Once the data is loaded from all sources, the InitConfig() method is invoked
// on the main struct as well as all the embedded ones not implementing the Config interface.
//
// Struct fields can be ignored with the tag cfg:"-" and renamed with any other value
// preceding the first coma.
//
package construct

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/pierrec/construct/internal/structs"
)

const (
	// TagID is the struct tag name used to annotate struct fields.
	// Struct fields with tag cfg:"-" are discarded.
	// Embedded structs with tag cfg:"name" are renamed with the given name.
	TagID = "cfg"

	// OptionSeparator is used to separate grouped options in command line flags.
	// Options are grouped using an embedded struct that does not implement the Config interface.
	// Embedded structs that do implement the Config interface are command line subcommands.
	OptionSeparator = "-"

	// SliceSeparator is used to separate slice items.
	SliceSeparator = ','
	sliceSeparator = string(SliceSeparator)

	// MapKeySeparator is used to separate map keys and their value.
	MapKeySeparator = ':'
	mapKeySeparator = string(MapKeySeparator)
)

var (
	// EnvSeparator is used to separate grouped options in environment variables.
	EnvSeparator = "_"
)

// Help requested on the cli.
// If set to true, it will prevent the InitConfig methods from being triggered.
var helpRequested bool

func init() {
	for _, s := range os.Args {
		switch s {
		case "-h", "-help", "--help":
			helpRequested = true
			break
		}
	}
}

// Config defines the main interface for a config struct.
// Any embedded struct is processed specifically depending if it implements Config or not:
//  - if so, it defines a Subcommand, which is automatically loaded if the subcommand is found in the flags
//    Subcommands are not case sensitive.
//  - if not, it defines a group of config items with a prefix named after the type name
//
// The embedded type and field names can be overriden by a struct tag specifying the name to be used.
type Config interface {
	DoConfig()

	// Init initializes the Config struct.
	// It is automatically invoked on Config and recursively on its embedded
	// Config structs that do not implement Config until an error is encountered.
	InitConfig() error

	// UsageConfig provides the usage message for the given option name.
	// If the name is the empty string, then the overall usage message is expected.
	// If the returned value is empty, then the option or subcommand is considered hidden
	// and not displayed in the flags usage message.
	UsageConfig(name string) string
}

// FromFlags defines the interface to set values from command line flags.
type FromFlags interface {
	// FlagsUsageConfig returns the Writer for use when the usage is requested.
	// If nil, it defaults to os.Stderr.
	FlagsUsageConfig() io.Writer
}

// FromEnv defines the interface to set values from environment variables.
type FromEnv interface {
	// EnvConfig returns the name of the environment variable used for the given option.
	// Return an empty value to ignore the option.
	EnvConfig(name string) string
}

// FromIO defines the interface to set values from an io source (typically a file).
// The supported formats are currently: ini and toml.
//TODO add support for json, yaml.
type FromIO interface {
	// LoadConfig returns the source for the ini data.
	LoadConfig() (io.ReadCloser, error)

	// WriteConfig returns the destination for the ini data.
	WriteConfig() (io.WriteCloser, error)

	New() ConfigIO
}

// Load populates the config with data from various sources.
// config must be a pointer to a struct.
//
// The values are set based on the implemented interfaces by config
// in order of priority:
//  - cli value: provided by the FromFlags interface
//  - env value: provided by the FromEnv interface
//  - ini value: provided by the FromIO interface
//  - default value: values initially set in config
func Load(config Config) error {
	args := os.Args[1:]
	if flag.Parsed() {
		// Arguments may have been parsed already, typically from go test binary.
		args = flag.Args()
	}
	return LoadArgs(config, args)
}

// LoadArgs is equivalent to Load using the given arguments.
// The first argument must be the real one, not the executable.
func LoadArgs(config Config, args []string) error {
	conf, err := newConfig(config)
	if err != nil {
		return err
	}
	return conf.Load(args)
}

// Usage writes out the config usage to the given Writer.
func Usage(config Config, out io.Writer) error {
	conf, err := newConfig(config)
	if err != nil {
		return err
	}
	if err := conf.buildFlags("", conf.root); err != nil {
		return err
	}
	usage := conf.buildFlagsUsage()

	return usage(out)
}

type config struct {
	raw Config
	// Internal reflect based representation of the struct to use as config.
	root *structs.StructStruct
	// Initially contains all the stringified keys of root.
	// The map keys are the normalized names for flags and the value the untouched names.
	// keys will be removed as they are set in order of highest priority first.
	trans map[string]string

	// Current subcommands.
	subs []string

	fs *flag.FlagSet
}

func newConfig(c Config) (*config, error) {
	root, err := structs.NewStruct(c, TagID)
	if err != nil {
		return nil, err
	}
	return &config{
		raw:   c,
		root:  root,
		trans: make(map[string]string),
	}, nil
}

func newConfigFromStruct(s *structs.StructStruct, c Config) *config {
	return &config{
		raw:   c,
		root:  s,
		trans: make(map[string]string),
	}
}

// Build the mapping of flags normalized names with their real names.
func (c *config) buildKeys(fields []*structs.StructField, section string) {
	for _, field := range fields {
		name := toName(section, field.Name())
		if emb := field.Embedded(); emb != nil {
			c.buildKeys(emb.Fields(), name)
			continue
		}
		lname := strings.ToLower(name)
		c.trans[lname] = name
	}
}

// Load initializes the config.
func (c *config) Load(args []string) (err error) {
	c.buildKeys(c.root.Fields(), "")

	if _, ok := c.raw.(FromFlags); ok {
		// Update the config with the cli values.
		if err := c.buildFlags("", c.root); err != nil {
			return err
		}

		if err := c.fs.Parse(args); err != nil {
			if err == flag.ErrHelp {
				os.Exit(0)
			}
			return err
		}

		if err := c.updateFlags(); err != nil {
			return err
		}

		// Process any subcommand.
		defer func() {
			if err != nil {
				return
			}
			args := c.fs.Args()
			if len(args) == 0 {
				return
			}
			// New subcommand.
			sub := strings.ToLower(args[0])
			c.subs = append(c.subs, sub)
			if field := c.root.Lookup(c.subs...); field != nil {
				if root, conf := getConfig(field); root != nil {
					err = newConfigFromStruct(root, conf).Load(args[1:])
					return
				}
			}
			err = fmt.Errorf("unknown subcommand %s", sub)
		}()
	}

	if from, ok := c.raw.(FromEnv); ok {
		// Update the config with the env values.
		for _, name := range c.trans {
			envvar := from.EnvConfig(name)
			if envvar == "" {
				continue
			}
			v, ok := os.LookupEnv(envvar)
			if !ok {
				continue
			}
			names := c.fromNameAll(name, EnvSeparator)
			field := c.root.Lookup(names...)

			if err := field.Set(v); err != nil {
				return fmt.Errorf("env %s: %v", envvar, err)
			}
			delete(c.trans, name)
		}
	}

	if from, ok := c.raw.(FromIO); ok {
		// Load the values from the ini source.
		cio, err := ioLoad(from)
		if err != nil {
			return err
		}

		if cio != nil {
			// Merge the file data with the current options.
			for _, name := range c.trans {
				keys := c.fromNameAll(name, OptionSeparator)
				field := c.root.Lookup(keys...)
				if !cio.Has(keys...) {
					// v := field.Value()
					//TODO cio.Set(v, keys...)
					continue
				}
				v, err := cio.Get(keys...)
				if err != nil {
					return fmt.Errorf("%s: %v", name, err)
				}

				if err := field.Set(v); err != nil {
					return err
				}
			}
		}

		if err := c.ioSave(cio, from); err != nil {
			return err
		}
	}

	return c.init()
}

// fromNameAll splits a concatenated name into all its names.
func (c *config) fromNameAll(name string, sep string) []string {
	name = strings.ToLower(name)
	return strings.Split(c.trans[name], sep)
}

// usage returns the description of the given name.
//
// It returns the first non empty result from the UsageConfig recursive method calls.
func (c *config) usage(name string) string {
	res, ok := callUntil(c.root, "UsageConfig", []interface{}{name}, callUsageConfig)
	if !ok {
		return ""
	}
	return res[0].(string)
}

func callUsageConfig(in []interface{}) bool {
	s := in[0].(string)
	return s != ""
}

// init invokes the InitConfig method recursively on the main type
// and all the embedded ones. It stops at the first error encountered.
func (c *config) init() error {
	if helpRequested {
		// Skip init if help is requested.
		return nil
	}

	// Make sure to skip the embedded structs implementing Config (aka subcommands)
	// as they only get initialized if the subcommand is actually invoked.
	res, ok := callUntil(c.root, "InitConfig", nil, callInitConfig)
	if !ok {
		return nil
	}
	return res[0].(error)
}

// callInitConfig detects an error returned by the InitConfig method.
func callInitConfig(in []interface{}) bool {
	err, ok := in[0].(error)
	return ok && err != nil
}

// toName concatenates 2 names.
func toName(a, b string) string {
	if a == "" {
		return b
	}
	return a + OptionSeparator + b
}

// callUntil recursively calls the given method m with arguments args
// on the StructStructs until the until function returns true.
// Fields matching the Config interface are ignored.
func callUntil(s *structs.StructStruct, m string, args []interface{},
	until func([]interface{}) bool) ([]interface{}, bool) {
	res, ok := s.Call(m, args)
	if ok && until(res) {
		return res, true
	}
	for _, field := range s.Fields() {
		if isConfig(field) {
			continue
		}
		emb := field.Embedded()
		if emb == nil {
			continue
		}
		res, ok := callUntil(emb, m, args, until)
		if ok && until(res) {
			return res, true
		}
	}
	return nil, false
}

// isConfig returns whether or not the field implements the Config interface.
func isConfig(field *structs.StructField) bool {
	if field == nil {
		return false
	}
	_, ok := field.PtrValue().(Config)
	return ok && field.Embedded() != nil
}

// getConfig returns the struct implementing the Config interface, if any.
func getConfig(field *structs.StructField) (*structs.StructStruct, Config) {
	if nc, ok := field.PtrValue().(Config); ok {
		if emb := field.Embedded(); emb != nil {
			return emb, nc
		}
	}
	return nil, nil
}
