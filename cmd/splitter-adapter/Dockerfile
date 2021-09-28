# Copyright 2021 TriggerMesh Inc.
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

FROM golang:1.17-bullseye AS builder

ENV CGO_ENABLED 0
ENV GOOS linux
ENV GOARCH amd64

WORKDIR /go/src/splitter-adapter

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN BIN_OUTPUT_DIR=/bin make splitter-adapter && \
    mkdir /kodata && \
    mv .git/* /kodata/ && \
    rm -rf ${GOPATH} && \
    rm -rf ${HOME}/.cache

FROM registry.access.redhat.com/ubi8/ubi-minimal

LABEL name "TriggerMesh Splitter"
LABEL vendor "TriggerMesh Inc."
LABEL version "v0.1.0"
LABEL release "1"
LABEL summary "TriggerMesh Routing Splitter"
LABEL description "TriggerMesh Splitter is a part of TriggerMesh Routing solution"

COPY LICENSE EULA.pdf /licenses/
COPY --from=builder /kodata/ ${KO_DATA_PATH}/
COPY --from=builder /bin/splitter-adapter /

ENTRYPOINT ["/splitter-adapter"]
