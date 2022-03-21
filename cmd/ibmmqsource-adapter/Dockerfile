FROM golang:1.17-bullseye as builder

RUN apt-get update && \
    apt-get install -y curl && \
    curl https://public.dhe.ibm.com/ibmdl/export/pub/software/websphere/messaging/mqdev/redist/9.2.3.0-IBM-MQC-Redist-LinuxX64.tar.gz -o mq.tar.gz          && \
    mkdir -p /opt/mqm && \
    tar -C /opt/mqm -xzf mq.tar.gz

ENV MQ_INSTALLATION_PATH="/opt/mqm"
ENV CGO_LDFLAGS_ALLOW="-Wl,-rpath.*"
ENV CGO_LDFLAGS="-L$MQ_INSTALLATION_PATH/lib64 -Wl,-rpath,$MQ_INSTALLATION_PATH/lib64"
ENV CGO_CFLAGS="-I$MQ_INSTALLATION_PATH/inc"

WORKDIR /go/triggermesh

COPY go.mod .
COPY go.sum .
RUN go mod download

COPY . .

RUN GOOS=linux GOARCH=amd64 go build -v -o ibmmqsource-adapter ./cmd/ibmmqsource-adapter 


FROM debian:stable-slim

WORKDIR /opt/mqm/
COPY --from=builder /opt/mqm .
COPY --from=builder /go/triggermesh/ibmmqsource-adapter .

ENTRYPOINT ["./ibmmqsource-adapter"]
