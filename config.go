package construct

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/pierrec/construct/internal/structs"
	"github.com/pkg/errors"
	flag "github.com/spf13/pflag"
)

const (
	// TagID is the struct tag name used to annotate struct fields.
	// Struct fields with tag cfg:"-" are discarded.
	// Embedded structs with tag cfg:"name" are renamed with the given name.
	TagID = "cfg"

	// TagSepID is the struct tag name used to specify separators for slice or map struct fields.
	// It is defined as a list of runes as follow:
	//  - a map has 2 runes: one to identify the map items, the other to identify the key
	//  - a slice has 1 rune to identify the slice items
	//
	// e.g. Field map[int][]int `...sep=" :,"..`
	//  means map items are separated by a space, its key by a : and the slice items by a ,
	//  so that `key1:123 key2:456` is deserialized as [key1:123 key2:456].
	TagSepID = "sep"
)

var (
	// ErrUsageRequested is to be returned when the flags usage is requested.
	ErrUsageRequested = errors.Errorf("flags usage requested")
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
// Any embedded struct is processed specifically depending on the interfaces it implements:
//  - Config: it defines a group of config items with a prefix set to the type name
//  - Config and FromFlags: it defines a subcommand, which is automatically loaded from flags.
//    Subcommands are not case sensitive.
//
// The embedded type names and field names can be overriden by a struct tag specifying the name to be used.
type Config interface {
	// Init initializes the Config struct.
	// It is automatically invoked on Config and recursively on its non subcommand embedded
	// structs until an error is encountered.
	InitConfig() error

	// UsageConfig provides the usage message for the given config item name.
	// If the name is the empty string, then the overall usage message is expected.
	// If the returned value is empty, then the config item or subcommand is considered hidden
	// and not displayed in the flags usage message.
	UsageConfig(name string) string
}

// FromFlags defines the interface to set values from command line flags.
type FromFlags interface {
	// FlagsDoneConfig is called with the remaining arguments on the last subcommand
	// once the flags have been processed.
	FlagsDoneConfig(args []string) error

	// FlagsShortConfig returns the short flag for the long name.
	FlagsShortConfig(name string) string
}

// FromEnv defines the interface to set values from environment variables.
type FromEnv interface {
	// EnvConfig returns the name of the environment variable used for the given config item.
	// Return an empty value to ignore the config item.
	EnvConfig(name string) string
}

// FromIO defines the interface to set values from an io source (typically a file).
// The supported formats are currently: ini, toml, json and yaml.
type FromIO interface {
	// LoadConfig returns the source for the data.
	LoadConfig() (io.ReadCloser, error)

	// WriteConfig returns the destination for the data.
	WriteConfig() (io.WriteCloser, error)

	// New returns a new instance of ConfigIO.
	New(seps func(key ...string) []rune) Store
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
func Load(config Config, options ...Option) error {
	args := os.Args[1:]
	if flag.Parsed() {
		// Arguments may have been parsed already, typically from go test binary.
		args = flag.Args()
	}
	return LoadArgs(config, args)
}

// LoadArgs is equivalent to Load using the given arguments.
// The first argument must be the real one, not the executable.
func LoadArgs(config Config, args []string, options ...Option) error {
	conf, err := newConfig(config)
	if err != nil {
		return err
	}

	for _, o := range options {
		err := o(conf)
		if err != nil {
			return err
		}
	}
	if conf.gsep == "" {
		conf.gsep = "-"
	}
	if conf.envsep == "" {
		conf.envsep = "_"
	}

	err = conf.Load(args)
	if err == ErrUsageRequested && conf.fs != nil {
		conf.fs.Usage()
		return nil
	}
	return err
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

	fs   *flag.FlagSet
	refs map[string]interface{} // Holds pointers of flags values.
	out  io.Writer              // Output for usage message.
	gsep string                 // Grouped config items separator.

	envsep string // Environment variables separator.
}

func newConfig(c Config) (*config, error) {
	root, err := structs.NewStruct(c, TagID, TagSepID)
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
func (c *config) buildKeys(fields []*structs.StructField, section string) error {
	for _, field := range fields {
		if emb := field.Embedded(); emb != nil {
			section := c.toSection(section, emb)
			if err := c.buildKeys(emb.Fields(), section); err != nil {
				return fmt.Errorf("%s: %v", field.Name(), err)
			}
			continue
		}
		name := c.toName(section, field)
		lname := strings.ToLower(name)
		if _, ok := c.trans[lname]; ok {
			return fmt.Errorf("duplicate config name: %s", lname)
		}
		c.trans[lname] = name
	}
	return nil
}

// Load initializes the config.
func (c *config) Load(args []string) (err error) {
	if err := c.buildKeys(c.root.Fields(), ""); err != nil {
		return err
	}

	if from, ok := c.raw.(FromFlags); ok {
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
			if len(args) > 0 {
				// Maybe a new subcommand.
				sub := strings.ToLower(args[0])
				field := c.root.Lookup(sub)
				if field == nil {
					goto flagsDone
				}
				emb := field.Embedded()
				if emb == nil {
					goto flagsDone
				}
				// A subcommand must be a Config and Flags.
				conf, okc := emb.Interface().(Config)
				_, okf := emb.Interface().(FromFlags)
				if okc && okf {
					err = newConfigFromStruct(emb, conf).Load(args[1:])
					return
				}
			}
		flagsDone:
			err = from.FlagsDoneConfig(args)
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
			names := c.fromNameAll(name, c.envsep)
			field := c.root.Lookup(names...)

			if err := field.Set(v); err != nil {
				return fmt.Errorf("env %s: %v", envvar, err)
			}
			delete(c.trans, name)
		}
	}

	if from, ok := c.raw.(FromIO); ok {
		// Load the values from the ini source.
		lookup := func(keys ...string) []rune {
			field := c.root.Lookup(keys...)
			if field == nil {
				return nil
			}
			return field.Separators()
		}

		cio, err := ioLoad(from, lookup)
		if err != nil {
			return err
		}

		if cio != nil {
			// Merge the file data with the current config items.
			for _, name := range c.trans {
				keys := c.fromNameAll(name, c.gsep)
				field := c.root.Lookup(keys...)
				if !cio.Has(keys...) {
					// Add the config item to the store for saving.
					v := field.Interface()
					if err := cio.Set(v, keys...); err != nil {
						return err
					}
					continue
				}
				v, err := cio.Get(keys...)
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
		}

		if err := c.ioSave(cio, from, lookup); err != nil {
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

// toName returns the field name.
func (c *config) toName(section string, f *structs.StructField) string {
	name := f.Name()
	if section == "" {
		return name
	}
	return section + c.gsep + name
}

// toSection returns the section name.
func (c *config) toSection(section string, s *structs.StructStruct) string {
	if s.Inlined() {
		return section
	}
	name := s.Name()
	if section == "" {
		return name
	}
	return section + c.gsep + name
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
		if c, _ := getCommand(field); c != nil {
			continue
		}
		emb := field.Embedded()
		if emb == nil {
			continue
		}
		if _, ok := emb.Interface().(Config); !ok {
			continue
		}
		res, ok := callUntil(emb, m, args, until)
		if ok && until(res) {
			return res, true
		}
	}
	return nil, false
}

// getCommand returns the struct implementing the Config and FromFlags interfaces, if any.
func getCommand(field *structs.StructField) (*structs.StructStruct, Config) {
	emb := field.Embedded()
	if emb == nil {
		return nil, nil
	}
	// A subcommand must implement Config and Flags.
	embi := emb.Interface()
	if conf, ok := embi.(Config); ok {
		if _, ok = embi.(FromFlags); ok {
			return emb, conf
		}
	}
	return nil, nil
}
