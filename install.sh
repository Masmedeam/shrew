#!/bin/sh
#
# Shrew - One-Command Installer
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/Masmedeam/shrew/main/install.sh | sh
#
# This script will automatically detect your OS and architecture, then download
# the latest pre-compiled binary of Shrew from GitHub Releases and install it
# into /usr/local/bin.

set -e

# --- Helper Functions ---
echo_info() {
  printf "\033[34m[INFO]\033[0m %s
" "$1"
}

echo_error() {
  printf "\033[31m[ERROR]\033[0m %s
" "$1" >&2
  exit 1
}

# --- Main Installation Logic ---
main() {
  REPO="Masmedeam/shrew"
  TARGET_DIR="/usr/local/bin"
  
  # 1. Determine OS and Architecture
  OS=$(uname -s | tr '[:upper:]' '[:lower:]')
  ARCH=$(uname -m)

  case "$ARCH" in
    x86_64 | amd64)
      ARCH="amd64"
      ;;
    aarch64 | arm64)
      ARCH="arm64"
      ;;
    *)
      echo_error "Unsupported architecture: $ARCH"
      ;;
  esac

  if [ "$OS" != "linux" ] && [ "$OS" != "darwin" ]; then
    echo_error "Unsupported OS: $OS. This script only supports Linux and macOS."
  fi
  
  # 2. Get the latest release tag from GitHub API
  echo_info "Fetching the latest version of Shrew..."
  LATEST_TAG=$(curl -s "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')

  if [ -z "$LATEST_TAG" ]; then
    echo_error "Could not fetch the latest release tag. Check the repository and your internet connection."
  fi
  
  echo_info "Latest version is $LATEST_TAG. Downloading binary for $OS/$ARCH..."

  # 3. Construct download URL and download the binary
  BINARY_NAME="shrew-${OS}-${ARCH}"
  DOWNLOAD_URL="https://github.com/$REPO/releases/download/$LATEST_TAG/$BINARY_NAME"
  
  # Use a temporary file for the download
  TMP_FILE=$(mktemp)
  
  if ! curl -fsSL "$DOWNLOAD_URL" -o "$TMP_FILE"; then
    echo_error "Failed to download binary from $DOWNLOAD_URL. Please check if a release for your platform exists."
  fi
  
  # 4. Install the binary
  echo_info "Installing Shrew to $TARGET_DIR..."
  
  if [ -w "$TARGET_DIR" ]; then
    SUDO=""
  else
    SUDO="sudo"
    echo_info "Sudo privileges are required to install to $TARGET_DIR."
  fi

  if ! chmod +x "$TMP_FILE"; then
    echo_error "Failed to make the binary executable."
  fi
  
  if ! $SUDO mv "$TMP_FILE" "$TARGET_DIR/shrew"; then
    echo_error "Failed to move binary to $TARGET_DIR. Please check permissions."
  fi
  
  # 5. Verify installation
  if ! command -v shrew >/dev/null; then
    echo_error "Installation failed. 'shrew' command not found in PATH."
  fi
  
  echo_info "Shrew was installed successfully!"
  echo_info "Run 'shrew' to get started."
}

# --- Run ---
main
