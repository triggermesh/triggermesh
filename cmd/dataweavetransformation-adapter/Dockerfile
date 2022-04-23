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

RUN set -eux; \
    apt-get update; \
    apt-get install -y --no-install-recommends unzip

WORKDIR /go/triggermesh
ENV DW_VERSION="1.0.19"

RUN  curl -sSLO https://github.com/mulesoft-labs/data-weave-cli/releases/download/v$DW_VERSION/dw-$DW_VERSION-Linux && \
    unzip -p dw-$DW_VERSION-Linux 'bin/dw' > dw && chmod +x dw
COPY . .
RUN go build -o /dataweavetransformation-adapter ./cmd/dataweavetransformation-adapter

FROM debian:stable-slim

# Ensure the /kodata entries used by Knative to augment the logger with the
# current VCS revision are present.
COPY --from=builder /go/triggermesh/.git/HEAD /go/triggermesh/.git/refs/ /kodata/
ENV KO_DATA_PATH=/kodata

WORKDIR /tmp/dw
ENV DW_HOME=/tmp/dw

COPY --from=builder /dataweavetransformation-adapter /
COPY --from=builder /go/triggermesh/dw /usr/local/bin/.

ENTRYPOINT ["/dataweavetransformation-adapter"]
