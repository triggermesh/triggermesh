# Copyright 2021 TriggerMesh Inc.

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

FROM golang:1.15-buster AS builder

ENV CGO_ENABLED 0
ENV GOOS linux
ENV GOARCH amd64

WORKDIR /go/src/targets

COPY go.mod go.sum ./
RUN go mod download

# NOTE: try not to modify the instructions above to keep cached layers for all targets

COPY ./ .
RUN BIN_OUTPUT_DIR=/bin make datadog-target-adapter && \
    mkdir /kodata && \
    mv .git/* /kodata/ && \
    rm -rf ${GOPATH} && \
    rm -rf ${HOME}/.cache

FROM registry.access.redhat.com/ubi8/ubi-minimal

LABEL name "Triggermesh Event Target Adapter for Datadog"
LABEL vendor "Triggermesh"
LABEL version "v0.1.0"
LABEL release "1"
LABEL summary "The Triggermesh Event Target Adapter for Datadog"
LABEL description "This is the Triggermesh Knative Event Target Adapter for Datadog"

ENV KO_DATA_PATH /kodata

COPY LICENSE EULA.pdf /licenses/
COPY --from=builder /kodata/ ${KO_DATA_PATH}/
COPY --from=builder /bin/datadog-target-adapter /datadog-target-adapter

ENTRYPOINT ["/datadog-target-adapter"]
