#!/bin/bash

echo "🚀 KastQL GraphQL Proxy Demo"
echo "============================"
echo ""

if [ ! -f "./kastql" ]; then
    echo "❌ kastql binary not found. Please build it first with: go build -o kastql main.go"
    exit 1
fi

echo "✅ KastQL binary found"
echo ""

echo "📋 Available Commands:"
./kastql --help
echo ""

echo "📝 Register Command Help:"
./kastql register --help
echo ""

echo "🔧 Example: Register GitHub GraphQL API"
echo "Note: This is just an example. You'll need a GitHub token for actual queries."
echo ""

echo "Command to register GitHub API:"
echo "./kastql register --id github --name \"GitHub API\" --endpoint \"https://api.github.com/graphql\" --description \"GitHub's GraphQL API\""
echo ""

echo "🚀 To start the proxy server:"
echo "./kastql serve --port 8080 --ui-port 3000"
echo ""

echo "🌐 Available Endpoints:"
echo "- GraphQL Endpoint: http://localhost:8080/graphql"
echo "- GraphQL Playground: http://localhost:8080/playground"
echo "- Health Check: http://localhost:8080/health"
echo "- Schema Info: http://localhost:8080/schema"
echo "- UI: http://localhost:3000"
echo ""

echo "📤 Example GraphQL Query:"
echo "curl -X POST http://localhost:8080/graphql \\"
echo "  -H \"Content-Type: application/json\" \\"
echo "  -d '{"
echo "    \"query\": \"query { viewer { login } }\""
echo "  }'"
echo ""

echo "🎉 Demo completed! Build and run KastQL to get started." 