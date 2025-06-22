package server

import (
	"fmt"
	"log"
	"net/http"

	"github.com/deformal/kastql/internal/storage"
	"github.com/deformal/kastql/internal/ui"
)

type ServerConfig struct {
	Port    int
	UIPort  int
	Verbose bool
}

func Server(config ServerConfig) {
	registry := storage.NewRegistry()

	graphqlMux := SetupGraphQLServer(registry)

	uiMux := http.NewServeMux()
	uiMux.Handle("/", ui.Handler())

	go func() {
		uiServingMessage := fmt.Sprintf("http://localhost:%d/", config.UIPort)
		fmt.Printf("The KastQL UI is running on: %s\n", uiServingMessage)
		err := http.ListenAndServe(fmt.Sprintf(":%d", config.UIPort), uiMux)
		if err != nil {
			log.Fatal(err)
		}
	}()

	graphqlServingMessage := fmt.Sprintf("http://localhost:%d/", config.Port)
	fmt.Printf("The KastQL GraphQL Engine is running on: %s\n", graphqlServingMessage)
	fmt.Printf("GraphQL endpoint: http://localhost:%d/graphql\n", config.Port)
	fmt.Printf("GraphQL Playground: http://localhost:%d/playground\n", config.Port)
	fmt.Printf("Health check: http://localhost:%d/health\n", config.Port)
	fmt.Printf("Schema info: http://localhost:%d/schema\n", config.Port)

	err := http.ListenAndServe(fmt.Sprintf(":%d", config.Port), graphqlMux)
	if err != nil {
		log.Fatal(err)
	}
}
