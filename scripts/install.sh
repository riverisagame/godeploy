#!/bin/bash
set -e

# GoDeployer Install Script for Linux
# This script installs the latest GoDeployer binary and configures systemd.

REPO="riverisagame/godeploy" # Update if needed
BIN_DIR="/usr/local/bin"
CONF_DIR="/etc/godeployer"
DATA_DIR="/var/lib/godeployer"
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
    echo "Could not find latest release for $ARCH. Do you want to use a local binary instead? (Make sure ./godeployer_linux is available) [y/N]"
    read -r response
    if [[ "$response" =~ ^([yY][eE][sS]|[yY])$ ]]; then
        if [ -f "./godeployer_linux" ]; then
            cp ./godeployer_linux $BIN_DIR/$BIN_NAME
            chmod +x $BIN_DIR/$BIN_NAME
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
    mv /tmp/$EXTRACT_NAME $BIN_DIR/$BIN_NAME
    chmod +x $BIN_DIR/$BIN_NAME
    rm -f /tmp/$ASSET_NAME
fi

echo "Creating directories..."
mkdir -p $CONF_DIR
mkdir -p $DATA_DIR

echo "Setting up default config.yaml if not exists..."
if [ ! -f "$CONF_DIR/config.yaml" ]; then
cat <<EOF > $CONF_DIR/config.yaml
# GoDeployer Configuration
listen_addr: ":8080"
auth:
  enabled: true
  token: "change-me"
data_dir: "$DATA_DIR"
projects: {}
EOF
fi

echo "Creating systemd service..."
cat <<EOF > /etc/systemd/system/$SERVICE_NAME
[Unit]
Description=GoDeployer Service
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=$DATA_DIR
ExecStart=$BIN_DIR/$BIN_NAME --config=$CONF_DIR/config.yaml
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

echo "Reloading systemd daemon..."
systemctl daemon-reload
echo "Enabling and starting $SERVICE_NAME..."
systemctl enable --now $SERVICE_NAME

echo "Installation complete!"
echo "Configuration is located at $CONF_DIR/config.yaml"
echo "You can check the logs with: journalctl -fu $SERVICE_NAME"
