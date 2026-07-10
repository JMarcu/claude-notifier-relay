#!/usr/bin/env bash
# Feature install script (containers.dev spec): runs once, as root, at
# devcontainer image build time. PORT and TARGETHOST are injected by the
# Dev Containers CLI from this feature's "port" / "targetHost" options.
set -euo pipefail

PORT="${PORT:-47291}"
TARGETHOST="${TARGETHOST:-host.docker.internal}"

ARCH="$(uname -m)"
case "$ARCH" in
  x86_64) BIN_ARCH=amd64 ;;
  aarch64|arm64) BIN_ARCH=arm64 ;;
  *)
    echo "cn-relay: unsupported architecture: $ARCH" >&2
    exit 1
    ;;
esac

BIN_SRC="./bin/cn-relay-linux-${BIN_ARCH}"
if [ ! -f "$BIN_SRC" ]; then
  echo "cn-relay: missing prebuilt binary for linux/${BIN_ARCH} (expected ${BIN_SRC})" >&2
  exit 1
fi

install -m 0755 "$BIN_SRC" /usr/local/bin/cn-relay

INSTALL_DIR=/usr/local/share/cn-relay
mkdir -p "$INSTALL_DIR"

cat > "$INSTALL_DIR/entrypoint.sh" <<EOF
#!/bin/sh
# Backgrounded by the Dev Containers CLI's entrypoint chaining before it
# execs the container's normal command.
/usr/local/bin/cn-relay -port "${PORT}" -target "${TARGETHOST}" &
exec "\$@"
EOF
chmod +x "$INSTALL_DIR/entrypoint.sh"

echo "cn-relay: installed, will listen on 127.0.0.1:${PORT} -> ${TARGETHOST}:${PORT}"
