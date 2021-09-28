# Copyright 2020 TriggerMesh Inc.
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

WORKDIR /go/src/awscognitoidentitysource

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN BIN_OUTPUT_DIR=/bin make awscognitoidentitysource && \
    mkdir /kodata && \
    mv .git/* /kodata/ && \
    rm -rf ${GOPATH} && \
    rm -rf ${HOME}/.cache

FROM registry.access.redhat.com/ubi8/ubi-minimal

ARG VERSION

LABEL name "TriggerMesh AWS Cognito Identity Pool Event Source"
LABEL vendor "TriggerMesh"
LABEL version "$VERSION"
LABEL release "1"
LABEL summary "The TriggerMesh Cognito Identity Pool Source"
LABEL description "This is the TriggerMesh Knative Event Source for AWS Cognito Identity Pool"

# Emulate ko builds
# https://github.com/google/ko/blob/v0.5.0/README.md#including-static-assets
ENV KO_DATA_PATH /kodata

COPY --from=builder /kodata/ ${KO_DATA_PATH}/
COPY --from=builder /bin/awscognitoidentitysource /
COPY LICENSE EULA.pdf /licenses/

ENTRYPOINT ["/awscognitoidentitysource"]
