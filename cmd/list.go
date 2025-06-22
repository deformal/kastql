package cmd

import (
	"fmt"
	"log"
	"time"

	"github.com/deformal/kastql/internal/config"
	"github.com/spf13/cobra"
)

var listConfigFile string

var ListCmd = &cobra.Command{
	Use:   "list",
	Short: "List registered GraphQL servers",
	Long:  `List all registered GraphQL servers and their status`,
	Run: func(cmd *cobra.Command, args []string) {
		listServers()
	},
}

func listServers() {
	// Load configuration
	cfg, err := config.LoadConfig(listConfigFile)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize registry manager
	registryManager := config.NewRegistryManager(cfg)

	// Load registry from persistent storage
	registry, err := registryManager.LoadRegistry()
	if err != nil {
		log.Fatalf("Failed to load registry: %v", err)
	}

	servers := registry.GetAllServers()
	totalServers := registry.GetServerCount()
	activeServers := registry.GetActiveServerCount()

	fmt.Printf("KastQL Server Registry\n")
	fmt.Printf("======================\n")
	fmt.Printf("Registry File: %s\n", registryManager.GetRegistryPath())
	fmt.Printf("Total Servers: %d\n", totalServers)
	fmt.Printf("Active Servers: %d\n", activeServers)
	fmt.Printf("Inactive Servers: %d\n", totalServers-activeServers)
	fmt.Println()

	if len(servers) == 0 {
		fmt.Println("No servers registered.")
		fmt.Println("Use 'kastql register' to add a GraphQL server.")
		return
	}

	for i, server := range servers {
		fmt.Printf("%d. %s (%s)\n", i+1, server.Name, server.ID)
		fmt.Printf("   Endpoint: %s\n", server.Endpoint)
		if server.Description != "" {
			fmt.Printf("   Description: %s\n", server.Description)
		}
		fmt.Printf("   Status: %s\n", getStatusString(server.IsActive))
		fmt.Printf("   Added: %s\n", server.AddedAt.Format(time.RFC3339))
		fmt.Printf("   Updated: %s\n", server.UpdatedAt.Format(time.RFC3339))

		if server.Schema != nil {
			fmt.Printf("   Schema:\n")
			if server.Schema.QueryType != nil {
				fmt.Printf("     Query Type: %s\n", server.Schema.QueryType.Name)
			}
			if server.Schema.MutationType != nil {
				fmt.Printf("     Mutation Type: %s\n", server.Schema.MutationType.Name)
			}
			if server.Schema.SubscriptionType != nil {
				fmt.Printf("     Subscription Type: %s\n", server.Schema.SubscriptionType.Name)
			}
			fmt.Printf("     Total Types: %d\n", len(server.Schema.Types))
		}
		fmt.Println()
	}
}

func getStatusString(isActive bool) string {
	if isActive {
		return "✓ Active"
	}
	return "✗ Inactive"
}

func init() {
	ListCmd.Flags().StringVarP(&listConfigFile, "config", "c", "", "Configuration file path")
	RootCmd.AddCommand(ListCmd)
}
