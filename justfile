default:
    @just --list

dev *args="":
    go run ./main.go {{args}}

build-dev: 
    @echo "Building"
    goreleaser build --clean --snapshot

start-dev:
    @echo "Starting built version"
    ./dist/kastql_darwin_amd64_v1/kastql