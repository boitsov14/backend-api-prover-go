FROM cgr.dev/chainguard/glibc-dynamic
WORKDIR /app
EXPOSE 3000
COPY bin/prover bin/prover-trace bin/
COPY main .
ENTRYPOINT ["./main"]
