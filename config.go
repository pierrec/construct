// Package iniconfig provides a simple way to load configuration from various
// standard inputs (file, command line flags and environment variables) while
// providing default values.
package iniconfig

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	ini "github.com/pierrec/go-ini"
	"github.com/spf13/pflag"
)

// FlagSeparator is used to separate an ini section name with a section key
// for command line flags.
const FlagSeparator = '-'

// FromFlag defines the interface to set values from command line flags.
type FromFlag interface {
	// UsageConfig returns the text to be used before listing the usage
	// of every flag.
	UsageConfig() string

	// FlagUsageConfig provides the usage message for the given option.
	// The name is in lowercase. It may contain a FlagSeparator if part
	// of an embedded struct.
	FlagUsageConfig(name string) string
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
	WriteConfig(*ini.INI) (io.WriteCloser, error)
}

// Load populates the config with data from various sources.
// config must be a pointer to a struct.
//
// The values are set based on the implemented interfaces by config
// in the following order:
//  - default value: values initially set in config
//  - ini value: provided by the FromIni interface
//  - cli value: provided by the FromFlag interface
//  - env value: provided by the FromEnv interface
func Load(config interface{}, options ...ini.Option) (err error) {
	// Command line, ini and env sources.
	cli, hasCli := config.(FromFlag)
	cini, hasIni := config.(FromIni)
	env, hasEnv := config.(FromEnv)

	if !hasCli && !hasIni && !hasEnv {
		// Nothing to do...
		return nil
	}

	// inic will contain all the config.
	inic, err := ini.New(options...)
	if err != nil {
		return err
	}

	// Initialize the config with the default values.
	if err := inic.Encode(config); err != nil {
		return err
	}

	// Cache the sections and keys since new ones are not used anyway.
	sections := append(inic.Sections(), "")
	sectionkeys := make([][]string, len(sections))
	for i, section := range sections {
		sectionkeys[i] = inic.Keys(section)
	}

	if hasCli || hasIni {
		// Define the FlagSet corresponding to the config.
		fs := pflag.NewFlagSet("", pflag.ContinueOnError)

		for i, section := range sections {
			for _, key := range sectionkeys[i] {
				v := inic.Get(section, key)
				name := toName(section, key)
				var usage string
				if hasCli {
					usage = cli.FlagUsageConfig(name)
				}
				fs.String(name, v, usage)
			}
		}
		if hasCli {
			fs.Usage = func() {
				usage := cli.UsageConfig()
				if usage == "" {
					name := filepath.Base(os.Args[0])
					usage = fmt.Sprintf("Usage of %s:\n", name)
				}
				usage += fs.FlagUsages()
				fmt.Fprint(os.Stderr, usage)
			}
		}

		if err := fs.Parse(os.Args[1:]); err != nil {
			return err
		}
		if hasCli {
			// Update the config with the cli values.
			cliUpdate(inic, fs)

			// Update the config as the FromIni methods may use
			// values set by the cli.
			if err := inic.Decode(config); err != nil {
				return err
			}
		}

		if hasIni {
			// Load the values from the ini source.
			if err := iniLoad(inic, cini); err != nil {
				return err
			}

			// Save the config.
			defer func() { err = iniSave(inic, cini) }()

			if hasCli {
				// Update the config with the cli values again
				// as they overwrite the ones from the config source.
				cliUpdate(inic, fs)
			}
		}
	}

	// Update the config with the env values.
	if hasEnv {
		for i, section := range sections {
			for _, key := range sectionkeys[i] {
				name := toName(section, key)
				env := env.EnvConfig(name)
				if env == "" {
					continue
				}
				if v, ok := os.LookupEnv(env); ok {
					inic.Set(section, key, v)
				}
			}
		}
	}

	// Update the input with the final values.
	return inic.Decode(config)
}

func toName(section, key string) string {
	name := key
	if section != "" {
		name = fmt.Sprintf("%s%c%s", section, FlagSeparator, key)
	}
	return strings.ToLower(name)
}

func iniLoad(inic *ini.INI, s FromIni) error {
	src, err := s.LoadConfig()
	if err != nil {
		return err
	}
	if src == nil {
		return nil
	}

	// Sections from the reference must be preserved.
	ini.MergeSectionsWithLastComments()(inic)
	if _, err := inic.ReadFrom(src); err != nil {
		return err
	}
	return src.Close()
}

func iniSave(inic *ini.INI, ini FromIni) error {
	dest, err := ini.WriteConfig(inic)
	if err != nil || dest == nil {
		return err
	}
	_, err = inic.WriteTo(dest)
	if err != nil {
		return err
	}
	return dest.Close()
}

func cliUpdate(inic *ini.INI, fs *pflag.FlagSet) {
	fs.Visit(func(f *pflag.Flag) {
		var section, key string
		if i := strings.IndexByte(f.Name, FlagSeparator); i > 0 {
			section = f.Name[:i]
			key = f.Name[i+1:]
		} else {
			key = f.Name
		}
		inic.Set(section, key, f.Value.String())
	})
}
