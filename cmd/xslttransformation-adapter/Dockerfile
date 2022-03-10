# (!) Debian 11 'bullseye' must be used in both the builder and final image to
# ensure the compatibility of the GNU libc.
FROM golang:1.17-bullseye as builder

RUN set -eux; \
    apt-get update; \
    apt-get install -y --no-install-recommends libxml2 libxml2-dev

WORKDIR ${GOPATH}/project

COPY . .
RUN go build -o /xslttransformation-adapter ./cmd/xslttransformation-adapter
# ldd /xslttransformation-adapter
# (i) Entries marked with a '*' are not included in the 'distroless:base' image.
#     linux-vdso.so.1
#   * libxml2.so.2 => /usr/lib/x86_64-linux-gnu/libxml2.so.2
#     libpthread.so.0 => /lib/x86_64-linux-gnu/libpthread.so.0
#     libc.so.6 => /lib/x86_64-linux-gnu/libc.so.6
#     libdl.so.2 => /lib/x86_64-linux-gnu/libdl.so.2
#   * libicuuc.so.67 => /usr/lib/x86_64-linux-gnu/libicuuc.so.67
#   * libz.so.1 => /lib/x86_64-linux-gnu/libz.so.1
#   * liblzma.so.5 => /lib/x86_64-linux-gnu/liblzma.so.5
#     libm.so.6 => /lib/x86_64-linux-gnu/libm.so.6
#     /lib64/ld-linux-x86-64.so.2
#   * libicudata.so.67 => /usr/lib/x86_64-linux-gnu/libicudata.so.67
#   * libstdc++.so.6 => /usr/lib/x86_64-linux-gnu/libstdc++.so.6
#   * libgcc_s.so.1 => /lib/x86_64-linux-gnu/libgcc_s.so.1

FROM gcr.io/distroless/base-debian11:nonroot

# (!) COPY follows symlinks
COPY --from=builder \
    /usr/lib/x86_64-linux-gnu/libxml2.so.2 \
    /usr/lib/x86_64-linux-gnu/libicuuc.so.67 \
    /lib/x86_64-linux-gnu/libz.so.1 \
    /lib/x86_64-linux-gnu/liblzma.so.5 \
    /usr/lib/x86_64-linux-gnu/libicudata.so.67 \
    /usr/lib/x86_64-linux-gnu/libstdc++.so.6 \
    /lib/x86_64-linux-gnu/libgcc_s.so.1 \
    /usr/lib/x86_64-linux-gnu/

COPY --from=builder /xslttransformation-adapter /

ENTRYPOINT ["/xslttransformation-adapter"]
