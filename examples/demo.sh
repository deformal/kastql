#!/bin/bash

# KastQL Demo Script
# This script demonstrates how to use KastQL as a GraphQL proxy

echo "🚀 KastQL GraphQL Proxy Demo"
echo "============================"
echo ""

# Check if kastql binary exists
if [ ! -f "./kastql" ]; then
    echo "❌ kastql binary not found. Please build it first with: go build -o kastql main.go"
    exit 1
fi

echo "✅ KastQL binary found"
echo ""

# Show available commands
echo "📋 Available Commands:"
./kastql --help
echo ""

# Show register command help
echo "📝 Register Command Help:"
./kastql register --help
echo ""

# Example: Register a public GraphQL API (GitHub)
echo "🔧 Example: Register GitHub GraphQL API"
echo "Note: This is just an example. You'll need a GitHub token for actual queries."
echo ""

# Show the command that would be used
echo "Command to register GitHub API:"
echo "./kastql register --id github --name \"GitHub API\" --endpoint \"https://api.github.com/graphql\" --description \"GitHub's GraphQL API\""
echo ""

# Show the serve command
echo "🚀 To start the proxy server:"
echo "./kastql serve --port 8080 --ui-port 3000"
echo ""

# Show available endpoints
echo "🌐 Available Endpoints:"
echo "- GraphQL Endpoint: http://localhost:8080/graphql"
echo "- GraphQL Playground: http://localhost:8080/playground"
echo "- Health Check: http://localhost:8080/health"
echo "- Schema Info: http://localhost:8080/schema"
echo "- UI: http://localhost:3000"
echo ""

# Show example query
echo "📤 Example GraphQL Query:"
echo "curl -X POST http://localhost:8080/graphql \\"
echo "  -H \"Content-Type: application/json\" \\"
echo "  -d '{"
echo "    \"query\": \"query { viewer { login } }\""
echo "  }'"
echo ""

echo "🎉 Demo completed! Build and run KastQL to get started." 