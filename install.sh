#!/bin/bash
set -euo pipefail

# Raycoon installer
# Usage: curl -fsSL https://raw.githubusercontent.com/AmirTlbyn/raycoon/main/install.sh | bash

REPO="AmirTlbyn/raycoon"
XRAY_REPO="XTLS/Xray-core"
INSTALL_DIR="/usr/local/bin"
XRAY_DIR="$HOME/.local/bin"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

info() { echo -e "${BLUE}[INFO]${NC} $*"; }
success() { echo -e "${GREEN}[OK]${NC} $*"; }
warn() { echo -e "${YELLOW}[WARN]${NC} $*"; }
error() { echo -e "${RED}[ERROR]${NC} $*" >&2; exit 1; }

detect_platform() {
    local os arch

    os="$(uname -s | tr '[:upper:]' '[:lower:]')"
    case "$os" in
        linux)  GOOS="linux" ;;
        darwin) GOOS="darwin" ;;
        *)      error "Unsupported OS: $os" ;;
    esac

    arch="$(uname -m)"
    case "$arch" in
        x86_64|amd64)   GOARCH="amd64" ;;
        aarch64|arm64)  GOARCH="arm64" ;;
        *)              error "Unsupported architecture: $arch" ;;
    esac

    info "Detected platform: ${GOOS}/${GOARCH}"
}

get_latest_release() {
    local repo="$1"
    local tag

    tag=$(curl -fsSL "https://api.github.com/repos/${repo}/releases/latest" \
        | grep '"tag_name"' | head -1 | cut -d'"' -f4)

    if [ -z "$tag" ]; then
        error "Failed to get latest release for ${repo}"
    fi

    echo "$tag"
}

get_installed_raycoon_version() {
    if [ -x "${INSTALL_DIR}/raycoon" ]; then
        "${INSTALL_DIR}/raycoon" --version 2>/dev/null \
            | grep -oE 'v?[0-9]+\.[0-9]+\.[0-9]+' | head -1 | sed 's/^v//'
    fi
}

get_installed_xray_version() {
    if [ -x "${XRAY_DIR}/xray" ]; then
        "${XRAY_DIR}/xray" --version 2>/dev/null \
            | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' | head -1
    fi
}

download_raycoon() {
    local version="$1"
    local tmpdir
    tmpdir=$(mktemp -d)
    trap "rm -rf '$tmpdir'" EXIT

    local filename="raycoon-${GOOS}-${GOARCH}"
    local url="https://github.com/${REPO}/releases/download/${version}/${filename}"

    info "Downloading raycoon ${version}..."
    curl -fsSL -o "${tmpdir}/raycoon" "$url" || error "Failed to download raycoon from ${url}"
    chmod +x "${tmpdir}/raycoon"

    info "Installing raycoon to ${INSTALL_DIR}/ (may require sudo)..."
    if [ -w "$INSTALL_DIR" ]; then
        mv "${tmpdir}/raycoon" "${INSTALL_DIR}/raycoon"
    else
        sudo mv "${tmpdir}/raycoon" "${INSTALL_DIR}/raycoon"
    fi
    success "raycoon ${version} installed to ${INSTALL_DIR}/raycoon"
}

download_xray() {
    local version="$1"
    local tmpdir
    tmpdir=$(mktemp -d)

    # Map to xray's naming convention
    local xray_os xray_arch
    case "$GOOS" in
        darwin) xray_os="macos" ;;
        linux)  xray_os="linux" ;;
    esac
    case "$GOARCH" in
        amd64) xray_arch="64" ;;
        arm64) xray_arch="arm64-v8a" ;;
    esac

    local filename="Xray-${xray_os}-${xray_arch}.zip"
    local url="https://github.com/${XRAY_REPO}/releases/download/${version}/${filename}"

    info "Downloading xray-core ${version}..."
    curl -fsSL -o "${tmpdir}/xray.zip" "$url" || error "Failed to download xray from ${url}"

    mkdir -p "$XRAY_DIR"

    info "Extracting xray to ${XRAY_DIR}/..."
    unzip -qo "${tmpdir}/xray.zip" -d "${tmpdir}/xray"

    # Install xray binary
    cp "${tmpdir}/xray/xray" "${XRAY_DIR}/xray"
    chmod +x "${XRAY_DIR}/xray"

    # Install geo files
    for f in geoip.dat geosite.dat; do
        if [ -f "${tmpdir}/xray/${f}" ]; then
            cp "${tmpdir}/xray/${f}" "${XRAY_DIR}/${f}"
        fi
    done

    rm -rf "$tmpdir"
    success "xray ${version} installed to ${XRAY_DIR}/"
}

setup_completions() {
    local shell_name
    shell_name="$(basename "$SHELL")"

    info "Setting up shell completions for ${shell_name}..."

    case "$shell_name" in
        zsh)
            local comp_dir="/usr/local/share/zsh/site-functions"
            local user_comp_dir="$HOME/.zsh/completions"

            if [ -w "$comp_dir" ] 2>/dev/null || [ -w "$(dirname "$comp_dir")" ] 2>/dev/null; then
                mkdir -p "$comp_dir"
                "${INSTALL_DIR}/raycoon" completion zsh > "${comp_dir}/_raycoon"
                success "Zsh completions installed to ${comp_dir}/_raycoon"
            else
                mkdir -p "$user_comp_dir"
                "${INSTALL_DIR}/raycoon" completion zsh > "${user_comp_dir}/_raycoon"
                success "Zsh completions installed to ${user_comp_dir}/_raycoon"
                if ! grep -q 'fpath.*\.zsh/completions' "$HOME/.zshrc" 2>/dev/null; then
                    warn "Add this to your ~/.zshrc to enable completions:"
                    echo '  fpath=(~/.zsh/completions $fpath)'
                    echo '  autoload -U compinit && compinit'
                fi
            fi
            ;;
        bash)
            local comp_dir="/etc/bash_completion.d"
            local user_comp_dir="$HOME/.local/share/bash-completion/completions"

            if [ -w "$comp_dir" ] 2>/dev/null; then
                "${INSTALL_DIR}/raycoon" completion bash > "${comp_dir}/raycoon"
                success "Bash completions installed to ${comp_dir}/raycoon"
            else
                mkdir -p "$user_comp_dir"
                "${INSTALL_DIR}/raycoon" completion bash > "${user_comp_dir}/raycoon"
                success "Bash completions installed to ${user_comp_dir}/raycoon"
            fi
            ;;
        fish)
            local comp_dir="$HOME/.config/fish/completions"
            mkdir -p "$comp_dir"
            "${INSTALL_DIR}/raycoon" completion fish > "${comp_dir}/raycoon.fish"
            success "Fish completions installed to ${comp_dir}/raycoon.fish"
            ;;
        *)
            warn "Unknown shell '${shell_name}'. Skipping completion setup."
            warn "You can manually generate completions with: raycoon completion [bash|zsh|fish]"
            ;;
    esac
}

create_dirs() {
    info "Creating data directories..."
    mkdir -p "$HOME/.config/raycoon"
    mkdir -p "$HOME/.local/share/raycoon"
    mkdir -p "$HOME/.cache/raycoon"
    success "Data directories created"
}

print_success() {
    local raycoon_version="$1"
    local xray_version="$2"
    local action="${3:-installed}"

    echo ""
    echo -e "${GREEN}============================================${NC}"
    if [ "$action" = "updated" ]; then
        echo -e "${GREEN}  Raycoon updated successfully!${NC}"
    else
        echo -e "${GREEN}  Raycoon installed successfully!${NC}"
    fi
    echo -e "${GREEN}============================================${NC}"
    echo ""
    echo "  raycoon : ${raycoon_version}"
    echo "  xray    : ${xray_version}"
    echo ""
    echo "  Quick start:"
    echo "    raycoon group create myproxy --subscription \"https://...\""
    echo "    raycoon sub update myproxy"
    echo "    raycoon test --all"
    echo "    raycoon connect --auto"
    echo ""
    echo "  Or use the TUI:"
    echo "    raycoon tui"
    echo ""
    echo -e "  ${YELLOW}Restart your shell or source your profile to enable completions.${NC}"
    echo ""
}

main() {
    echo ""
    echo "  Raycoon Installer"
    echo "  ========================"
    echo ""

    detect_platform

    info "Fetching latest release versions..."
    local raycoon_version xray_version
    raycoon_version=$(get_latest_release "$REPO")
    xray_version=$(get_latest_release "$XRAY_REPO")
    info "raycoon: ${raycoon_version}, xray: ${xray_version}"

    local current_raycoon current_xray action="installed"
    current_raycoon=$(get_installed_raycoon_version)
    current_xray=$(get_installed_xray_version)

    local raycoon_plain="${raycoon_version#v}"
    local xray_plain="${xray_version#v}"

    # Install or update raycoon
    if [ -n "$current_raycoon" ]; then
        if [ "$current_raycoon" = "$raycoon_plain" ]; then
            success "raycoon ${raycoon_version} is already up to date, skipping"
        else
            info "Updating raycoon: v${current_raycoon} → ${raycoon_version}"
            action="updated"
            download_raycoon "$raycoon_version"
        fi
    else
        download_raycoon "$raycoon_version"
    fi

    # Install or update xray
    if [ -n "$current_xray" ]; then
        if [ "$current_xray" = "$xray_plain" ]; then
            success "xray ${xray_version} is already up to date, skipping"
        else
            info "Updating xray: v${current_xray} → ${xray_version}"
            download_xray "$xray_version"
        fi
    else
        download_xray "$xray_version"
    fi

    create_dirs
    setup_completions
    print_success "$raycoon_version" "$xray_version" "$action"
}

main
