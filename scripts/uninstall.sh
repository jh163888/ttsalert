#!/bin/bash
set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

check_root() {
    if [[ $EUID -ne 0 ]]; then
        log_error "This script must be run as root"
        exit 1
    fi
}

main() {
    echo "========================================"
    echo "  TTS Alert Uninstaller"
    echo "  GitHub: github.com/jh163888/ttsalert"
    echo "========================================"
    echo ""
    
    check_root
    
    read -p "Are you sure you want to uninstall TTS Alert? [y/N] " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        log_info "Uninstall cancelled"
        exit 0
    fi
    
    log_info "Stopping service..."
    systemctl stop ttsalert || true
    systemctl disable ttsalert || true
    
    log_info "Removing service file..."
    rm -f /etc/systemd/system/ttsalert.service
    systemctl daemon-reload
    
    log_info "Removing binary..."
    rm -f /usr/local/bin/ttsalert
    
    read -p "Remove configuration files? [y/N] " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        log_info "Removing configuration..."
        rm -rf /etc/ttsalert
    fi
    
    read -p "Remove audio files and data? [y/N] " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        log_info "Removing data..."
        rm -rf /var/lib/ttsalert
    fi
    
    read -p "Remove ttsalert user? [y/N] " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        log_info "Removing user..."
        userdel ttsalert || true
    fi
    
    log_info "Uninstall complete"
}

main "$@"
