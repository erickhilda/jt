#!/bin/sh
set -e

# jt installer
# Usage: curl -sSfL https://raw.githubusercontent.com/erickhilda/jt/master/install.sh | sh
#    or: curl -sSfL https://raw.githubusercontent.com/erickhilda/jt/master/install.sh | sh -s -- -d /usr/local/bin

REPO="erickhilda/jt"
BINARY="jt"
INSTALL_DIR="/usr/local/bin"

usage() {
    echo "Usage: $0 [-d install_dir] [-v version]"
    echo "  -d    Installation directory (default: /usr/local/bin)"
    echo "  -v    Version to install (default: latest)"
    exit 1
}

while getopts "d:v:h" opt; do
    case "$opt" in
        d) INSTALL_DIR="$OPTARG" ;;
        v) VERSION="$OPTARG" ;;
        h) usage ;;
        *) usage ;;
    esac
done

detect_os() {
    os=$(uname -s | tr '[:upper:]' '[:lower:]')
    case "$os" in
        linux)  echo "linux" ;;
        darwin) echo "darwin" ;;
        mingw*|msys*|cygwin*) echo "windows" ;;
        *)
            echo "Error: unsupported OS: $os" >&2
            exit 1
            ;;
    esac
}

detect_arch() {
    arch=$(uname -m)
    case "$arch" in
        x86_64|amd64)  echo "amd64" ;;
        aarch64|arm64) echo "arm64" ;;
        *)
            echo "Error: unsupported architecture: $arch" >&2
            exit 1
            ;;
    esac
}

get_latest_version() {
    if command -v curl >/dev/null 2>&1; then
        curl -sSfL "https://api.github.com/repos/${REPO}/releases/latest" |
            grep '"tag_name"' | sed -E 's/.*"tag_name": *"([^"]+)".*/\1/'
    elif command -v wget >/dev/null 2>&1; then
        wget -qO- "https://api.github.com/repos/${REPO}/releases/latest" |
            grep '"tag_name"' | sed -E 's/.*"tag_name": *"([^"]+)".*/\1/'
    else
        echo "Error: curl or wget is required" >&2
        exit 1
    fi
}

download() {
    url="$1"
    output="$2"
    if command -v curl >/dev/null 2>&1; then
        curl -sSfL -o "$output" "$url"
    elif command -v wget >/dev/null 2>&1; then
        wget -qO "$output" "$url"
    fi
}

OS=$(detect_os)
ARCH=$(detect_arch)

if [ -z "$VERSION" ]; then
    echo "Fetching latest version..."
    VERSION=$(get_latest_version)
fi

if [ -z "$VERSION" ]; then
    echo "Error: could not determine latest version" >&2
    echo "Specify a version with: $0 -v v0.1.0" >&2
    exit 1
fi

# Strip leading 'v' for the archive name (goreleaser uses version without 'v' prefix)
VERSION_NUM="${VERSION#v}"

if [ "$OS" = "windows" ]; then
    ARCHIVE="${BINARY}_${VERSION_NUM}_${OS}_${ARCH}.zip"
else
    ARCHIVE="${BINARY}_${VERSION_NUM}_${OS}_${ARCH}.tar.gz"
fi

DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${VERSION}/${ARCHIVE}"

echo "Installing ${BINARY} ${VERSION} (${OS}/${ARCH})..."

TMP_DIR=$(mktemp -d)
trap 'rm -rf "$TMP_DIR"' EXIT

echo "Downloading ${DOWNLOAD_URL}..."
download "$DOWNLOAD_URL" "${TMP_DIR}/${ARCHIVE}"

echo "Extracting..."
if [ "$OS" = "windows" ]; then
    unzip -q "${TMP_DIR}/${ARCHIVE}" -d "$TMP_DIR"
else
    tar -xzf "${TMP_DIR}/${ARCHIVE}" -C "$TMP_DIR"
fi

# Install the binary
if [ -w "$INSTALL_DIR" ]; then
    cp "${TMP_DIR}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
    chmod +x "${INSTALL_DIR}/${BINARY}"
else
    echo "Elevated permissions required to install to ${INSTALL_DIR}"
    sudo cp "${TMP_DIR}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
    sudo chmod +x "${INSTALL_DIR}/${BINARY}"
fi

echo "Successfully installed ${BINARY} to ${INSTALL_DIR}/${BINARY}"
echo ""
echo "Run 'jt init' to get started."
