###################################
# Basic configuration
###################################

# For windows compatibility
set windows-shell := ["C:\\Program Files\\Git\\bin\\sh.exe", "-c"]

# Ignore recipe lines beginning with #.
set ignore-comments := true

# Load environment variables from .env file
set dotenv-load := true

# Format justfile
just-fmt:
    just --fmt --unstable

###################################
# Update
###################################

# Update Go
# To update golangci-lint, visit https://golangci-lint.run/docs/welcome/install/#binaries
# To update tools, Ctrl+Shift+P and search for "Go: Install/Update Tools"
update-go:
    go version
    winget upgrade GoLang.Go || true

# Update Go version in go.mod
update-mod:
    go mod edit -go=1.25.1

# Update dependencies
update:
    go mod tidy
    go get -t -u ./...
    go mod tidy

###################################
# Formatter and Linter
###################################

# Fmt
fmt:
    golangci-lint fmt

# Lint
lint:
    just fmt
    golangci-lint run --fix

###################################
# Run
###################################

# Run the project
run:
    ENV=dev go run .

###################################
# Dependencies
###################################

# Add dependency to go.mod
add package:
    go get {{ package }}

# Install binary package globally
# To delete an installed package, visit `C:\Users\xxx\go\bin`, and delete the exe file
# To check installed packages, visit the same folder
# To update an installed package, run `go install package@latest` again
# To update vscode go extension, Ctrl+Shift+P and search for "Go: Install/Update Tools"
install package:
    go install {{ package }}

###################################
# Build
###################################

# Build binary for Linux
build:
    GOOS=linux CGO_ENABLED=0 go build -ldflags="-s -w" -trimpath -o bin/main

# Copy binary from Rust project
copy:
    # rm -rf ./bin/*
    -cp "$RUST_PROJECT_PATH/target/release/theorem-prover-rs.exe" "./bin/prover-windows.exe"
    -cp "$RUST_PROJECT_PATH/target/trace/theorem-prover-rs.exe" "./bin/prover-trace-windows.exe"
    -cp "$RUST_PROJECT_PATH/target/x86_64-unknown-linux-gnu/release/theorem-prover-rs" "./bin/prover"
    -cp "$RUST_PROJECT_PATH/target/x86_64-unknown-linux-gnu/trace/theorem-prover-rs" "./bin/prover-trace"

###################################
# Docker
###################################

# Build Docker image
docker:
    docker build -t prover .

# Stop and remove Docker container
stop:
    docker stop prover || true
    docker rm prover || true

# Run Docker container
container:
    just stop
    docker run --env-file .env -p 3000:3000 --name prover prover

# Run all steps
all:
    just fmt
    just lint
    just update
    just build
    just copy
    just docker
    just container

###################################
# Deploy
###################################
