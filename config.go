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
	//  - a map has 2 runes: one to identify the map items, the other to identify the key within an item
	//  - a slice has 1 rune to identify the slice items
	//
	// e.g. for a field is defined as
	//      Field map[int][]string `...sep=" :,"...`
	//
	//  map items are separated by a space, its key by a : and the slice items by a ,
	//  so that `key1:a,b key2:x,y` is deserialized as [key1:["a","b"] key2:["x","y"]].
	TagSepID = "sep"
)

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
	Init() error

	// Usage provides the usage message for the given config item name.
	// If the name is the empty string, then the overall usage message is expected.
	// If the returned value is empty, then the config item or subcommand is considered hidden
	// and not displayed in the flags usage message.
	Usage(name string) string
}

// FromFlags defines the interface to set values from command line flags.
type FromFlags interface {
	// FlagsDone is called once the flags have been processed
	// with the previous subcommands and the remaining arguments.
	FlagsDone(cmds []Config, args []string) error

	// FlagsShort returns the short flag for the long name.
	FlagsShort(name string) string
}

// FromEnv defines the interface to set values from environment variables.
type FromEnv interface {
	// Env returns the name of the environment variable used for the given config item.
	// Return an empty value to ignore the config item.
	Env(name string) string
}

// FromIO defines the interface to set values from an io source (typically a file).
// The supported formats are currently: ini, toml, json and yaml.
type FromIO interface {
	// Load returns the source for the data.
	Load() (io.ReadCloser, error)

	// Save returns the destination for the data.
	Save() (io.WriteCloser, error)

	// New returns a new instance of Store.
	New(seps LookupFn) Store
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
	conf, err := newConfig(config, options)
	if err != nil {
		return err
	}

	for _, s := range args {
		switch s {
		case "-h", "-help", "--help":
			conf.helpRequested = true
			break
		}
	}

	return conf.Load(args)
}

type config struct {
	helpRequested bool // If true, prevent the Init methods from being triggered.
	raw           Config
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
	prev []Config               // Previous Config items.

	options struct {
		fout   io.Writer                                // Flags usage output.
		gsep   string                                   // Grouped config items separator.
		envsep string                                   // Environment variables separator.
		fusage func(error, func(io.Writer) error) error // Called upon flags parsing error or help requested.
	}
}

func newConfig(c Config, options []Option) (*config, error) {
	root, err := structs.NewStruct(c, TagID, TagSepID)
	if err != nil {
		return nil, err
	}
	conf := newConfigFromStruct(root, c, nil)

	// User defined options.
	for _, o := range options {
		err := o(conf)
		if err != nil {
			return nil, err
		}
	}

	// Default options.
	if conf.options.fout == nil {
		conf.options.fout = os.Stderr
	}
	if conf.options.gsep == "" {
		conf.options.gsep = "-"
	}
	if conf.options.envsep == "" {
		conf.options.envsep = "_"
	}
	if conf.options.fusage == nil {
		out := conf.options.fout
		conf.options.fusage = func(err error, usage func(io.Writer) error) error {
			if err != nil {
				fmt.Fprintln(out, err)
			}
			usage(out)
			os.Exit(2)
			return nil
		}
	}

	return conf, nil
}

func newConfigFromStruct(s *structs.StructStruct, c Config, conf *config) *config {
	nconf := &config{
		raw:   c,
		root:  s,
		trans: make(map[string]string),
	}
	if conf != nil {
		nconf.options = conf.options
		nconf.prev = append(conf.prev, conf.raw)
	}
	return nconf
}

// Build the mapping of flags normalized names with their real names.
func (c *config) buildKeys(fields []*structs.StructField, section string) error {
	for _, field := range fields {
		if emb := field.Embedded(); emb != nil {
			section := c.toSection(section, emb)
			if err := c.buildKeys(emb.Fields(), section); err != nil {
				return errors.Errorf("%s: %v", field.Name(), err)
			}
			continue
		}
		name := c.toName(section, field)
		lname := strings.ToLower(name)
		if _, ok := c.trans[lname]; ok {
			return errors.Errorf("duplicate config name: %s", lname)
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
		// Prepare for the callback on the last command only.
		lastCommand := true
		defer func() {
			if err != nil || !lastCommand {
				return
			}
			err = from.FlagsDone(c.prev, c.fs.Args())
		}()

		if err := c.fs.Parse(args); err != nil {
			if err == flag.ErrHelp {
				err = nil
			}
			usage := c.buildFlagsUsage()
			return c.options.fusage(err, usage)
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
			// Maybe a new subcommand.
			sub := args[0]
			field := c.root.Lookup(sub)
			if field == nil {
				return
			}
			emb := field.Embedded()
			if emb == nil {
				return
			}
			// A subcommand must be a Config and Flags.
			conf, okc := emb.Interface().(Config)
			_, okf := emb.Interface().(FromFlags)
			if okc && okf {
				lastCommand = false
				err = newConfigFromStruct(emb, conf, c).Load(args[1:])
			}
		}()
	}

	if from, ok := c.raw.(FromEnv); ok {
		// Update the config with the env values.
		for _, name := range c.trans {
			envvar := from.Env(name)
			if envvar == "" {
				continue
			}
			v, ok := os.LookupEnv(envvar)
			if !ok {
				continue
			}
			names := c.fromNameAll(name, c.options.envsep)
			field := c.root.Lookup(names...)

			if err := field.Set(v); err != nil {
				return errors.Errorf("env %s: %v", envvar, err)
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

		store, err := ioLoad(from, lookup)
		if err != nil {
			return err
		}

		// Merge the file data with the current config items.
		if err := c.updateIO(store); err != nil {
			return err
		}

		if err := c.ioSave(store, from, lookup); err != nil {
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

// init invokes the Init method recursively on the main type
// and all the embedded ones. It stops at the first error encountered.
func (c *config) init() error {
	if c.helpRequested {
		// Skip init if help is requested.
		return nil
	}

	// Make sure to skip the embedded structs implementing Config (aka subcommands)
	// as they only get initialized if the subcommand is actually invoked.
	res, ok := callUntil(c.root, "Init", nil, callInitConfig)
	if !ok {
		return nil
	}
	return res[0].(error)
}

// callInitConfig detects an error returned by the Init method.
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
	return section + c.options.gsep + name
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
	return section + c.options.gsep + name
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
