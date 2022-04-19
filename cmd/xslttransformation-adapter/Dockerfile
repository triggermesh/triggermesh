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

# (!) Debian 11 'bullseye' must be used in both the builder and final image to
# ensure the compatibility of the GNU libc.
FROM golang:1.18-bullseye as builder

RUN set -eux; \
    apt-get update; \
    apt-get install -y --no-install-recommends libxml2-dev libxslt1-dev liblzma-dev zlib1g-dev


WORKDIR /go/triggermesh

COPY . .
RUN go build -o /xslttransformation-adapter ./cmd/xslttransformation-adapter
# ldd /xslttransformation-adapter
# (i) Entries marked with a '*' are not included in the 'distroless:base' image.
#     linux-vdso.so.1
#   * libxml2.so.2 => /usr/lib/x86_64-linux-gnu/libxml2.so.2
#   * libxslt.so.1 => /usr/lib/x86_64-linux-gnu/libxslt.so.1
#   * libexslt.so.0 => /usr/lib/x86_64-linux-gnu/libexslt.so.0
#     libpthread.so.0 => /lib/x86_64-linux-gnu/libpthread.so.0
#     libc.so.6 => /lib/x86_64-linux-gnu/libc.so.6
#     libdl.so.2 => /lib/x86_64-linux-gnu/libdl.so.2
#   * libicuuc.so.67 => /usr/lib/x86_64-linux-gnu/libicuuc.so.67
#   * libz.so.1 => /lib/x86_64-linux-gnu/libz.so.1
#   * liblzma.so.5 => /lib/x86_64-linux-gnu/liblzma.so.5
#     libm.so.6 => /lib/x86_64-linux-gnu/libm.so.6
#   * libgcrypt.so.20 => /usr/lib/x86_64-linux-gnu/libgcrypt.so.20
#     /lib64/ld-linux-x86-64.so.2
#   * libicudata.so.67 => /usr/lib/x86_64-linux-gnu/libicudata.so.67
#   * libstdc++.so.6 => /usr/lib/x86_64-linux-gnu/libstdc++.so.6
#   * libgcc_s.so.1 => /lib/x86_64-linux-gnu/libgcc_s.so.1
#   * libgpg-error.so.0 => /lib/x86_64-linux-gnu/libgpg-error.so.0


FROM gcr.io/distroless/base-debian11:nonroot

# Ensure the /kodata entries used by Knative to augment the logger with the
# current VCS revision are present.
COPY --from=builder /go/triggermesh/.git/HEAD /go/triggermesh/.git/refs/ /kodata/
ENV KO_DATA_PATH=/kodata

# (!) COPY follows symlinks
COPY --from=builder \
    /usr/lib/x86_64-linux-gnu/libxml2.so.2 \
    /usr/lib/x86_64-linux-gnu/libxslt.so.1 \
    /usr/lib/x86_64-linux-gnu/libexslt.so.0 \
    /usr/lib/x86_64-linux-gnu/libicuuc.so.67 \
    /lib/x86_64-linux-gnu/libz.so.1 \
    /lib/x86_64-linux-gnu/liblzma.so.5 \
    /usr/lib/x86_64-linux-gnu/libgcrypt.so.20 \
    /usr/lib/x86_64-linux-gnu/libicudata.so.67 \
    /usr/lib/x86_64-linux-gnu/libstdc++.so.6 \
    /lib/x86_64-linux-gnu/libgcc_s.so.1 \
    /lib/x86_64-linux-gnu/libgpg-error.so.0 \
    /usr/lib/x86_64-linux-gnu/

COPY --from=builder /xslttransformation-adapter /

ENTRYPOINT ["/xslttransformation-adapter"]
