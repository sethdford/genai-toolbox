#!/bin/bash
# Enterprise GenAI Toolbox - One-Line Installer
#
# Quick install: curl -fsSL https://raw.githubusercontent.com/sethdford/genai-toolbox-enterprise/main/scripts/install.sh | bash
#

set -e

REPO="sethdford/genai-toolbox-enterprise"
BINARY_NAME="genai-toolbox"
INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"
VERSION="${VERSION:-latest}"

echo ""
echo "🚀 Installing Enterprise GenAI Toolbox..."
echo ""

# Detect OS and architecture
detect_platform() {
  local os=""
  local arch=""

  # Detect OS
  case "$(uname -s)" in
    Darwin*)
      os="darwin"
      ;;
    Linux*)
      os="linux"
      ;;
    MINGW*|MSYS*|CYGWIN*)
      os="windows"
      ;;
    *)
      echo "❌ Unsupported operating system: $(uname -s)"
      exit 1
      ;;
  esac

  # Detect architecture
  case "$(uname -m)" in
    x86_64|amd64)
      arch="amd64"
      ;;
    arm64|aarch64)
      arch="arm64"
      ;;
    *)
      echo "❌ Unsupported architecture: $(uname -m)"
      exit 1
      ;;
  esac

  echo "$os" "$arch"
}

# Get latest release version
get_latest_version() {
  if [ "$VERSION" = "latest" ]; then
    curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" \
      | grep '"tag_name":' \
      | sed -E 's/.*"v([^"]+)".*/\1/' \
      | head -n 1
  else
    echo "$VERSION"
  fi
}

# Download and install
install_binary() {
  local os=$1
  local arch=$2
  local version=$3

  echo "Platform: $os/$arch"
  echo "Version: $version"
  echo ""

  # Determine archive format
  local ext="tar.gz"
  local binary_ext=""
  if [ "$os" = "windows" ]; then
    ext="zip"
    binary_ext=".exe"
  fi

  # Download URL
  local archive_name="${BINARY_NAME}-${os}-${arch}.${ext}"
  local download_url="https://github.com/$REPO/releases/download/v${version}/${archive_name}"

  echo "Downloading from: $download_url"

  # Create temp directory
  local tmp_dir=$(mktemp -d)
  trap "rm -rf $tmp_dir" EXIT

  # Download
  if ! curl -fsSL "$download_url" -o "$tmp_dir/$archive_name"; then
    echo "❌ Failed to download binary"
    echo ""
    echo "Please check:"
    echo "  - Version exists: https://github.com/$REPO/releases/tag/v${version}"
    echo "  - Platform supported: $os/$arch"
    exit 1
  fi

  echo "✓ Downloaded successfully"

  # Extract
  echo "Extracting..."
  if [ "$ext" = "zip" ]; then
    unzip -q "$tmp_dir/$archive_name" -d "$tmp_dir"
  else
    tar -xzf "$tmp_dir/$archive_name" -C "$tmp_dir"
  fi
  echo "✓ Extracted successfully"

  # Install
  mkdir -p "$INSTALL_DIR"
  local binary_path="$INSTALL_DIR/${BINARY_NAME}${binary_ext}"

  mv "$tmp_dir/${BINARY_NAME}${binary_ext}" "$binary_path"
  chmod +x "$binary_path"

  echo "✓ Installed to: $binary_path"
  echo ""
  echo "✅ Enterprise GenAI Toolbox installed successfully!"
  echo ""

  # Check if install dir is in PATH
  if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
    echo "⚠️  Note: $INSTALL_DIR is not in your PATH"
    echo ""
    echo "Add it to your shell profile:"
    echo "  echo 'export PATH=\"$INSTALL_DIR:\$PATH\"' >> ~/.bashrc"
    echo "  echo 'export PATH=\"$INSTALL_DIR:\$PATH\"' >> ~/.zshrc"
    echo ""
  fi

  echo "Usage:"
  echo "  $BINARY_NAME --help"
  echo "  $BINARY_NAME --tools-file tools.yaml"
  echo ""
}

# Main
main() {
  read -r os arch < <(detect_platform)
  local version=$(get_latest_version)

  install_binary "$os" "$arch" "$version"
}

main
