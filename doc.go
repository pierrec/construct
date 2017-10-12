// Package construct provides a simple way to load configuration into a struct
// from multiple sources and formats, by relying on the struct embedded types
// and special interfaces.
//
// The goal is to reduce the code boilerplate used to initialize data structures to
// a minimum, by leveraging Go's types syntax definitions, interface methods, types
// reusability and by providing full support for complex field types (basic Go types,
// slices, maps and any combination of those.)
//
// For instance, when loading the data from the command line, instead of defining
// the cli options definitions using struct field tags, as it is usually done for
// cli parsing packages, the definitions are retrieved from the Config.Usage()
// method. This allows building definitions dynamically.
//
// Overview
//
// A struct has its fields populated by calling the Load() function.
// The rules to populate the struct fields are as follow:
//  - fields are only processed if they are exported
//  - a field represents a config item for the Config interface
//  - an embedded type implementing the Config interface is used to group config items logically
//  - an embedded type implementing the Config and FromFlags interfaces represents a subcommand
//  - fields processing can be modified using field tags with the following format
//
//     `... cfg:"[<key>][,<flag1>[,<flag2>]]" ...`
//
// If the key is "-", the field is ignored.
// If the key is not empty, the value is used as the field name.
//
// The following struct tag flags are currently supported:
//
//     inline       Inline the field which must be a struct, instead of
//                  processing it as a group of config items. Inlined fields
//                  must not collide with the outer struct ones.
//                  It has no effect on non embedded types.
//
// Subcommands
//
// Subcommands in command line flags are defined by embedding a struct
// implementing both the Config and FromFlags interfaces.
//
// The FlagsDone() method is invoked on the last subcommand with
// the remaining command line arguments.
//
// Sources
//
// Data used to populate structs can be fetched from various sources to override
// the current struct instance values in the following order:
//  - file in various formats
//  - environment variables
//  - command line flags
//
// The data sources are defined by implementing the relevant interfaces on the struct:
//  - FromFlags interface for command line flags
//  - FromEnv interface for environment variables
//  - FromIO interface for io sources
//
// Once the data is loaded from all sources, the Init() method is invoked
// on the main struct as well as all the embedded ones except subcommands that have
// not been requested.
//
// Supported field types
//
// The following types, as well as slices and maps of them, are supported:
//  - time.Duration, time.Time
//  - *url.URL
//  - *regexp.Regexp
//  - *text/template.Template, *html/template.Template
//  - *net.IPAddr, *net.IPNet
//  - bool
//  - string
//  - float32, float64
//  - int, int8, int16, int32, int64
//  - uint, uint8, uint16, uint32, uint64
//  - types implementing encoding.TextMarshaler and encoding.TextUnmarshaler
//
// Configuration formats
//
// The FromIO interface is used to load and save the configuration from and to
// any kind of storage and using any format.
//
// Implementations for file based storage and widely used formats such as json, toml,
// yaml or ini are available in the construct/constructs package.
//
package construct
