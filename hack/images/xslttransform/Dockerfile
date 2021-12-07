FROM debian:11-slim

RUN apt update && apt install -y --no-install-recommends libxml2 \
      && rm -rf /var/lib/apt/lists/* \
      && rm -rf /usr/share/
