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

type Server struct {
	construct.ConfigFileINI

	Host     string
	Port     int
	Login    string
	Password construct.Password
}

var _ construct.Config = (*Server)(nil)
var _ construct.FromFlags = (*Server)(nil)

func (c *Server) DoConfig() {}

// UsageConfig returns the usage for the Server struct fields.
// The UsageConfig method for the embedded struct is automatically called by construct.
func (c *Server) UsageConfig(name string) string {
	switch name {
	case "host":
		return "host to connect to"
	case "port":
		return "listening port to connect to"
	case "login":
		return "login username"
	case "password":
		return "password for the user"
	}
	return ""
}

func Example() {
	Server := &Server{
		ConfigFileINI: construct.ConfigFileINI{
			Name:            "config.ini",
			BackupExtension: ".bak",
			Save:            true},
		Host:     "localhost",
		Port:     80,
		Login:    "xxlogin",
		Password: "xxpwd",
	}

	err := construct.Load(Server)
	if err != nil {
		fmt.Println(err)
		return
	}

	pretty.Println(Server)

	// Output:
	// 	&construct_test.Server{
	//     ConfigFileINI: construct.ConfigFileINI{
	//         Name:            "config.ini",
	//         BackupExtension: ".bak",
	//         Save:            true,
	//         configFile:      construct.configFile{},
	//     },
	//     Host:     "localhost",
	//     Port:     80,
	//     Login:    "xxlogin",
	//     Password: "xxpwd",
	// }
}
