// Package construct provides a simple way to load configuration into a struct,
// strongly relying on embedded types and interfaces.
//
// Data can be fetched from various sources in order of priority, overriding the
// struct instance (default) values:
//  - command line flags
//  - environment variables
//  - file in various formats
//
// The data sources are defined by implementing the relevant interfaces on the struct:
//  - FromFlags interface for command line flags
//  - FromEnv interface for environment variables
//  - FromIO interface for io sources
//
// Subcommands in command line flags is supported and defined by embeddeing a struct
// implementing both the Config and FromFlags interface.
//
// Once the data is loaded from all sources, the InitConfig() method is invoked
// on the main struct as well as all the embedded ones except subcommands.
//
// The rules on the struct fields are as follow:
//  - fields are only processed if they are exported
//  - a field represents an option for the Config
//  - an embedded type implementing the Config and FromFlags interfaces represents a subcommand
//  - an embedded type implementing the Config interface is used to group options
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
package construct
