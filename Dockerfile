FROM gcr.io/distroless/static

COPY nanocmd-linux-amd64 /nanocmd

EXPOSE 9003

ENTRYPOINT ["/nanocmd"]
