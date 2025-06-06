package commandsandflags_test

import "github.com/deformal/kastql/cmd/types"

var Commands = map[string]types.Command{
	"serve": {
		Name:        "Serve",
		Description: "This KastQl command serves up the web interface for the Graphql Router.",
		Flags: &map[string]types.CommandFlags{
			"port": {
				Name:        "Port",
				Description: "The port flag helps you define a port for the Graphql Router to be launched on.",
			},
			"config": {
				Name:        "Config",
				Description: "This flag helps you point the router to the local configuration ('config.yaml') file if any.",
			},
			"help": {
				Name:        "Help",
				Description: "Documentation and examples on how to use the serve command",
			},
		},
	},
	"status": {
		Name:        "Status",
		Description: "This KastQl command shows the status of the GraphQl Router.",
	},
}
