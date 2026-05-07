# Pinned to ubuntu:noble-20260410 by digest for supply-chain integrity.
# Multi-arch manifest list covers linux/amd64 and linux/arm64 (the build
# matrix's two targets). Dependabot's docker ecosystem opens PRs that bump
# both the tag and the digest together.
FROM ubuntu:noble-20260410@sha256:c4a8d5503dfb2a3eb8ab5f807da5bc69a85730fb49b5cfca2330194ebcc41c7b

COPY ./bin/ /tmp/appcli/

# ca-certificates lets the binary speak TLS to outbound dependencies (e.g.
# a managed PostgreSQL endpoint). Pruned to apt's cache to keep the layer
# small.
RUN apt-get update && \
    apt-get install -y --no-install-recommends ca-certificates && \
    rm -rf /var/lib/apt/lists/* && \
    mkdir -p /tmp/appcli && \
    ls -lah /tmp/appcli && \
    ARCH="$(dpkg --print-architecture)" && \
    case "$ARCH" in \
        x86_64|amd64) ARCH="amd64" ;; \
        aarch64|arm64) ARCH="arm64" ;; \
        *) echo "Unsupported architecture: $ARCH" && exit 1 ;; \
    esac && \
    cp -v /tmp/appcli/appcli-linux-${ARCH} /usr/local/bin/appcli && \
    chmod 0755 /usr/local/bin/appcli && \
    chown root:root /usr/local/bin/appcli && \
    rm -rf /tmp/appcli && \
    groupadd --system --gid 10001 appcli && \
    useradd --system --uid 10001 --gid 10001 --shell /usr/sbin/nologin \
            --no-create-home --comment "appcli service account" appcli

USER 10001:10001

# Bare ENTRYPOINT lets `docker run image serve` and `docker run image copy ...`
# both work — Cobra owns argv parsing.
ENTRYPOINT ["appcli"]
CMD ["--help"]
