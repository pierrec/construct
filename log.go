package iniconfig

import (
	"fmt"
	"io"
	"log"
	"os"

	"comail.io/go/colog"
	lumberjack "gopkg.in/natefinch/lumberjack.v2"
)

// ConfigLog provides the options for the logging facility.
// The logger is based on CoLog (https://texlution.com/post/colog-prefix-based-logging-in-golang/).
type ConfigLog struct {
	Filename   string
	Level      string
	MaxSize    BytesSize
	MaxAge     int
	MaxBackups int
	LocalTime  bool

	log *colog.CoLog
}

var (
	_ Config    = (*ConfigLog)(nil)
	_ FromFlags = (*ConfigLog)(nil)
)

// ConfigLogDefault represents sensible values for a default ConfigLog.
var ConfigLogDefault = ConfigLog{
	Level:      "error",
	MaxSize:    10 << 20, // 10 MB
	MaxAge:     30,
	MaxBackups: 3,
	LocalTime:  true,
}

// InitConfig makes ConfigLog implement Config.
func (lg *ConfigLog) InitConfig() error {
	lvl, err := colog.ParseLevel(lg.Level)
	if err != nil {
		return err
	}

	var out io.Writer = os.Stderr
	if lg.Filename != "" {
		out = &lumberjack.Logger{
			Filename:   lg.Filename,
			MaxSize:    int(lg.MaxSize),
			MaxBackups: lg.MaxBackups,
			MaxAge:     lg.MaxAge,
			LocalTime:  lg.LocalTime,
		}
	}
	flags := log.Ldate | log.Ltime | log.Lshortfile
	if !lg.LocalTime {
		flags |= log.LUTC
	}
	lg.log = colog.NewCoLog(out, "", flags)
	lg.log.SetMinLevel(lvl)

	// Disable default settings by the log library and register colog.
	log.SetPrefix("")
	log.SetFlags(0)
	log.SetOutput(lg.log)

	return nil
}

// HasFlagsConfig makes ConfigLog implement FromFlags.
func (*ConfigLog) FlagsUsageConfig() []string {
	return nil
}

// UsageConfig makes ConfigLog implement Config.
func (lg *ConfigLog) UsageConfig() []string {
	return nil
}

// OptionUsageConfig makes ConfigLog implement Config.
func (lg *ConfigLog) OptionUsageConfig(name string) []string {
	var s string
	switch name {
	case "log-filename":
		s = "file to write logs to (default=stderr)"
	case "log-level":
		levels := []colog.Level{colog.LTrace, colog.LDebug, colog.LInfo, colog.LWarning, colog.LError}
		s = fmt.Sprintf("logging level (one of %v)", levels)
	case "log-maxsize":
		s = "maximum size in megabytes of the log file"
	case "log-maxage":
		s = "maximum number of days to retain old log files"
	case "log-maxbackups":
		s = "maximum number of old log files to retain"
	case "log-localtime":
		s = "do not use UTC time for formatting the timestamps in files"
	default:
		return nil
	}
	return []string{s}
}
