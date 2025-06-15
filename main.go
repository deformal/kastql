package main

import (
	"log"
	"os"

	"github.com/deformal/kastql/cmd"
)

func main() {
	if err := cmd.RootCmd.Execute(); err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
}
