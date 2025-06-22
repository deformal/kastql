package cmd

import (
	"github.com/deformal/kastql/internal/core/server"
	"github.com/spf13/cobra"
)

var (
	port      int  = 8080
	uiPort    int  = 3000
	useConfig bool = false
	verbose   bool = false
)

var ServeCmd = &cobra.Command{
	Use:   "serve",
	Short: "Serve the GraphQL Engine",
	Long:  `Serve the GraphQL Engine on the specified port`,
	Run: func(cmd *cobra.Command, args []string) {
		serverConfig := server.ServerConfig{
			Port:    port,
			UIPort:  uiPort,
			Verbose: verbose,
		}
		server.Server(serverConfig)
	},
}

func init() {
	ServeCmd.Flags().IntVarP(&port, "port", "p", 8080, "Port for the KastQl Engine")
	ServeCmd.Flags().IntVarP(&uiPort, "ui-port", "u", 3000, "Port for the KastQL UI")
	ServeCmd.Flags().BoolVarP(&useConfig, "config", "c", false, "Config file if any")
	ServeCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose logging")
	RootCmd.AddCommand(ServeCmd)
}
