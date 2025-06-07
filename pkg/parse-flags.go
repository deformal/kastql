package pkg

import (
	"flag"
	"fmt"
	"log"
	"os"
	"slices"
	"strings"

	"github.com/deformal/kastql/internal/utils"
)

func ProcessCommandLineFlagsForServeCommand(osArgs []string) {
	serveComamnd := flag.NewFlagSet("serve", flag.ExitOnError)
	port := serveComamnd.Int("port", utils.DefaultPort, "Port for the KasQl Enigne")
	config := serveComamnd.String("config", utils.ConfigFilePathAndName, "Config file if any")
	verbose := serveComamnd.Bool("verbose", false, "Enable verbose logging")
	serveComamnd.Parse(osArgs)
	configFileFormat := strings.Split(*config, ".")[1]
	if !*verbose {
		fmt.Println("Verbose logging disabled")
	}
	if len(*config) > 0 {
		if !slices.Contains(utils.AcceptedConfigFileFormats, configFileFormat) {
			log.Fatal("The config file must be a .yaml or yml file")
			os.Exit(1)
		}
		if _, err := os.Stat(*config); os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "❌ Config file not found: %s\n", *config)
			os.Exit(1)
		}
	}

	fmt.Printf("The Graphql Engine is starting on port: %d with config file", *port)
}
