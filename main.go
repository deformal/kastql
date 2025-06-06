package main

import (
	"log"
	"os"

	"github.com/deformal/kastql/cmd"
	"github.com/deformal/kastql/pkg"

	"github.com/deformal/kastql/cmd/types"
)

func main() {
	args := cmd.WelcomeMessage()
	switch args.Command {
	case types.Serve:
		pkg.ProcessCommandLineFlagsForServeCommand(args.Flags)
	case types.Status:
		cmd.Status()
	default:
		log.Fatal("Invalid Command passed.")
		os.Exit(1)
	}
}
