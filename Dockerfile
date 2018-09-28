FROM golang:1.11 as builder
WORKDIR /go/src/github.com/koshatul/auth-proxy
COPY . /go/src/github.com/koshatul/auth-proxy
RUN make artifacts/build/release/linux/amd64/proxy

FROM ubuntu:xenial
RUN apt-get update \
    && apt-get install --quiet --yes --no-install-recommends ca-certificates=20170717~16.04.1 openssl=1.0.2g-1ubuntu4.13 \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

ENV HTTP_PORT="80" AUDIENCE="tls-web-client-auth" BACKEND_URL=""

EXPOSE 80/tcp
COPY --from=builder /go/src/github.com/koshatul/auth-proxy/artifacts/build/release/linux/amd64/proxy /proxy
ENTRYPOINT ["/proxy"]
