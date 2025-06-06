default:
    @just --list
    
build: 
    @echo "Building the tool"
    goreleaser build --clean