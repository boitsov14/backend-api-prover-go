# For windows compatibility
set windows-shell := ["C:\\Program Files\\Git\\bin\\sh.exe", "-c"]

# Ignore recipe lines beginning with #.
set ignore-comments := true

# Load environment variables from .env file
set dotenv-load := true

# Format justfile
just-fmt:
    just --fmt --unstable

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

# Fmt
fmt:
    golangci-lint fmt

# Lint
lint:
    golangci-lint run --fix

# Add dependency to go.mod
add package:
    go get {{ package }}

###############################################
# Build and Deploy
###############################################

# Build binary for Linux
build:
    GOOS=linux CGO_ENABLED=0 go build -ldflags="-s -w" -trimpath -o main

# Copy binary from Rust project
copy-prover:
    cp "$RUST_PROJECT_PATH/target/x86_64-unknown-linux-gnu/release/theorem-prover-rs" "./prover"

# Build Docker image
docker:
    docker build -t prover .

# Run Docker container
container:
    docker stop prover || true
    docker rm prover || true
    docker run --env-file .env -p 3000:3000 --name prover prover

# Stop and remove Docker container
stop:
    docker stop prover || true
    docker rm prover || true

# Run all steps
all:
    just fmt
    just lint
    just update
    just build
    just copy-prover
    just docker
    just container
