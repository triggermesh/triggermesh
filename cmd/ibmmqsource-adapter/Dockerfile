# Copyright 2022 TriggerMesh Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

FROM golang:1.18-bullseye as builder

RUN apt-get update && \
    apt-get install -y curl && \
    curl https://public.dhe.ibm.com/ibmdl/export/pub/software/websphere/messaging/mqdev/redist/9.3.0.0-IBM-MQC-Redist-LinuxX64.tar.gz -o mq.tar.gz          && \
    mkdir -p /opt/mqm && \
    tar -C /opt/mqm -xzf mq.tar.gz

ENV MQ_INSTALLATION_PATH="/opt/mqm"
ENV CGO_LDFLAGS_ALLOW="-Wl,-rpath.*"
ENV CGO_LDFLAGS="-L$MQ_INSTALLATION_PATH/lib64 -Wl,-rpath,$MQ_INSTALLATION_PATH/lib64"
ENV CGO_CFLAGS="-I$MQ_INSTALLATION_PATH/inc"

WORKDIR /go/triggermesh

COPY . .

RUN GOOS=linux GOARCH=amd64 go build -v -o ibmmqsource-adapter ./cmd/ibmmqsource-adapter


FROM debian:stable-slim

# Ensure the /kodata entries used by Knative to augment the logger with the
# current VCS revision are present.
COPY --from=builder /go/triggermesh/.git/HEAD /go/triggermesh/.git/refs/ /kodata/
ENV KO_DATA_PATH=/kodata

WORKDIR /opt/mqm/
COPY --from=builder /opt/mqm .
COPY --from=builder /go/triggermesh/ibmmqsource-adapter .

ENTRYPOINT ["./ibmmqsource-adapter"]
