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
	"path/filepath"
	"strings"
)

const (
	// OptionSeparator is used to separate an ini section name with a section key
	// for command line flags.
	OptionSeparator = "-"

	// SliceSeparator is the rune used to separate slice items.
	SliceSeparator = ','
	sliceSeparator = string(SliceSeparator)
)

// Config defines the interface to set values from command line flags.
type Config interface {
	// Init initializes the Config struct.
	// It is automatically invoked on Config and recursively on its embedded Config structs.
	InitConfig() error

	// UsageConfig returns the text to describe the Config.
	UsageConfig() []string

	// OptionUsageConfig provides the usage message for the given option.
	// The name is in lowercase. It may contain an OptionSeparator if part of an embedded struct.
	OptionUsageConfig(name string) []string
}

// FromFlags defines the interface to set values from command line flags.
type FromFlags interface {
	// FlagsUsageConfig returns the text used for each flag usage.
	FlagsUsageConfig() []string
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
	conf, err := newConfig(config)
	if err != nil {
		return err
	}
	return conf.Load()
}

type config struct {
	raw Config
	// reflect based representation of the struct to use as config.
	root *structs
	// set of lowercased normalized names and the non lowercased ones.
	// keys will be removed as they are set in order of highest priority first.
	trans map[string]string

	fs *flag.FlagSet
}

func newConfig(c Config) (*config, error) {
	root, err := newStructs(c)
	if err != nil {
		return nil, err
	}
	return &config{
		raw:   c,
		root:  root,
		trans: make(map[string]string),
	}, nil
}

// Build the mapping of lowercase names with their real names.
func (c *config) buildKeys(fields []*structfield, section string) {
	for _, field := range fields {
		name := toName(section, field.field.Name)
		if field.embedded != nil {
			c.buildKeys(field.embedded.data, name)
			continue
		}
		lname := strings.ToLower(name)
		c.trans[lname] = name
	}
}

// Load initializes the config.
func (c *config) Load() error {
	c.buildKeys(c.root.Fields(), "")

	if cli, ok := c.raw.(FromFlags); ok {
		// Update the config with the cli values.
		c.fs = flag.NewFlagSet("", flag.ContinueOnError)

		c.fs.Usage = func() {
			var out = os.Stderr
			if usage := cli.FlagsUsageConfig(); len(usage) == 0 {
				name := filepath.Base(os.Args[0])
				fmt.Fprintf(out, "Usage of %s:\n", name)
			} else {
				fmt.Fprint(out, strings.Join(usage, "\n"), "\n")
			}
			c.fs.SetOutput(out)
			c.fs.PrintDefaults()
		}

		if err := c.buildFlags("", c.root.Fields()); err != nil {
			return err
		}

		if err := c.fs.Parse(os.Args[1:]); err != nil {
			return err
		}

		if err := c.updateFlags(); err != nil {
			return err
		}
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
			field := c.root.Get(names...)

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
			inic, err := newIni()
			if err != nil {
				return err
			}
			if err := inic.Encode(c.raw); err != nil {
				return err
			}
			c.iniAddComments(inic)
			if err := iniSave(inic, cini); err != nil {
				return err
			}
		} else {
			// Load the ini file data.
			for _, name := range c.trans {
				section, key := c.fromName(name)
				if !inic.Has(section, key) {
					continue
				}
				field := c.root.Get(section, key)
				v := inic.Get(section, key)

				if err := field.Set(v); err != nil {
					return err
				}
			}

			if err := iniSave(inic, cini); err != nil {
				return err
			}
		}
	}

	return c.init()
}

// fromNameAll splits a concatenated name into all its names.
func (c *config) fromNameAll(name string) []string {
	return strings.Split(c.trans[name], OptionSeparator)
}

// fromName split a concatenated name into its first name the rest.
func (c *config) fromName(name string) (string, string) {
	lst := strings.SplitN(c.trans[name], OptionSeparator, 2)
	if len(lst) == 2 {
		return lst[0], lst[1]
	}
	return "", lst[0]
}

// usage returns the description of the given name.
//
// It returns the first non empty result from the OptionUsageConfig method.
func (c *config) usage(name string) []string {
	lname := strings.ToLower(name)
	res, ok := c.root.CallFirst("OptionUsageConfig", []interface{}{lname}, callSliceStringMethod)
	if !ok {
		return nil
	}
	return res[0].([]string)
}

// in must be a slice of strings.
func callSliceStringMethod(in []interface{}) bool {
	if len(in) != 1 {
		return false
	}
	res := in[0].([]string)
	return len(res) > 0
}

func (c *config) init() error {
	res, ok := c.root.CallFirst("Init", nil, func(in []interface{}) bool {
		err, ok := in[0].(error)
		return ok && err != nil
	})
	if !ok {
		return nil
	}
	return res[0].(error)
}

// toName concatenates 2 names.
func toName(section, key string) string {
	name := key
	if section != "" {
		name = fmt.Sprintf("%s%s%s", section, OptionSeparator, key)
	}
	return name
}
