FROM golang:1.17-bullseye as builder

RUN set -eux; \
    apt-get update; \
    apt-get install -y --no-install-recommends unzip

WORKDIR ${GOPATH}/project
ENV DW_VERSION="1.0.19"

RUN  curl -sSLO https://github.com/mulesoft-labs/data-weave-cli/releases/download/v$DW_VERSION/dw-$DW_VERSION-Linux && \
    unzip -p dw-$DW_VERSION-Linux 'bin/dw' > dw && chmod +x dw
COPY . .
RUN go build -o /dataweavetransformation-adapter ./cmd/dataweavetransformation-adapter

FROM debian:stable-slim

WORKDIR /opt/dw
ENV DW_HOME=/opt/dw
ENV PATH=/opt/dw:$PATH

COPY --from=builder /dataweavetransformation-adapter /
COPY --from=builder /go/project/dw .

ENTRYPOINT ["/dataweavetransformation-adapter"]
