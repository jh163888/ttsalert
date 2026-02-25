#!/bin/bash
set -e

VERSION="${1:-latest}"
INSTALL_DIR="/usr/local/bin"
CONFIG_DIR="/etc/ttsalert"
DATA_DIR="/var/lib/ttsalert"
SERVICE_FILE="/etc/systemd/system/ttsalert.service"

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

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

check_root() {
    if [[ $EUID -ne 0 ]]; then
        log_error "This script must be run as root"
        exit 1
    fi
}

check_systemd() {
    if ! command -v systemctl &> /dev/null; then
        log_error "systemd is not available on this system"
        exit 1
    fi
}

install_dependencies() {
    log_info "Installing dependencies..."
    
    if command -v apt-get &> /dev/null; then
        apt-get update
        apt-get install -y python3 python3-pip ffmpeg
        pip3 install edge-tts --break-system-packages
    elif command -v yum &> /dev/null; then
        yum install -y epel-release
        yum install -y python3 python3-pip ffmpeg
        pip3 install edge-tts
    elif command -v dnf &> /dev/null; then
        dnf install -y python3 python3-pip ffmpeg
        pip3 install edge-tts
    else
        log_warn "Package manager not recognized. Please install python3 and edge-tts manually."
    fi
    
    if ! command -v edge-tts &> /dev/null; then
        log_error "Failed to install edge-tts"
        exit 1
    fi
    
    log_info "EdgeTTS installed successfully"
}

create_user() {
    if ! id "ttsalert" &>/dev/null; then
        log_info "Creating ttsalert user..."
        useradd --system --no-create-home --shell /bin/false ttsalert
    else
        log_info "User ttsalert already exists"
    fi
}

create_directories() {
    log_info "Creating directories..."
    
    mkdir -p "$CONFIG_DIR"
    mkdir -p "$DATA_DIR/audio"
    
    chown -R ttsalert:ttsalert "$DATA_DIR"
    chmod 755 "$DATA_DIR"
    chmod 755 "$DATA_DIR/audio"
}

install_binary() {
    log_info "Installing binary..."
    
    ARCH=$(uname -m)
    case $ARCH in
        x86_64)
            ARCH="amd64"
            ;;
        aarch64)
            ARCH="arm64"
            ;;
        *)
            log_error "Unsupported architecture: $ARCH"
            exit 1
            ;;
    esac
    
    if [[ "$VERSION" == "latest" ]]; then
        URL="https://github.com/jh163888/ttsalert/releases/latest/download/ttsalert_linux_${ARCH}.tar.gz"
    else
        URL="https://github.com/jh163888/ttsalert/releases/download/${VERSION}/ttsalert_linux_${ARCH}.tar.gz"
    fi
    
    log_info "Downloading from: $URL"
    
    TMP_DIR=$(mktemp -d)
    cd "$TMP_DIR"
    
    if command -v curl &> /dev/null; then
        curl -L -o ttsalert.tar.gz "$URL"
    elif command -v wget &> /dev/null; then
        wget -O ttsalert.tar.gz "$URL"
    else
        log_error "Neither curl nor wget is available"
        exit 1
    fi
    
    tar -xzf ttsalert.tar.gz
    chmod +x ttsalert
    mv ttsalert "$INSTALL_DIR/"
    
    cd - > /dev/null
    rm -rf "$TMP_DIR"
    
    log_info "Binary installed to $INSTALL_DIR/ttsalert"
}

install_config() {
    if [[ ! -f "$CONFIG_DIR/config.yaml" ]]; then
        log_info "Installing default configuration..."
        cat > "$CONFIG_DIR/config.yaml" << 'EOF'
server:
  host: "0.0.0.0"
  port: 8080

tts:
  use_edgetts: true
  voice: "zh-CN-XiaoxiaoNeural"
  rate: "+0%"
  volume: "+0%"
  pitch: "+0Hz"
  output_dir: "/var/lib/ttsalert/audio"
  audio_format: "mp3"

sip:
  server: "sip.example.com"
  port: 5060
  local_port: 5060
  username: "alert_user"
  password: "your_password"
  domain: "example.com"
  from_user: "alert_user"
  max_call_duration: 120s
  ring_timeout: 30s
  max_retries: 3
  retry_delay: 5s

queue:
  size: 100
  workers: 3

logging:
  level: "info"
EOF
        log_info "Config installed to $CONFIG_DIR/config.yaml"
        log_warn "Please edit the config file with your SIP server settings"
    else
        log_info "Config file already exists, skipping"
    fi
    
    chown ttsalert:ttsalert "$CONFIG_DIR/config.yaml"
    chmod 640 "$CONFIG_DIR/config.yaml"
}

install_service() {
    log_info "Installing systemd service..."
    
    cat > "$SERVICE_FILE" << 'EOF'
[Unit]
Description=TTS Alert Service - Voice Alert System
Documentation=https://github.com/ttsalert/ttsalert
After=network.target network-online.target
Wants=network-online.target

[Service]
Type=simple
User=ttsalert
Group=ttsalert

ExecStart=/usr/local/bin/ttsalert
Restart=on-failure
RestartSec=5s

Environment="PATH=/usr/local/bin:/usr/bin:/bin"
Environment="HOME=/var/lib/ttsalert"

WorkingDirectory=/var/lib/ttsalert

ReadWritePaths=/var/lib/ttsalert/audio
ReadWritePaths=/etc/ttsalert

NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=read-only
PrivateTmp=true

StandardOutput=journal
StandardError=journal
SyslogIdentifier=ttsalert

[Install]
WantedBy=multi-user.target
EOF

    systemctl daemon-reload
    log_info "Service file installed"
}

enable_service() {
    log_info "Enabling and starting service..."
    systemctl enable ttsalert
    systemctl start ttsalert
    
    sleep 2
    
    if systemctl is-active --quiet ttsalert; then
        log_info "TTS Alert service started successfully"
    else
        log_warn "Service may not have started. Check logs with: journalctl -u ttsalert -f"
    fi
}

show_status() {
    echo ""
    log_info "Installation complete!"
    echo ""
    echo "Service status:"
    systemctl status ttsalert --no-pager -l || true
    echo ""
    echo "Useful commands:"
    echo "  Start:   systemctl start ttsalert"
    echo "  Stop:    systemctl stop ttsalert"
    echo "  Restart: systemctl restart ttsalert"
    echo "  Status:  systemctl status ttsalert"
    echo "  Logs:    journalctl -u ttsalert -f"
    echo ""
    echo "Configuration: $CONFIG_DIR/config.yaml"
    echo "Audio files:   $DATA_DIR/audio"
    echo "Webhook URL:   http://localhost:8080/webhook/generic"
    echo ""
}

main() {
    echo "========================================"
    echo "  TTS Alert Installer"
    echo "  GitHub: github.com/jh163888/ttsalert"
    echo "========================================"
    echo ""
    
    check_root
    check_systemd
    install_dependencies
    create_user
    create_directories
    install_binary
    install_config
    install_service
    enable_service
    show_status
}

main "$@"
