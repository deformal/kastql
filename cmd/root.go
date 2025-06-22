package cmd

import (
	"github.com/spf13/cobra"
)

var RootCmd = cobra.Command{
	Use:   "kastql [command] [...flags]",
	Short: "KastQL is a full fledged GraphQL engine.",
	Long: `KastQL is a powerful and flexible GraphQL engine that provides a complete solution for building and managing GraphQL APIs. 
It offers robust query execution, schema management, and type safety features, making it an ideal choice for developers looking to implement GraphQL in their applications.

Key features include:
- Efficient query parsing and execution
- Comprehensive schema validation
- Built-in type system
- Support for complex queries and mutations
- Extensible architecture for custom resolvers
- Performance optimized for production use

Whether you're building a new GraphQL API or migrating from REST, KastQL provides all the tools you need to create a robust and scalable GraphQL service.`,
	Version: "0.1.0",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}
