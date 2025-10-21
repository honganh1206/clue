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

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
VERSION=$(cat "$SCRIPT_DIR/../VERSION" 2>/dev/null || echo "0.2.2")
BASE_URL="https://github.com/honganh1206/clue/releases/download"
DOWNLOAD_URL="${BASE_URL}/${VERSION}/clue_${VERSION}_${OS}_${ARCH}"

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

CLUE_INSTALL_DIR=$(dirname ${BINDIR})

status "Downloading clue version ${VERSION} for ${OS}/${ARCH}..."

if ! curl -fsSL -o clue ${DOWNLOAD_URL}; then
    error "Download failed. Please check your internet connection and try again."
    exit 1
fi

chmod +x clue

if [ -d "$CLUE_INSTALL_DIR/clue" ] ; then
    status "Cleaning up old version at $OLLAMA_INSTALL_DIR/ollama"
    $SUDO rm -rf "$OLLAMA_INSTALL_DIR/ollama"
fi

# Allow clue.service to connect to local sqlite3
# since when we run clue.service with systemd, the current user is root,
# and thus cannot connect to local DBs.
# So we pre-fetch the current local user.
# TODO: Make sqlite3 DB system-wide?
CURRENT_USER="$(whoami)"

status "Installing clue to ${CLUE_INSTALL_DIR}..."

$SUDO mv clue $BINDIR

install_success() {
    status 'The Clue API is now available at 127.0.0.1:11435.'
    status 'Install complete. Run "clue" from the command line.'
}
trap install_success EXIT

configure_systemd() {
    # TODO: set HOME=/usr/share/clue for clue user
    # and set write access to ./local/.clue for clue user
    # and might be moving the DBs to shared user space?
    # if ! id clue >/dev/null 2>$1; then
    #     status "Creating clue user..."
    #     $SUDO useradd -r -s /bin/false -U -m -d /usr/share/clue clue
    # fi

    # status "Adding current user to clue group..."
    # $SUDO usermod -a -G clue $(whoami)
    status "Creating clue systemd service..."
    cat <<EOF | $SUDO tee /etc/systemd/system/clue.service >/dev/null
[Unit]
Description=Clue AI Coding Agent Server
After=network-online.target

[Service]
ExecStart=$BINDIR/clue serve
User=$CURRENT_USER
Group=$(id -gn)
Restart=always
RestartSec=3
Environment="PATH=$PATH"
Environment="HOME=$HOME"

[Install]
WantedBy=default.target
EOF

    status "Enabling and starting clue service..."
    $SUDO systemctl daemon-reload
    $SUDO systemctl enable clue.service
    $SUDO systemctl start clue.service
}

if available systemctl; then
    configure_systemd
fi