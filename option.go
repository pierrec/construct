package construct

import "io"

// Option is used to customize the behaviour of construct.
type Option func(*config) error

// OptionFlagsWriter sets the Writer for use when the usage is requested.
//
// If nil, it defaults to os.Stderr.
func OptionFlagsWriter(w io.Writer) Option {
	return func(c *config) error {
		c.options.fout = w
		return nil
	}
}

// OptionFlagsGroupSep defines the separator for grouped config items in command line flags.
// Config items are grouped using an embedded struct that does not implement the Config interface.
//
// If not set, it defaults to '-'.
func OptionFlagsGroupSep(sep rune) Option {
	return func(c *config) error {
		c.options.gsep = string(sep)
		return nil
	}
}

// OptionEnvSep is used to separate grouped config items in environment variables.
//
// If not set, it defaults to '_'.
func OptionEnvSep(sep rune) Option {
	return func(c *config) error {
		c.options.envsep = string(sep)
		return nil
	}
}

// OptionFlagsUsage defines the function to be called when an error is encountered
// while parsing command line flags.
// The supplied error is nil if the help was requested.
func OptionFlagsUsage(usage func(error, func(io.Writer) error) error) Option {
	return func(c *config) error {
		c.options.fusage = usage
		return nil
	}
}
