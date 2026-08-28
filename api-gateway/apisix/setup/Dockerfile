FROM debian:bookworm-slim

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        protobuf-compiler \
        libprotobuf-dev \
        curl \
        jq \
        openssl \
        bash \
        ca-certificates \
    && rm -rf /var/lib/apt/lists/*

COPY setup-routes.sh /usr/local/bin/setup-routes.sh
COPY routes.json /etc/apisix-setup/routes.json
RUN chmod +x /usr/local/bin/setup-routes.sh

ENTRYPOINT ["/usr/local/bin/setup-routes.sh"]
