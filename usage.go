package construct

import "io"

// Usage writes out the config usage to the given Writer.
func Usage(config Config, out io.Writer) error {
	conf, err := newConfig(config)
	if err != nil {
		return err
	}
	usage := conf.buildFlagsUsage()

	return usage(out)
}
