package iniconfig_test

import (
	"crypto/aes"
	"fmt"

	"github.com/kr/pretty"
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

type Config struct {
	iniconfig.ConfigFile

	Host     string
	Port     int
	Login    string
	Password iniconfig.Password
}

var _ iniconfig.Config = (*Config)(nil)

var _ iniconfig.FromFlags = (*Config)(nil)

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

func Example() {
	config := &Config{
		ConfigFile: iniconfig.ConfigFile{"config.ini", true},
		Host:       "localhost",
		Port:       80,
		Login:      "xxlogin",
		Password:   "xxpwd",
	}

	err := iniconfig.Load(config)
	if err != nil {
		fmt.Println(err)
		return
	}

	pretty.Println(config)

	// Output:
	// 	&iniconfig_test.Config{
	//     ConfigFile: iniconfig.ConfigFile{Name:"config.ini", Save:true},
	//     Host:       "localhost",
	//     Port:       80,
	//     Login:      "xxlogin",
	//     Password:   "xxpwd",
	// }
}
