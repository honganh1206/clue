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

if [[ "$OSTYPE" == "linux-gnu"* ]]; then
    os="linux"
elif [[ "$OSTYPE" == "darwin"* ]]; then
    os="darwin"
else
    error "Unsupported operating system. Only Linux and macOS are currently supported."
    exit 1
fi

if [[ "$(uname -m)" == "x86_64" ]]; then
    arch="amd64"
elif [[ "$(uname -m)" == "aarch64" || "$(uname -m)" == "arm64" ]]; then
    arch="arm64"
else
    error "Unsupported architecture. clue requires a 64-bit system (x86_64 or arm64)."
    exit 1
fi

version="0.1.4"
base_url="https://github.com/honganh1206/clue/releases/download"
download_url="${base_url}/${version}/clue_${version}_${os}_${arch}"


status "Downloading clue version ${version} for ${os}/${arch}..."
if ! curl -fsSL -o clue ${download_url}; then
    error "Download failed. Please check your internet connection and try again."
    exit 1
fi

chmod +x clue

status "Installing clue..."

SUDO=
if [ "$(id -u)" -ne 0 ]; then
    # Running as root, no need for sudo
    if ! available sudo; then
        error "This script requires superuser permissions. Please re-run as root."
        exit 1
    fi

    SUDO="sudo"
fi

$SUDO mv clue /usr/local/bin/

# TODO: Uncomment this when working with 1st time config
#  if ! tlm config set shell auto &>/dev/null; then
#     error "tlm config set shell <auto> failed."
#     exit 1
# fi

# TODO: Uncomment this after done config file for app
# $SUDO chown $SUDO_USER ~/.clue.yaml

status "Type 'clue' to get started."
exit 0
