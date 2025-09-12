FROM gcr.io/distroless/cc:nonroot
WORKDIR /app
USER nonroot:nonroot
EXPOSE 3000
COPY prover .
COPY main .
ENTRYPOINT ["./main"]
