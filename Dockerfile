FROM golang:1.14 as builder
WORKDIR /code
RUN apt-get update
RUN apt-get install -qy unzip uuid-runtime zip
COPY . /code
RUN go mod download
RUN make artifacts/build/release/linux/amd64/proxy

FROM ubuntu:xenial
RUN apt-get update \
    && apt-get install --quiet --yes --no-install-recommends ca-certificates openssl \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

ENV HTTP_PORT="80" AUDIENCE="tls-web-client-auth" BACKEND_URL=""

EXPOSE 80/tcp
COPY --from=builder /code/artifacts/build/release/linux/amd64/proxy /proxy
ENTRYPOINT ["/proxy"]
