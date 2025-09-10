FROM gcr.io/distroless/cc:nonroot
WORKDIR /app
COPY main prover ./
USER nonroot:nonroot
EXPOSE 3000
ENTRYPOINT ["./main"]
