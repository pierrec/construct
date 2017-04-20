package construct_test

import (
	"crypto/aes"
	"fmt"

	"github.com/kr/pretty"
	"github.com/pierrec/construct"
)

func init() {
	key := []byte("this is a private key for aes256")
	var err error
	construct.PasswordBlock, err = aes.NewCipher(key)
	if err != nil {
		panic(err)
	}
}

type Config struct {
	construct.ConfigFile

	Host     string
	Port     int
	Login    string
	Password construct.Password
}

var _ construct.Config = (*Config)(nil)

var _ construct.FromFlags = (*Config)(nil)

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
		ConfigFile: construct.ConfigFile{"config.ini", true},
		Host:       "localhost",
		Port:       80,
		Login:      "xxlogin",
		Password:   "xxpwd",
	}

	err := construct.Load(config)
	if err != nil {
		fmt.Println(err)
		return
	}

	pretty.Println(config)

	// Output:
	// 	&construct_test.Config{
	//     ConfigFile: construct.ConfigFile{Name:"config.ini", Save:true},
	//     Host:       "localhost",
	//     Port:       80,
	//     Login:      "xxlogin",
	//     Password:   "xxpwd",
	// }
}
