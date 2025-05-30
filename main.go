package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/deformal/kastql/cmd"
)

func main() {
	cmd.WelcomeMessage()
	switch os.Args[1] {
	case "serve":
		serveCmd := flag.NewFlagSet("serve", flag.ExitOnError)
		port := serveCmd.Int("port", 9000, "Port to serve KastQL on")
		serveCmd.Parse(os.Args[2:])
		fmt.Println("Port", *port)
	default:
		os.Exit(1)
	}
}
