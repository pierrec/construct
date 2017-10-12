# construct
--
    import "github.com/pierrec/construct"

Package construct provides a simple way to load configuration into a struct from
multiple sources and formats, by relying on the struct embedded types and
special interfaces.

The goal is to reduce the code boilerplate used to initialize data structures to
a minimum, by leveraging Go's types syntax definitions, interface methods, types
reusability and by providing full support for complex field types (basic Go
types, slices, maps and any combination of those.)

For instance, when loading the data from the command line, instead of defining
the cli options definitions using struct field tags, as it is usually done for
cli parsing packages, the definitions are retrieved from the Config.Usage()
method. This allows building definitions dynamically.

### Status

This is work in progress and there are still issues to be adressed.

### Overview

A struct has its fields populated by calling the Load() function. The rules to
populate the struct fields are as follow:

    - fields are only processed if they are exported
    - a field represents a config item for the Config interface
    - an embedded type implementing the Config interface is used to group config items logically
    - an embedded type implementing the Config and FromFlags interfaces represents a subcommand
    - fields processing can be modified using field tags with the following format

       `... cfg:"[<key>][,<flag1>[,<flag2>]]" ...`

If the key is "-", the field is ignored. If the key is not empty, the value is
used as the field name.

The following struct tag flags are currently supported:

    inline       Inline the field which must be a struct, instead of
                 processing it as a group of config items. Inlined fields
                 must not collide with the outer struct ones.
                 It has no effect on non embedded types.


### Subcommands

Subcommands in command line flags are defined by embedding a struct implementing
both the Config and FromFlags interfaces.

The FlagsDone() method is invoked on the last subcommand with the remaining
command line arguments.


### Sources

Data used to populate structs can be fetched from various sources to override
the current struct instance values in the following order:

    - file in various formats
    - environment variables
    - command line flags

The data sources are defined by implementing the relevant interfaces on the
struct:

    - FromFlags interface for command line flags
    - FromEnv interface for environment variables
    - FromIO interface for io sources

Once the data is loaded from all sources, the Init() method is invoked on the
main struct as well as all the embedded ones except subcommands that have not
been requested.


Supported field types

The following types, as well as slices and maps of them, are supported:

    - time.Duration, time.Time
    - *url.URL
    - *regexp.Regexp
    - *text/template.Template, *html/template.Template
    - *net.IPAddr, *net.IPNet
    - bool
    - string
    - float32, float64
    - int, int8, int16, int32, int64
    - uint, uint8, uint16, uint32, uint64
    - types implementing encoding.TextMarshaler and encoding.TextUnmarshaler


Configuration formats

The FromIO interface is used to load and save the configuration from and to any
kind of storage and using any format.

Implementations for file based storage and widely used formats such as json,
toml, yaml or ini are available in the construct/constructs package.

## Usage

```go
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
	// e.g. for a field defined as
	//      Field map[int][]string `...sep=" :,"...`
	//
	//  map items are separated by a space, its key by a ':' and the slice items by a ','
	//  so that `key1:a,b key2:x,y` is deserialized as [key1:["a","b"] key2:["x","y"]].
	TagSepID = "sep"
)
```

#### func  Load

```go
func Load(config Config, options ...Option) error
```
Load populates the config with data from various sources. config must be a
pointer to a struct.

The values are set based on the implemented interfaces by config in order of
priority:

    - cli value: provided by the FromFlags interface
    - env value: provided by the FromEnv interface
    - ini value: provided by the FromIO interface
    - default value: values initially set in config

#### func  LoadArgs

```go
func LoadArgs(config Config, args []string, options ...Option) error
```
LoadArgs is equivalent to Load using the given arguments. The first argument
must be the real one, not the executable.

#### type Config

```go
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
```

Config defines the main interface for a config struct. Any embedded struct is
processed specifically depending on the interfaces it implements:

    - Config interface: it defines a group of config items with a prefix set to the embedded type name
    - Config and FromFlags interfaces: it defines a subcommand, which is automatically loaded from flags.
      Subcommands are not case sensitive.

The embedded type names and field names can be overriden by a struct tag
specifying the name to be used.

#### type FromEnv

```go
type FromEnv interface {
	// Env returns the name of the environment variable used for the given config item.
	// Return an empty value to ignore the config item.
	Env(name string) string
}
```

FromEnv defines the interface to set values from environment variables.

#### type FromFlags

```go
type FromFlags interface {
	// FlagsDone is called once the flags have been processed
	// with the previous subcommands and the remaining arguments.
	FlagsDone(cmds []Config, args []string) error

	// FlagsShort returns the short flag for the long name.
	FlagsShort(name string) string
}
```

FromFlags defines the interface to set values from command line flags.

#### type FromIO

```go
type FromIO interface {
	// Load returns the source for the data.
	Load() (io.ReadCloser, error)

	// Save returns the destination for the data.
	Save() (io.WriteCloser, error)

	// New returns a new instance of Store.
	New(seps LookupFn) Store
}
```

FromIO defines the interface to set values from an io source (typically a file).
The supported formats are currently: ini, toml, json and yaml.

#### type LookupFn

```go
type LookupFn func(key ...string) []rune
```

LookupFn is the function signature used to return the runes used for
(de)serializing data on a given key.

#### type Option

```go
type Option func(*config) error
```

Option is used to customize the behaviour of construct.

#### func  OptionEnvSep

```go
func OptionEnvSep(sep rune) Option
```
OptionEnvSep is used to separate grouped config items in environment variables.

If not set, it defaults to '_'.

#### func  OptionFlagsGroupSep

```go
func OptionFlagsGroupSep(sep rune) Option
```
OptionFlagsGroupSep defines the separator for grouped config items in command
line flags. Config items are grouped using an embedded struct that does not
implement the Config interface.

If not set, it defaults to '-'.

#### func  OptionFlagsUsage

```go
func OptionFlagsUsage(usage func(error, func(io.Writer) error) error) Option
```
OptionFlagsUsage defines the function to be called when an error is encountered
while parsing command line flags. The supplied error is nil if the help was
requested.

#### func  OptionFlagsWriter

```go
func OptionFlagsWriter(w io.Writer) Option
```
OptionFlagsWriter sets the Writer for use when the usage is requested.

If nil, it defaults to os.Stderr.

#### type Store

```go
type Store interface {
	// Has check the existence of the key.
	Has(keys ...string) bool

	// Get retrieves the value of the given key.
	Get(keys ...string) (interface{}, error)

	// Set changes the value of the given key.
	Set(value interface{}, keys ...string) error

	// SetComment defines the comment for the given key.
	SetComment(comment string, keys ...string) error

	// Used when deserializing config items.
	io.ReaderFrom

	// Used when serializing config items.
	io.WriterTo

	// StructTag returns the tag id used in struct field tags for the data format.
	// Field tags set to "-" are ignored.
	StructTag() string
}
```

Store defines the interface for retrieving config items stored in various data
formats.

Check the constructs package for implementations.
