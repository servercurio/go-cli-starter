# Pinned to ubuntu:noble-20260410 by digest for supply-chain integrity.
# Multi-arch manifest list covers linux/amd64 and linux/arm64 (the build
# matrix's two targets). Dependabot's docker ecosystem opens PRs that bump
# both the tag and the digest together.
FROM ubuntu:noble-20260410@sha256:c4a8d5503dfb2a3eb8ab5f807da5bc69a85730fb49b5cfca2330194ebcc41c7b

COPY ./bin/ /tmp/appsvr/

# curl is needed for the HEALTHCHECK below; ca-certificates lets curl
# resolve the daemon's HTTPS endpoint when consumers point HEALTHCHECK at
# the TLS listener instead. Both are pruned to apt's cache to keep the
# layer small.
RUN apt-get update && \
    apt-get install -y --no-install-recommends curl ca-certificates && \
    rm -rf /var/lib/apt/lists/* && \
    mkdir -p /tmp/appsvr && \
    ls -lah /tmp/appsvr && \
    ARCH="$(dpkg --print-architecture)" && \
    case "$ARCH" in \
        x86_64|amd64) ARCH="amd64" ;; \
        aarch64|arm64) ARCH="arm64" ;; \
        *) echo "Unsupported architecture: $ARCH" && exit 1 ;; \
    esac && \
    cp -v /tmp/appsvr/appsvrd-linux-${ARCH} /usr/local/bin/appsvrd && \
    chmod 0755 /usr/local/bin/appsvrd && \
    chown root:root /usr/local/bin/appsvrd && \
    rm -rf /tmp/appsvr && \
    groupadd --system --gid 10001 appsvr && \
    useradd --system --uid 10001 --gid 10001 --shell /usr/sbin/nologin \
            --no-create-home --comment "appsvrd service account" appsvr

USER 10001:10001

# 8080 is the binary's HTTP default. Override the URL via your orchestrator
# (e.g. HEALTHCHECK in compose / k8s readinessProbe) when the daemon is
# reconfigured to bind elsewhere or to TLS only.
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
    CMD curl -sf http://127.0.0.1:8080/api/v1/livez || exit 1

CMD ["appsvrd"]
