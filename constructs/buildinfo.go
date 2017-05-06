package constructs

import (
	"os"
	"text/template"

	"github.com/pierrec/construct"
)

var _ construct.Config = (*BuildInfo)(nil)

// BuildInfoMessage is the default template used to display the BuildInfo.
const BuildInfoMessage = "version {{.Version}} commit {{.Commit}} built on {{.BuildTime}}\n"

// BuildInfo provides a way to display a binary build information.
// The Data part must be set during the binary initialization,
// typically by providing the info with the go linker into
// custom string variables and setting them to the Data fields.
type BuildInfo struct {
	Show    bool   `cfg:"version"`
	Message string `cfg:"-"`
	Data    struct {
		Version   string
		Commit    string
		BuildTime string
	}
}

// Init displays the build information to os.Stdout and exits.
func (bi *BuildInfo) Init() (err error) {
	if !bi.Show {
		return nil
	}
	msg := bi.Message
	if msg == "" {
		msg = BuildInfoMessage
	}
	t, err := template.New("").Parse(msg)
	if err != nil {
		return err
	}
	if err := t.Execute(os.Stdout, bi.Data); err != nil {
		return err
	}
	os.Exit(0)
	return nil
}

// Usage returns the BuildInfo usage for each of its options.
func (bi *BuildInfo) Usage(name string) string {
	switch name {
	case "version":
		return "Print version information and quit"
	}
	return ""
}
