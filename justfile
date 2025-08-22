# For windows compatibility
set windows-shell := ["C:\\Program Files\\Git\\bin\\sh.exe", "-c"]

# Ignore recipe lines beginning with #.
set ignore-comments := true

# Format justfile
just-fmt:
    just --fmt --unstable

# Update Go
update-go:
    winget upgrade --id GoLang.Go

# Update dependencies
update:
    go get -u ./...
    go mod tidy

# Add dependency to go.mod
add package:
    go get {{ package }}
