package construct_test

import (
	"crypto/aes"
	"fmt"
	"os"

	"github.com/kr/pretty"
	"github.com/pierrec/construct"
	"github.com/pierrec/construct/constructs"
)

func init() {
	key := []byte("this is a private key for aes256")
	var err error
	constructs.PasswordBlock, err = aes.NewCipher(key)
	if err != nil {
		panic(err)
	}
}

type Server struct {
	constructs.ConfigFileINI

	Host     string
	Port     int
	Login    string
	Password constructs.Password
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
		ConfigFileINI: constructs.ConfigFileINI{
			Name:   "config.ini",
			Backup: ".bak",
			Save:   true},
		Host:     "localhost",
		Port:     80,
		Login:    "xxlogin",
		Password: "xxpwd",
	}
	defer func() {
		os.Remove("config.ini")
		os.Remove("config.bak")
	}()

	err := construct.Load(Server)
	if err != nil {
		fmt.Println(err)
		return
	}

	pretty.Println(Server)

	// Output:
	// 	&construct_test.Server{
	//     ConfigFileINI: constructs.ConfigFileINI{
	//         Name:       "config.ini",
	//         Backup:     ".bak",
	//         Save:       true,
	//         configFile: constructs.configFile{},
	//     },
	//     Host:     "localhost",
	//     Port:     80,
	//     Login:    "xxlogin",
	//     Password: "xxpwd",
	// }
}
