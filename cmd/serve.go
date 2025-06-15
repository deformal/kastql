package cmd

import (
	"fmt"
	"log"
	"net/http"

	"github.com/deformal/kastql/internal/ui"
	"github.com/spf13/cobra"
)

var (
	port    int  = 8080
	config  bool = false
	verbose bool = false
)

var ServeCmd = &cobra.Command{
	Use:   "serve",
	Short: "Serve the GraphQL Engine",
	Long:  `Serve the GraphQL Engine on the specified port`,
	Run: func(cmd *cobra.Command, args []string) {
		serve(cmd, args)
	},
}

func serve(cmd *cobra.Command, args []string) {
	mux := http.NewServeMux()
	mux.Handle("/", ui.Handler())
	servingMessage := fmt.Sprintf("http://localhost:%d/", port)
	fmt.Printf("The Graphql Engine is running on: %s", servingMessage)
	err := http.ListenAndServe(fmt.Sprintf(":%d", port), mux)
	if err != nil {
		log.Fatal(err)
	}
}

func init() {
	ServeCmd.Flags().IntVarP(&port, "port", "p", 8080, "Port for the KasQl Enigne")
	ServeCmd.Flags().BoolVarP(&config, "config", "c", false, "Config file if any")
	ServeCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose logging")
	RootCmd.AddCommand(ServeCmd)
}
