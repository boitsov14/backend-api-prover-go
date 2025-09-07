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
# To update go.mod, run `go mod edit -go=1.xx.x`
# To update golangci-lint, visit https://golangci-lint.run/docs/welcome/install/#binaries
# To update tools, Ctrl+Shift+P and search for "Go: Install/Update Tools"
update-go:
    go version
    winget upgrade GoLang.Go || true

# Update dependencies
update:
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

# Copy binary from Rust project
copy-prover:
    cp "$RUST_PROJECT_PATH/target/x86_64-unknown-linux-gnu/release/theorem-prover-rs" "./prover"
