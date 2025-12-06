#!/bin/bash

# Exit immediately in case of non-zero status
set -e

status() { echo ">>> $*" >&2; }
error() { echo "ERROR $*"; }
warning() { echo "WARNING: $*"; }

print_message() {
    local message="$1"
    local color="$2"
    echo -e "\e[${color}m${message}\e[0m"
}

available() { command -v $1 >/dev/null; }

require() {
    local MISSING=''
    for TOOL in $*; do
        if ! available $TOOL; then
            MISSING="$MISSING $TOOL"
        fi
    done

    echo $MISSING
}

OS=$OSTYPE
case "$OS" in
    "linux-gnu") OS="linux" ;;
    *) "Unsupported operating system: $OS" ;;
esac

ARCH=$(uname -m)
case "$ARCH" in
    x86_64) ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    *) error "Unsupported architecture: $ARCH" ;;
esac

PROJECT_ROOT=$(git rev-parse --show-toplevel)
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
VERSION=$(cat "$PROJECT_ROOT/VERSION" 2>/dev/null)
BASE_URL="https://github.com/honganh1206/tinker/releases/download"
DOWNLOAD_URL="${BASE_URL}/${VERSION}/tinker_${VERSION}_${OS}_${ARCH}"

SUDO=
if [ "$(id -u)" -ne 0 ]; then
    # Running as root, no need for sudo
    if ! available sudo; then
        error "This script requires superuser permissions. Please re-run as root."
    fi

    SUDO="sudo"
fi

NEEDS=$(require curl grep tee)
if [ -n "$NEEDS" ]; then
    status "ERROR: The following tools are required but missing:"
    for NEED in $NEEDS; do
        echo "  - $NEED"
    done
    exit 1
fi


for BINDIR in /usr/local/bin /usr/bin /bin; do
    echo $PATH | grep -q $BINDIR && break || continue
done

TINKER_INSTALL_DIR=$(dirname ${BINDIR})

status "Downloading tinker version ${VERSION} for ${OS}/${ARCH}..."

if ! curl -fsSL -o tinker ${DOWNLOAD_URL}; then
    error "Download failed. Please check your internet connection and try again."
    exit 1
fi

chmod +x tinker

if [ -d "$TINKER_INSTALL_DIR/tinker" ] ; then
    status "Cleaning up old version at $OLLAMA_INSTALL_DIR/ollama"
    $SUDO rm -rf "$OLLAMA_INSTALL_DIR/ollama"
fi

# Allow tinker.service to connect to local sqlite3
# since when we run tinker.service with systemd, the current user is root,
# and thus cannot connect to local DBs.
# So we pre-fetch the current local user.
# TODO: Make sqlite3 DB system-wide?
CURRENT_USER="$(whoami)"

status "Installing tinker to ${TINKER_INSTALL_DIR}..."

$SUDO mv tinker $BINDIR

install_success() {
    status 'The Tinker API is now available at 127.0.0.1:11435.'
    status 'Install complete. Run "tinker" from the command line.'
}
trap install_success EXIT

configure_systemd() {
    # TODO: set HOME=/usr/share/tinker for tinker user
    # and set write access to ./local/.tinker for tinker user
    # and might be moving the DBs to shared user space?
    # if ! id tinker >/dev/null 2>$1; then
    #     status "Creating tinker user..."
    #     $SUDO useradd -r -s /bin/false -U -m -d /usr/share/tinker tinker
    # fi

    # status "Adding current user to tinker group..."
    # $SUDO usermod -a -G tinker $(whoami)
    status "Creating tinker systemd service..."
    cat <<EOF | $SUDO tee /etc/systemd/system/tinker.service >/dev/null
[Unit]
Description=Tinker AI Coding Agent Server
After=network-online.target

[Service]
ExecStart=$BINDIR/tinker serve
User=$CURRENT_USER
Group=$(id -gn)
Restart=always
RestartSec=3
Environment="PATH=$PATH"
Environment="HOME=$HOME"

[Install]
WantedBy=default.target
EOF

    status "Enabling and starting tinker service..."
    $SUDO systemctl daemon-reload
    $SUDO systemctl enable tinker.service
    $SUDO systemctl start tinker.service
}

if available systemctl; then
    configure_systemd
fi
