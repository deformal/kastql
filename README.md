# KastQL - GraphQL Proxy Engine

KastQL is a powerful GraphQL proxy/gateway that introspects multiple GraphQL servers and provides a unified GraphQL endpoint that routes queries, mutations, and subscriptions to the appropriate backend servers.

## Features

- **GraphQL Introspection**: Automatically introspects GraphQL servers to understand their schemas
- **Intelligent Routing**: Routes queries, mutations, and subscriptions to the correct backend servers
- **Schema Registry**: Manages multiple GraphQL server registrations
- **GraphQL Playground**: Built-in GraphQL playground for testing queries
- **Health Monitoring**: Health check endpoints for monitoring server status
- **CORS Support**: Built-in CORS support for cross-origin requests

## Quick Start

### 1. Build the application

```bash
go build -o kastql main.go
```

### 2. Register a GraphQL server

```bash
./kastql register \
  --id "github-api" \
  --name "GitHub GraphQL API" \
  --endpoint "https://api.github.com/graphql" \
  --description "GitHub's GraphQL API"
```

### 3. Start the proxy server

```bash
./kastql serve --port 8080 --ui-port 3000
```

### 4. Access the endpoints

- **GraphQL Endpoint**: http://localhost:8080/graphql
- **GraphQL Playground**: http://localhost:8080/playground
- **Health Check**: http://localhost:8080/health
- **Schema Info**: http://localhost:8080/schema
- **UI**: http://localhost:3000

## Commands

### Register a server

```bash
kastql register --id <server-id> --name <server-name> --endpoint <graphql-url> [--description <description>]
```

### List registered servers

```bash
kastql list
```

### Start the proxy server

```bash
kastql serve [--port <port>] [--ui-port <ui-port>] [--verbose]
```

### Check server status

```bash
kastql status
```

## Architecture

### Core Components

1. **Introspection Engine** (`internal/introspection/`)
   - Performs GraphQL introspection on remote servers
   - Validates server endpoints
   - Parses and stores schema information

2. **Schema Registry** (`internal/storage/`)
   - Manages registered GraphQL servers
   - Stores introspected schemas
   - Provides server lookup functionality

3. **GraphQL Router** (`internal/core/router/`)
   - Routes GraphQL requests to appropriate servers
   - Parses queries to determine target servers
   - Handles request forwarding

4. **Proxy Server** (`internal/core/server/`)
   - Serves the unified GraphQL endpoint
   - Provides health checks and monitoring
   - Includes GraphQL Playground

### How it Works

1. **Registration**: When you register a GraphQL server, KastQL introspects it to understand its schema
2. **Storage**: The server information and schema are stored in the registry
3. **Routing**: When a GraphQL request comes in, KastQL parses the query to determine which server should handle it
4. **Forwarding**: The request is forwarded to the appropriate backend server
5. **Response**: The response is returned to the client

## Example Usage

### Register multiple servers

```bash
# Register GitHub API
kastql register --id github --name "GitHub API" --endpoint "https://api.github.com/graphql"

# Register a local GraphQL server
kastql register --id local --name "Local API" --endpoint "http://localhost:4000/graphql"
```

### Query through the proxy

```bash
curl -X POST http://localhost:8080/graphql \
  -H "Content-Type: application/json" \
  -d '{
    "query": "query { viewer { login } }"
  }'
```

## Development

### Project Structure

```
kastql/
├── cmd/                    # CLI commands
├── internal/
│   ├── core/              # Core server and router
│   ├── introspection/     # GraphQL introspection
│   ├── storage/           # Schema registry
│   ├── ui/                # Web UI
│   └── utils/             # Utilities
├── main.go                # Application entry point
└── README.md
```

### Building

```bash
go build -o kastql main.go
```

### Running Tests

```bash
go test ./...
```

## License

This project is licensed under the MIT License - see the LICENSE file for details.
