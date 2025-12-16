#!/bin/bash
#
# MageBox Installation Script
# https://magebox.dev
#
# Usage:
#   curl -fsSL https://get.magebox.dev | bash
#
# Environment variables:
#   MAGEBOX_VERSION - Install specific version (default: latest)
#   MAGEBOX_INSTALL_DIR - Installation directory (default: /usr/local/bin)
#

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
GITHUB_REPO="qoliber/magebox"
INSTALL_DIR="${MAGEBOX_INSTALL_DIR:-/usr/local/bin}"
BINARY_NAME="magebox"

print_banner() {
    local version="${1:-}"
    echo -e "${BLUE}"
    echo '                            _'
    echo '                           | |'
    echo ' _ __ ___   __ _  __ _  ___| |__   _____  __'
    echo '| '"'"'_ ` _ \ / _` |/ _` |/ _ \ '"'"'_ \ / _ \ \/ /'
    echo '| | | | | | (_| | (_| |  __/ |_) | (_) >  <'
    echo '|_| |_| |_|\__,_|\__, |\___|_.__/ \___/_/\_\'
    echo '                  __/ |'
    echo -n '                 |___/'
    echo -e "${NC}  ${version}"
    echo ""
    echo "Fast, native Magento development environment"
    echo ""
}

info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

success() {
    echo -e "${GREEN}[OK]${NC} $1"
}

warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1"
    exit 1
}

detect_os() {
    OS="$(uname -s)"
    case "$OS" in
        Linux*)     echo "linux" ;;
        Darwin*)    echo "darwin" ;;
        *)          error "Unsupported operating system: $OS" ;;
    esac
}

detect_arch() {
    ARCH="$(uname -m)"
    case "$ARCH" in
        x86_64)     echo "amd64" ;;
        amd64)      echo "amd64" ;;
        arm64)      echo "arm64" ;;
        aarch64)    echo "arm64" ;;
        *)          error "Unsupported architecture: $ARCH" ;;
    esac
}

get_latest_version() {
    curl -fsSL "https://api.github.com/repos/${GITHUB_REPO}/releases/latest" | \
        grep '"tag_name"' | \
        sed -E 's/.*"v([^"]+)".*/\1/'
}

download_binary() {
    local version="$1"
    local os="$2"
    local arch="$3"
    local url="https://github.com/${GITHUB_REPO}/releases/download/v${version}/magebox-${os}-${arch}"
    local tmp_file="/tmp/magebox-$$"

    # Use >&2 to send info to stderr so it doesn't get captured with the return value
    info "Downloading MageBox v${version} for ${os}/${arch}..." >&2

    if command -v curl &> /dev/null; then
        curl -fsSL "$url" -o "$tmp_file" || error "Download failed. Check if release exists: $url"
    elif command -v wget &> /dev/null; then
        wget -q "$url" -O "$tmp_file" || error "Download failed. Check if release exists: $url"
    else
        error "Neither curl nor wget found. Please install one of them."
    fi

    echo "$tmp_file"
}

verify_checksum() {
    local file="$1"
    local version="$2"
    local os="$3"
    local arch="$4"
    local checksum_url="https://github.com/${GITHUB_REPO}/releases/download/v${version}/magebox-${os}-${arch}.sha256"

    info "Verifying checksum..."

    local expected_checksum
    expected_checksum=$(curl -fsSL "$checksum_url" 2>/dev/null | awk '{print $1}')

    if [ -z "$expected_checksum" ]; then
        warn "Checksum file not found, skipping verification"
        return 0
    fi

    local actual_checksum
    if command -v sha256sum &> /dev/null; then
        actual_checksum=$(sha256sum "$file" | awk '{print $1}')
    elif command -v shasum &> /dev/null; then
        actual_checksum=$(shasum -a 256 "$file" | awk '{print $1}')
    else
        warn "No checksum tool found, skipping verification"
        return 0
    fi

    if [ "$expected_checksum" != "$actual_checksum" ]; then
        error "Checksum verification failed!"
    fi

    success "Checksum verified"
}

install_binary() {
    local tmp_file="$1"
    local install_path="${INSTALL_DIR}/${BINARY_NAME}"

    info "Installing to ${install_path}..."

    # Check if we need sudo
    if [ -w "$INSTALL_DIR" ]; then
        mv "$tmp_file" "$install_path"
        chmod +x "$install_path"
    else
        info "Requesting sudo access to install to ${INSTALL_DIR}"
        sudo mv "$tmp_file" "$install_path"
        sudo chmod +x "$install_path"
    fi

    success "Installed to ${install_path}"
}

setup_aliases() {
    local install_path="${INSTALL_DIR}/${BINARY_NAME}"

    echo ""
    echo -e "${BLUE}Short Command Aliases${NC}"
    echo ""
    echo "Create shorter command aliases for faster typing:"
    echo ""
    echo "  1) mbox        - recommended, descriptive"
    echo "  2) mb          - shortest (2 chars)"
    echo "  3) Both        - create both aliases"
    echo "  4) Skip        - use only 'magebox'"
    echo ""

    # Read user choice
    read -p "Choose [1-4, default: 1]: " choice
    choice="${choice:-1}"

    case "$choice" in
        1)
            create_alias "mbox" "$install_path"
            ;;
        2)
            create_alias "mb" "$install_path"
            ;;
        3)
            create_alias "mbox" "$install_path"
            create_alias "mb" "$install_path"
            ;;
        4)
            info "Skipping alias creation"
            ;;
        *)
            info "Invalid choice, skipping alias creation"
            ;;
    esac
}

create_alias() {
    local alias_name="$1"
    local target="$2"
    local alias_path="${INSTALL_DIR}/${alias_name}"

    info "Creating alias '${alias_name}'..."

    if [ -w "$INSTALL_DIR" ]; then
        ln -sf "$target" "$alias_path"
    else
        sudo ln -sf "$target" "$alias_path"
    fi

    success "Created alias: ${alias_name} -> magebox"
}

main() {
    # Check dependencies first (quietly)
    if ! command -v curl &> /dev/null && ! command -v wget &> /dev/null; then
        error "curl or wget is required but not installed"
    fi

    # Get version early so we can show it in the banner
    VERSION="${MAGEBOX_VERSION:-$(get_latest_version)}"
    if [ -z "$VERSION" ]; then
        error "Failed to determine version. Set MAGEBOX_VERSION or check your internet connection."
    fi

    # Now print banner with version
    print_banner "$VERSION"

    # Detect platform
    OS=$(detect_os)
    ARCH=$(detect_arch)
    info "Detected platform: ${OS}/${ARCH}"
    info "Version: ${VERSION}"

    # Download
    TMP_FILE=$(download_binary "$VERSION" "$OS" "$ARCH")

    # Verify
    verify_checksum "$TMP_FILE" "$VERSION" "$OS" "$ARCH"

    # Install
    install_binary "$TMP_FILE"

    # Setup aliases
    setup_aliases

    echo ""
    success "MageBox v${VERSION} installed successfully!"
    echo ""
    echo "Next steps:"
    echo "  1. Run 'mbox bootstrap' to set up dependencies"
    echo "  2. Navigate to your Magento project"
    echo "  3. Run 'mbox init' to configure your project"
    echo "  4. Run 'mbox start' to start services"
    echo ""
    echo "Documentation: https://magebox.dev"
    echo ""
}

main "$@"
