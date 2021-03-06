package constructs

import (
	"fmt"
	"io"
	"log"
	"os"

	"github.com/pierrec/construct"

	"comail.io/go/colog"
	lumberjack "gopkg.in/natefinch/lumberjack.v2"
)

var _ construct.Config = (*ConfigLog)(nil)

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

// ConfigLogDefault represents sensible values for a default ConfigLog.
var ConfigLogDefault = ConfigLog{
	Level:      "error",
	MaxSize:    10 << 20, // 10 MB
	MaxAge:     30,
	MaxBackups: 3,
	LocalTime:  true,
}

// Init makes ConfigLog implement Config.
func (lg *ConfigLog) Init() error {
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

// Usage makes ConfigLog implement Config.
func (lg *ConfigLog) Usage(name string) string {
	switch name {
	case "Filename":
		return "file to write logs to (default=stderr)"
	case "Level":
		levels := []colog.Level{colog.LTrace, colog.LDebug, colog.LInfo, colog.LWarning, colog.LError}
		return fmt.Sprintf("logging level (one of %v)", levels)
	case "MaxSize":
		return "maximum size in megabytes of the log file"
	case "MaxAge":
		return "maximum number of days to retain old log files"
	case "MaxBackups":
		return "maximum number of old log files to retain"
	case "LocalTime":
		return "do not use UTC time for formatting the timestamps in files"
	}
	return ""
}
