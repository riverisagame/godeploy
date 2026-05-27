#!/bin/bash
set -e

# GoDeployer Update Script for Linux
# This script stops the current service, updates the binary, and restarts it.

REPO="riverisagame/godeploy" # Update if needed
BIN_DIR="/usr/local/bin"
BIN_NAME="godeployer"
SERVICE_NAME="godeployer.service"

if [ "$EUID" -ne 0 ]; then
  echo "Please run as root"
  exit 1
fi

echo "Detecting system architecture..."
ARCH=$(uname -m)
case $ARCH in
  x86_64)
    ASSET_NAME="godeployer-linux-amd64.tar.gz"
    EXTRACT_NAME="godeployer-linux-amd64"
    ;;
  aarch64|arm64)
    ASSET_NAME="godeployer-linux-arm64.tar.gz"
    EXTRACT_NAME="godeployer-linux-arm64"
    ;;
  *)
    echo "Unsupported architecture: $ARCH"
    exit 1
    ;;
esac

echo "Fetching latest release from GitHub..."
LATEST_RELEASE_URL=$(curl -s "https://api.github.com/repos/$REPO/releases/latest" | grep "browser_download_url.*$ASSET_NAME" | cut -d '"' -f 4)

if [ -z "$LATEST_RELEASE_URL" ]; then
    echo "Could not find latest release for $ARCH."
    echo "Do you want to update from a local binary? (Make sure ./godeployer_linux is available) [y/N]"
    read -r response
    if [[ "$response" =~ ^([yY][eE][sS]|[yY])$ ]]; then
        if [ -f "./godeployer_linux" ]; then
            echo "Stopping $SERVICE_NAME..."
            systemctl stop $SERVICE_NAME
            echo "Copying local binary..."
            cp ./godeployer_linux $BIN_DIR/$BIN_NAME
            chmod +x $BIN_DIR/$BIN_NAME
            echo "Starting $SERVICE_NAME..."
            systemctl start $SERVICE_NAME
            echo "Update complete!"
        else
            echo "Local binary ./godeployer_linux not found. Aborting."
            exit 1
        fi
    else
        exit 1
    fi
else
    echo "Downloading $LATEST_RELEASE_URL..."
    curl -L -o /tmp/$ASSET_NAME "$LATEST_RELEASE_URL"
    tar -xzf /tmp/$ASSET_NAME -C /tmp/
    
    echo "Stopping $SERVICE_NAME..."
    systemctl stop $SERVICE_NAME
    
    echo "Replacing binary..."
    mv /tmp/$EXTRACT_NAME $BIN_DIR/$BIN_NAME
    chmod +x $BIN_DIR/$BIN_NAME
    rm -f /tmp/$ASSET_NAME
    
    echo "Starting $SERVICE_NAME..."
    systemctl start $SERVICE_NAME
    
    echo "Update complete!"
    $BIN_DIR/$BIN_NAME -version || true
fi
