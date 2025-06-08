package main

import (
	"fmt"
	"log"
	"os"

	"github.com/deformal/kastql/cmd"
	"github.com/deformal/kastql/internal/utils"

	"github.com/deformal/kastql/cmd/types"
)

func main() {
	args := cmd.WelcomeMessage()
	switch args.Command {
	case types.Serve:
		cmd.ProcessCommandLineFlagsForServeCommand(args.Flags)
	case types.Status:
		cmd.Status()
	case types.Version:
		fmt.Printf("Current version is %s", utils.Version)
	default:
		log.Fatal("At least one command to be expected\nshow docs")
		os.Exit(1)
	}
}
