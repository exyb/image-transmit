#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
BIN_DIR="$PROJECT_DIR/embedded/bin"
TMP_DIR=$(mktemp -d)

SKOPEO_VERSION="${SKOPEO_VERSION:-v1.16.1}"
CONTAINERD_VER="${CONTAINERD_VER:-v1.7.28}"
CRICTL_VERSION="${CRICTL_VERSION:-v1.31.0}"
NERDCTL_VERSION="${NERDCTL_VERSION:-v2.0.2}"
REGCTL_VERSION="${REGCTL_VERSION:-v0.7.2}"
MC_VERSION="${MC_VERSION:-RELEASE.2025-04-16T18-10-43Z}"

OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"
case "$ARCH" in
    x86_64)  ARCH="amd64" ;;
    aarch64) ARCH="arm64" ;;
esac

echo "=== Downloading embedded tools for ${OS}/${ARCH} ==="
echo ""
echo "  [container] skopeo:   $SKOPEO_VERSION"
echo "  [container] ctr:      $CONTAINERD_VER (containerd)"
echo "  [container] crictl:   $CRICTL_VERSION"
echo "  [container] nerdctl:  $NERDCTL_VERSION"
echo "  [container] regctl:   $REGCTL_VERSION"
echo "  [storage]   mc:       $MC_VERSION"
echo "  [database]  usql:     (build from source)"
echo "  [database]  redis-cli: (copy from system)"
echo ""

mkdir -p "$BIN_DIR"

download() {
    local url="$1"
    local output="$2"
    echo "  Downloading: $url"
    curl -fSL -o "$output" "$url"
}

compress_and_place() {
    local src="$1"
    local name="$2"
    local embed_name
    embed_name=$(echo "$name" | tr '-' '_')
    chmod +x "$src"
    gzip -f -k "$src"
    mv "${src}.gz" "$BIN_DIR/${embed_name}.gz"
    echo "  -> $BIN_DIR/${embed_name}.gz"
}

echo "[1/8] Downloading skopeo..."
SKOPEO_URL="https://github.com/lework/skopeo-binary/releases/download/${SKOPEO_VERSION}/skopeo-${OS}-${ARCH}"
download "$SKOPEO_URL" "$TMP_DIR/skopeo"
compress_and_place "$TMP_DIR/skopeo" "skopeo"

echo "[2/8] Downloading ctr (from containerd)..."
CONTAINERD_URL="https://github.com/containerd/containerd/releases/download/${CONTAINERD_VER}/containerd-${CONTAINERD_VER#v}-${OS}-${ARCH}.tar.gz"
download "$CONTAINERD_URL" "$TMP_DIR/containerd.tar.gz"
tar xzf "$TMP_DIR/containerd.tar.gz" -C "$TMP_DIR" bin/ctr 2>/dev/null || true
if [ ! -f "$TMP_DIR/bin/ctr" ]; then
    tar xzf "$TMP_DIR/containerd.tar.gz" -C "$TMP_DIR"
fi
compress_and_place "$TMP_DIR/bin/ctr" "ctr"

echo "[3/8] Downloading crictl..."
CRICTL_URL="https://github.com/kubernetes-sigs/cri-tools/releases/download/${CRICTL_VERSION}/crictl-${CRICTL_VERSION}-${OS}-${ARCH}.tar.gz"
download "$CRICTL_URL" "$TMP_DIR/crictl.tar.gz"
tar xzf "$TMP_DIR/crictl.tar.gz" -C "$TMP_DIR" crictl
compress_and_place "$TMP_DIR/crictl" "crictl"

echo "[4/8] Downloading nerdctl..."
NERDCTL_URL="https://github.com/containerd/nerdctl/releases/download/${NERDCTL_VERSION}/nerdctl-${NERDCTL_VERSION#v}-${OS}-${ARCH}.tar.gz"
download "$NERDCTL_URL" "$TMP_DIR/nerdctl.tar.gz"
tar xzf "$TMP_DIR/nerdctl.tar.gz" -C "$TMP_DIR" nerdctl
compress_and_place "$TMP_DIR/nerdctl" "nerdctl"

echo "[5/8] Downloading regctl..."
REGCTL_URL="https://github.com/regclient/regclient/releases/download/${REGCTL_VERSION}/regctl-${OS}-${ARCH}"
download "$REGCTL_URL" "$TMP_DIR/regctl"
compress_and_place "$TMP_DIR/regctl" "regctl"

echo "[6/8] Downloading mc (MinIO Client)..."
MC_URL="https://dl.min.io/client/mc/release/${OS}-${ARCH}/mc"
download "$MC_URL" "$TMP_DIR/mc"
compress_and_place "$TMP_DIR/mc" "mc"

echo "[7/8] Building usql (universal SQL client)..."
if command -v go &>/dev/null; then
    USQL_BIN="$(go env GOPATH)/bin/usql"
    if [ -x "$USQL_BIN" ]; then
        echo "  Using existing usql: $USQL_BIN"
        compress_and_place "$USQL_BIN" "usql"
    else
        echo "  Building usql with 'most' drivers (this may take a while)..."
        if go install -tags 'most' github.com/xo/usql@latest 2>/dev/null; then
            compress_and_place "$USQL_BIN" "usql"
        else
            echo "  WARNING: usql build failed, skipping"
        fi
    fi
else
    echo "  WARNING: Go not found, skipping usql build"
fi

echo "[8/8] Copying redis-cli from system..."
if command -v redis-cli &>/dev/null; then
    compress_and_place "$(which redis-cli)" "redis-cli"
else
    echo "  WARNING: redis-cli not found on system, skipping"
fi

rm -rf "$TMP_DIR"

echo ""
echo "=== All embedded tools downloaded and compressed ==="
echo ""
echo "Tool mapping (embed filename -> runtime name):"
for f in "$BIN_DIR"/*.gz; do
    embed_name=$(basename "$f" .gz)
    runtime_name=$(echo "$embed_name" | tr '_' '-')
    size=$(ls -lh "$f" | awk '{print $5}')
    echo "  $embed_name.gz -> $runtime_name ($size)"
done
echo ""
echo "Total size: $(du -sh "$BIN_DIR" | awk '{print $1}')"
