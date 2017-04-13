package iniconfig_test

import (
	"crypto/aes"
	"fmt"
	"io"
	"os"

	ini "github.com/pierrec/go-ini"
	"github.com/pierrec/go-iniconfig"
)

func init() {
	key := []byte("this is a private key for aes256")
	var err error
	iniconfig.PasswordBlock, err = aes.NewCipher(key)
	if err != nil {
		panic(err)
	}
}

type Log struct {
	FileSize iniconfig.BytesSize
}

type Config struct {
	ConfigFile string `ini:"-"`
	SaveConfig bool   `ini:"-"`

	Host     string
	Port     int
	Login    string
	Password iniconfig.Password

	Log
}

func (c *Config) FlagUsageConfig(name string) string {
	switch name {
	case "host":
		return "host to connect to"
	case "port":
		return "listening port to connect to"
	case "login":
		return "login username"
	case "password":
		return "password for the user"
	case "log-filesize":
		return "logfile max size"
	}
	return ""
}

var _ iniconfig.FromIni = (*Config)(nil)

func (c *Config) LoadConfig() (io.ReadCloser, error) {
	if c.ConfigFile == "" {
		return nil, nil
	}
	f, err := os.Open(c.ConfigFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	return f, nil
}

func (c *Config) WriteConfig() (io.WriteCloser, error) {
	if !c.SaveConfig {
		return nil, nil
	}
	fname := c.ConfigFile
	if fname == "" {
		fname = "config.ini"
	}
	return os.Create(fname)
}

func Example() {
	config := &Config{
		ConfigFile: "config.ini",
		SaveConfig: true,
		Host:       "localhost",
		Port:       80,
		Login:      "xxlogin",
		Password:   "xxpwd",
		Log:        Log{1 << 20},
	}

	err := iniconfig.Load(config, ini.Comment('#'))
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Printf(">> %#v\n", config)

	// Output:
}
