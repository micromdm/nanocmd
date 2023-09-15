FROM gcr.io/distroless/static

ARG TARGETOS TARGETARCH

COPY nanocmd-$TARGETOS-$TARGETARCH /app/nanocmd

EXPOSE 9003

VOLUME ["/app/db"]

WORKDIR /app

ENTRYPOINT ["/app/nanocmd"]
