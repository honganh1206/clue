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

status "Clue installed successfully!"

echo ""
read -p "Do you want to install and start the clue service? (y/N): " install_service
case $install_service in
    [Yy]* )
        status "Installing clue systemd user service..."

        mkdir -p ~/.config/systemd/user

        if [ -f "clue.service" ]; then
            cp clue.service ~/.config/systemd/user/
        fi

        if systemctl --user daemon-reload; then
            if systemctl --user enable clue.service; then
                if systemctl --user start clue.service; then
                    print_message "✓ Clue service installed and started!" "32"
                    status "Server is running on http://localhost:11435"
                    status "Use 'systemctl --user status clue' to check status"
                    status "Use 'journalctl --user -u clue -f' to view logs"
                    status "Use 'systemctl --user stop clue' to stop the server"
                else
                    warning "Failed to start clue service"
                    status "You can start it manually with: systemctl --user start clue"
                fi
            else
                warning "Failed to enable clue service"
            fi
        else
            warning "Failed to reload systemd user daemon"
        fi
        ;;
    * )
        status "Service not installed. You can run clue manually with:"
        status "  clue serve          # Run in foreground"
        status "Or install the service later by copying clue.service to ~/.config/systemd/user/"
        ;;
esac

echo ""
status "Installation complete!"
status "Type 'clue' to get started."
status "Use 'clue serve --help' to see server options."
exit 0
