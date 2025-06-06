package pkg

import (
	"flag"
	"fmt"
	"os"

	"github.com/deformal/kastql/cmd/types"
	"github.com/deformal/kastql/internal/utils"
)

func ProcessCommandLineFlagsForServeCommand(osArgs []string) {
	flags := osArgs[2:]
	serveCmd := flag.NewFlagSet("serve", flag.ExitOnError)
	serveCmd.Parse(flags)
	for _, userFlag := range flags {
		switch userFlag {
		case types.ServeCommandPortFlag:
			port := serveCmd.Int("port", utils.DefaultPort, "Port to serve KastQL on")
			fmt.Printf("The Graphql Engine is starting on port: %d", port)
		default:
			fmt.Println("Incorrect Flags passed by the user")
			os.Exit(1)
		}
	}
}
