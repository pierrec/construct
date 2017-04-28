package construct_test

import (
	"io"
	"testing"

	"github.com/pierrec/construct"
	"github.com/pierrec/construct/constructs"
)

type invalid int

func (invalid) DoConfig()                      {}
func (invalid) InitConfig() error              { return nil }
func (invalid) UsageConfig(name string) string { return "" }

// Invalid input: not a pointer to a struct.
func TestInvalid(t *testing.T) {
	var i invalid

	if err := construct.Load(i); err == nil {
		t.Error("error expected")
	}
	if err := construct.Load(&i); err == nil {
		t.Error("error expected")
	}
}

type cfg struct {
	Skip int `cfg:"-"`
	S    string
	I    int
	SN   string `cfg:"mystr"`
	IN   int    `cfg:"myint"`
}

func (*cfg) DoConfig()                      {}
func (*cfg) InitConfig() error              { return nil }
func (*cfg) UsageConfig(name string) string { return "" }

type cfgFlags struct {
	cfg
}

func (*cfgFlags) FlagsUsageConfig() io.Writer { return nil }

type cfgIO struct {
	constructs.ConfigFileINI
	cfg
}

func (*cfgIO) UsageConfig(name string) string { return "" }

func _TestLoadNoEmbedded(t *testing.T) {
	c := cfg{
		S:  "a",
		I:  1,
		SN: "b",
		IN: 2,
	}

	if err := construct.Load(&c); err != nil {
		t.Fatal(err)
	}

	cf := cfgFlags{c}
	if err := construct.Load(&cf); err != nil {
		t.Fatal(err)
	}

	cio := cfgIO{cfg: c}
	if err := construct.Load(&cio); err != nil {
		t.Fatal(err)
	}
}

type Group struct {
	V int
}

func (c *Group) InitConfig() error {
	c.V *= 100
	return nil
}

func (c *Group) UsageConfig(name string) string { return "" }

type cfgEmb struct {
	Group
	V int
}

func (*cfgEmb) DoConfig() {}
func (c *cfgEmb) InitConfig() error {
	c.V *= 10
	return nil
}
func (c *cfgEmb) UsageConfig(name string) string { return "" }

func TestLoadEmbedded(t *testing.T) {
	c := cfgEmb{
		Group{123},
		456,
	}

	if err := construct.Load(&c); err != nil {
		t.Fatal(err)
	}

	// Check that InitConfig() is called on embedded types.
	w := cfgEmb{Group{12300}, 4560}
	if got, want := c, w; got != want {
		t.Errorf("got %v; expected %v", got, want)
	}
}

var _ construct.Config = (*ConfigGroup)(nil)

type ConfigGroup struct {
	Group
}

func (*ConfigGroup) DoConfig() {}

type cfgEmbConfig struct {
	ConfigGroup
	V int
}

func (c *cfgEmbConfig) InitConfig() error {
	c.V *= 10
	return nil
}
func (c *cfgEmbConfig) UsageConfig(name string) string { return "" }

func TestLoadEmbeddedConfig(t *testing.T) {
	c := cfgEmbConfig{
		ConfigGroup{Group{123}},
		456}

	if err := construct.Load(&c); err != nil {
		t.Fatal(err)
	}

	// Check that InitConfig() is NOT called on Config embedded types.
	w := cfgEmbConfig{
		ConfigGroup{Group{123}},
		4560}
	if got, want := c, w; got != want {
		t.Errorf("got %v; expected %v", got, want)
	}
}
