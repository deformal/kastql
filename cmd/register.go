package cmd

import (
	"fmt"
	"log"

	"github.com/deformal/kastql/internal/config"
	"github.com/deformal/kastql/internal/introspection"
	"github.com/spf13/cobra"
)

var (
	serverID          string
	serverName        string
	serverEndpoint    string
	serverDescription string
	configFilePath    string
)

var RegisterCmd = &cobra.Command{
	Use:   "register",
	Short: "Register a GraphQL server for introspection",
	Long:  `Register a GraphQL server endpoint for introspection and routing`,
	Run: func(cmd *cobra.Command, args []string) {
		registerServer()
	},
}

func registerServer() {
	if serverID == "" {
		log.Fatal("Server ID is required (--id)")
	}
	if serverName == "" {
		log.Fatal("Server name is required (--name)")
	}
	if serverEndpoint == "" {
		log.Fatal("Server endpoint is required (--endpoint)")
	}

	cfg, err := config.LoadConfig(configFilePath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	registryManager := config.NewRegistryManager(cfg)

	registry, err := registryManager.LoadRegistry()
	if err != nil {
		log.Fatalf("Failed to load registry: %v", err)
	}

	if existingServer, err := registry.GetServer(serverID); err == nil {
		log.Fatalf("Server with ID '%s' already exists: %s (%s)", serverID, existingServer.Name, existingServer.Endpoint)
	}

	engine := introspection.NewEngine()

	fmt.Printf("Validating endpoint: %s\n", serverEndpoint)
	if err := engine.ValidateEndpoint(serverEndpoint); err != nil {
		log.Fatalf("Failed to validate endpoint: %v", err)
	}
	fmt.Println("✓ Endpoint is accessible")

	fmt.Printf("Performing introspection on: %s\n", serverEndpoint)
	schema, err := engine.Introspect(serverEndpoint)
	if err != nil {
		log.Fatalf("Failed to introspect server: %v", err)
	}
	fmt.Println("✓ Introspection completed successfully")

	err = registry.RegisterServer(serverID, serverName, serverEndpoint, serverDescription, schema)
	if err != nil {
		log.Fatalf("Failed to register server: %v", err)
	}

	if err := registryManager.SaveRegistry(registry); err != nil {
		log.Fatalf("Failed to save registry: %v", err)
	}

	fmt.Printf("✓ Successfully registered server '%s' (%s)\n", serverName, serverID)
	fmt.Printf("  Endpoint: %s\n", serverEndpoint)
	if serverDescription != "" {
		fmt.Printf("  Description: %s\n", serverDescription)
	}
	fmt.Printf("  Registry saved to: %s\n", registryManager.GetRegistryPath())

	if schema.QueryType != nil {
		fmt.Printf("  Query Type: %s\n", schema.QueryType.Name)
	}
	if schema.MutationType != nil {
		fmt.Printf("  Mutation Type: %s\n", schema.MutationType.Name)
	}
	if schema.SubscriptionType != nil {
		fmt.Printf("  Subscription Type: %s\n", schema.SubscriptionType.Name)
	}
	fmt.Printf("  Total Types: %d\n", len(schema.Types))
}

func init() {
	RegisterCmd.Flags().StringVarP(&serverID, "id", "i", "", "Unique server ID")
	RegisterCmd.Flags().StringVarP(&serverName, "name", "n", "", "Server name")
	RegisterCmd.Flags().StringVarP(&serverEndpoint, "endpoint", "e", "", "GraphQL server endpoint URL")
	RegisterCmd.Flags().StringVarP(&serverDescription, "description", "d", "", "Server description (optional)")
	RegisterCmd.Flags().StringVarP(&configFilePath, "config", "c", "", "Configuration file path")

	RegisterCmd.MarkFlagRequired("id")
	RegisterCmd.MarkFlagRequired("name")
	RegisterCmd.MarkFlagRequired("endpoint")

	RootCmd.AddCommand(RegisterCmd)
}
