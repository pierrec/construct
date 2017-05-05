// Package construct provides a simple way to load configuration into a struct,
// from multiple sources, by relying on embedded types and interfaces defined on those.
//
// The goal is to reduce the code boilerplate used to initialize data structures to
// a minimum, by leveraging Go's types syntax definitions, interface methods, types
// reusability and by providing full support for complex field types (basic Go types, slices,
// maps and any combination of those).
//
// Overview
//
// A struct has its fields populated by calling the Load() function.
// The rules to populate the struct fields are as follow:
//  - fields are only processed if they are exported
//  - a field represents an option for the Config
//  - an embedded type implementing the Config interface is used to group options logically
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
//                  processing it as a group of options. Inlined fields must
//                  not collide with the outer struct ones.
//                  It has no effect on non embedded types.
//
// Subcommands
//
// Subcommands in command line flags is supported and defined by embedding a struct
// implementing both the Config and FromFlags interfaces. The FlagsDoneConfig() method
// is invoked on the last subcommand with the remaining command line arguments.
//
// Sources
//
// Data used to populate structs can be fetched from various sources in order of priority,
// overriding the struct instance (default) values:
//  - command line flags
//  - environment variables
//  - file in various formats
//
// The data sources are defined by implementing the relevant interfaces on the struct:
//  - FromFlags interface for command line flags
//  - FromEnv interface for environment variables
//  - FromIO interface for io sources
//
// Once the data is loaded from all sources, the InitConfig() method is invoked
// on the main struct as well as all the embedded ones except subcommands that have
// not been requested.
//
// TODO
//  - support comments in config files
//  - expand variables? (cf yaml)
//  - option to change any separator
package construct
