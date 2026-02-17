#!/usr/bin/env bash
set -e

# commitai installer for macOS and Linux
# Usage: curl -fsSL https://raw.githubusercontent.com/kaiqui/commitai/main/scripts/install.sh | bash

REPO="kaiqui/commitai"
BINARY="commitai"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
CYAN='\033[0;36m'
YELLOW='\033[1;33m'
NC='\033[0m'

info()    { echo -e "${CYAN}→ $1${NC}"; }
success() { echo -e "${GREEN}✅ $1${NC}"; }
warn()    { echo -e "${YELLOW}⚠️  $1${NC}"; }
error()   { echo -e "${RED}❌ $1${NC}"; exit 1; }

# Detect OS and architecture
detect_platform() {
    local os arch

    case "$(uname -s)" in
        Darwin) os="darwin" ;;
        Linux)  os="linux" ;;
        *)      error "Unsupported OS: $(uname -s)" ;;
    esac

    case "$(uname -m)" in
        x86_64|amd64) arch="amd64" ;;
        arm64|aarch64) arch="arm64" ;;
        *) error "Unsupported architecture: $(uname -m)" ;;
    esac

    echo "${os}_${arch}"
}

# Get latest release version from GitHub
get_latest_version() {
    local version
    if command -v curl &>/dev/null; then
        version=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | sed -E 's/.*"tag_name": *"([^"]+)".*/\1/')
    elif command -v wget &>/dev/null; then
        version=$(wget -qO- "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | sed -E 's/.*"tag_name": *"([^"]+)".*/\1/')
    else
        error "curl or wget is required"
    fi
    echo "$version"
}

# Download and install
install_binary() {
    local platform=$1
    local version=$2
    local tmp_dir

    tmp_dir=$(mktemp -d)
    trap "rm -rf $tmp_dir" EXIT

    local filename="${BINARY}_${platform}.tar.gz"
    local url="https://github.com/${REPO}/releases/download/${version}/${filename}"

    info "Downloading ${BINARY} ${version} for ${platform}..."

    if command -v curl &>/dev/null; then
        curl -fsSL "$url" -o "${tmp_dir}/${filename}"
    else
        wget -q "$url" -O "${tmp_dir}/${filename}"
    fi

    info "Extracting..."
    tar -xzf "${tmp_dir}/${filename}" -C "$tmp_dir"

    # Check if we need sudo
    if [ -w "$INSTALL_DIR" ]; then
        mv "${tmp_dir}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
    else
        info "Requesting sudo to install to ${INSTALL_DIR}..."
        sudo mv "${tmp_dir}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
    fi

    chmod +x "${INSTALL_DIR}/${BINARY}"
}

main() {
    echo ""
    echo "  🤖 commitai installer"
    echo "  ─────────────────────"
    echo ""

    # Check if already installed
    if command -v commitai &>/dev/null; then
        current=$(commitai version 2>/dev/null | awk '{print $2}' || echo "unknown")
        warn "commitai is already installed (version: ${current})"
        printf "  Reinstall? [y/N]: "
        read -r answer
        if [[ "$answer" != "y" && "$answer" != "Y" ]]; then
            echo "Aborted."
            exit 0
        fi
    fi

    local platform
    platform=$(detect_platform)
    info "Detected platform: ${platform}"

    local version
    version=$(get_latest_version)
    if [ -z "$version" ]; then
        error "Could not determine latest version. Check https://github.com/${REPO}/releases"
    fi
    info "Latest version: ${version}"

    install_binary "$platform" "$version"

    success "commitai installed to ${INSTALL_DIR}/${BINARY}"
    echo ""
    echo "  Next steps:"
    echo "  1. Get your free Gemini API key: https://aistudio.google.com/app/apikey"
    echo "  2. Configure: commitai config --key YOUR_API_KEY"
    echo "  3. Stage files: git add <files>"
    echo "  4. Generate commit: commitai"
    echo ""
    echo "  Optional: set language to Portuguese:"
    echo "  commitai config --lang pt-br"
    echo ""
}

main "$@"
