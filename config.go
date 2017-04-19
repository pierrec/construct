// Package iniconfig provides a simple way to load configuration into a struct
// from various sources in order of priority, overriding its default values:
//  - command line flags
//  - environment variables
//  - ini file
//
// The sources to load data from are determined by implementing the relevant
// interfaces on the struct:
//  - FromFlags interface for command line flags
//  - FromEnv interface for environment variables
//  - FromIni interface for ini file
//
// Once the data is loaded from all sources, the Init() method is invoked
// on the main struct as well as all the embedded ones that implement the
// Config interface.
//
// Struct fields can be ignored with the tag cfg:"-"
//
package iniconfig

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/pierrec/go-iniconfig/internal/structs"
)

const (
	configTagID = "cfg"
	// OptionSeparator is used to separate an ini section name with a section key
	// for command line flags.
	OptionSeparator = "-"

	// SliceSeparator is used to separate slice items.
	SliceSeparator = ','
	sliceSeparator = string(SliceSeparator)

	// MapKeySeparator is used to separate map keys and their value.
	MapKeySeparator = ':'
	mapKeySeparator = string(MapKeySeparator)
)

// Help requested on the cli.
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

// Config defines the interface to set values from command line flags.
// Subcommands are defined by embedding Config structs.
type Config interface {
	// Init initializes the Config struct.
	// It is automatically invoked on Config and recursively on its embedded Config structs.
	InitConfig() error

	// UsageConfig provides the usage message for the given option name.
	// If the name is the empty string, then the overall usage message is expected.
	// If the returned message is empty, then the option is hidden.
	UsageConfig(name string) string
}

// FromFlags defines the interface to set values from command line flags.
type FromFlags interface {
	FlagsConfig()
}

// FromEnv defines the interface to set values from environment variables.
type FromEnv interface {
	// EnvConfig returns the name of the environment variable used for
	// the given option name.
	// The name is in lowercase.
	EnvConfig(name string) string
}

// FromIni defines the interface to set values from an ini source.
type FromIni interface {
	// LoadConfig returns the source for the ini data.
	LoadConfig() (io.ReadCloser, error)

	// WriteConfig returns the destination for the ini data.
	WriteConfig() (io.WriteCloser, error)
}

// Load populates the config with data from various sources.
// config must be a pointer to a struct.
//
// The values are set based on the implemented interfaces by config
// in the following order (last one wins):
//  - default value: values initially set in config
//  - ini value: provided by the FromIni interface
//  - env value: provided by the FromEnv interface
//  - cli value: provided by the FromFlags interface
func Load(config Config) error {
	return load(config, os.Args)
}

func load(config Config, args []string) error {
	conf, err := newConfig(config)
	if err != nil {
		return err
	}
	return conf.Load(args)
}

type config struct {
	raw Config
	// reflect based representation of the struct to use as config.
	root *structs.StructStruct
	// set of lowercased normalized names and the non lowercased ones.
	// keys will be removed as they are set in order of highest priority first.
	trans map[string]string

	// Current subcommands.
	subs []string

	fs *flag.FlagSet
}

func newConfig(c Config) (*config, error) {
	root, err := structs.NewStruct(c, configTagID)
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

// Build the mapping of lowercase names with their real names.
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
		c.fs = flag.NewFlagSet("", flag.ContinueOnError)

		if err := c.buildFlags("", c.root.Fields()); err != nil {
			return err
		}

		if err := c.fs.Parse(args[1:]); err != nil {
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
			sub := args[0]
			c.subs = append(c.subs, sub)
			if field := c.root.Lookup(c.subs...); field != nil {
				if root, conf := getConfig(field); root != nil {
					err = newConfigFromStruct(root, conf).Load(args)
					return
				}
			}
			err = fmt.Errorf("unknown subcommand %s", sub)
		}()
	}

	if env, ok := c.raw.(FromEnv); ok {
		// Update the config with the env values.
		for _, name := range c.trans {
			envvar := env.EnvConfig(name)
			v, ok := os.LookupEnv(envvar)
			if !ok {
				continue
			}
			names := c.fromNameAll(name)
			field := c.root.Lookup(names...)

			if err := field.Set(v); err != nil {
				return fmt.Errorf("env %s: %v", envvar, err)
			}
			delete(c.trans, name)
		}
	}

	if cini, ok := c.raw.(FromIni); ok {
		// Load the values from the ini source.
		inic, err := iniLoad(cini)
		if err != nil {
			return err
		}

		if inic == nil {
			// No ini file, create one.
			inic, err = newIni()
			if err != nil {
				return err
			}
			if err := inic.Encode(c.raw); err != nil {
				return err
			}
			c.iniAddComments(inic)
		} else {
			// Load the ini file data.
			for _, name := range c.trans {
				section, key := c.fromName(name)
				if !inic.Has(section, key) {
					continue
				}
				var field *structs.StructField
				if section == "" {
					field = c.root.Lookup(key)
				} else {
					field = c.root.Lookup(section, key)
				}
				v := inic.Get(section, key)

				if err := field.Set(v); err != nil {
					return err
				}
			}
		}

		// Remove hidden options.
		c.iniRemoveHidden(inic)

		if err := iniSave(inic, cini); err != nil {
			return err
		}
	}

	return c.init()
}

// fromNameAll splits a concatenated name into all its names.
func (c *config) fromNameAll(name string) []string {
	name = strings.ToLower(name)
	return strings.Split(c.trans[name], OptionSeparator)
}

// fromName split a concatenated name into its first name the rest.
func (c *config) fromName(name string) (string, string) {
	name = strings.ToLower(name)
	lst := strings.SplitN(c.trans[name], OptionSeparator, 2)
	if len(lst) == 2 {
		return lst[0], lst[1]
	}
	return "", lst[0]
}

// usage returns the description of the given name.
//
// It returns the first non empty result from the UsageConfig method.
func (c *config) usage(name string) string {
	lname := strings.ToLower(name)
	return c.raw.UsageConfig(lname)
}

func (c *config) init() error {
	// Skip init if help is requested.
	if helpRequested {
		return nil
	}

	res, ok := callUntil(c.root, "InitConfig", nil, callInitConfig)
	if !ok {
		return nil
	}
	return res[0].(error)
}

func callInitConfig(in []interface{}) bool {
	err, ok := in[0].(error)
	return ok && err != nil
}

// toName concatenates 2 names.
func toName(section, key string) string {
	name := key
	if section != "" {
		name = fmt.Sprintf("%s%s%s", section, OptionSeparator, key)
	}
	return name
}

func callUntil(s *structs.StructStruct, m string, args []interface{}, until func([]interface{}) bool) ([]interface{}, bool) {
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
		res, ok := emb.CallUntil(m, args, until)
		if ok && until(res) {
			return res, true
		}
	}
	return nil, false
}

func isConfig(field *structs.StructField) bool {
	if field == nil {
		return false
	}
	_, ok := field.PtrValue().(Config)
	return ok
}

func getConfig(field *structs.StructField) (*structs.StructStruct, Config) {
	if nc, ok := field.PtrValue().(Config); ok {
		if emb := field.Embedded(); emb != nil {
			return emb, nc
		}
	}
	return nil, nil
}
