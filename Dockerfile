FROM gcr.io/distroless/cc-debian12:nonroot
WORKDIR /app
COPY main prover ./
USER nonroot:nonroot
ENTRYPOINT ["./main"]
