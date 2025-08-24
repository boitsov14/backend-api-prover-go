# Build stage
FROM golang:alpine AS builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY main.go .
RUN CGO_ENABLED=0 go build -o main -ldflags="-s -w" -trimpath

# Runtime stage
FROM gcr.io/distroless/cc-debian12:nonroot
WORKDIR /app
COPY --from=builder /build/main .
USER nonroot:nonroot
ENTRYPOINT ["./main"]
