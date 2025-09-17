FROM cgr.dev/chainguard/glibc-dynamic
WORKDIR /app
EXPOSE 3000
COPY prover .
COPY main .
ENTRYPOINT ["./main"]
