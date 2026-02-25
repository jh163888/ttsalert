# TTS Alert 安装指南

## 快速安装 (推荐)

一键安装脚本（需要 root 权限）:

```bash
curl -fsSL https://raw.githubusercontent.com/ttsalert/ttsalert/main/scripts/install.sh | bash -
```

或下载后执行:

```bash
wget https://raw.githubusercontent.com/ttsalert/ttsalert/main/scripts/install.sh
chmod +x install.sh
sudo ./install.sh
```

## 手动安装

### 1. 安装依赖

**Debian/Ubuntu:**
```bash
sudo apt-get update
sudo apt-get install -y python3 python3-pip ffmpeg
pip3 install edge-tts --break-system-packages
```

**CentOS/RHEL:**
```bash
sudo yum install -y epel-release
sudo yum install -y python3 python3-pip ffmpeg
pip3 install edge-tts
```

**Fedora:**
```bash
sudo dnf install -y python3 python3-pip ffmpeg
pip3 install edge-tts
```

### 2. 创建用户和目录

```bash
sudo useradd --system --no-create-home --shell /bin/false ttsalert
sudo mkdir -p /etc/ttsalert
sudo mkdir -p /var/lib/ttsalert/audio
sudo chown -R ttsalert:ttsalert /var/lib/ttsalert
```

### 3. 安装二进制文件

从 Release 页面下载:
```bash
wget https://github.com/ttsalert/ttsalert/releases/latest/download/ttsalert_linux_amd64.tar.gz
tar -xzf ttsalert_linux_amd64.tar.gz
sudo mv ttsalert /usr/local/bin/
sudo chmod +x /usr/local/bin/ttsalert
```

或编译安装:
```bash
git clone https://github.com/ttsalert/ttsalert.git
cd ttsalert
make build
sudo make install
```

### 4. 配置

```bash
sudo cp configs/config.example.yaml /etc/ttsalert/config.yaml
sudo vi /etc/ttsalert/config.yaml
```

必填配置:
```yaml
sip:
  server: "你的 SIP 服务器"
  username: "SIP 用户名"
  password: "SIP 密码"
  domain: "SIP 域"
```

### 5. 安装 systemd 服务

```bash
sudo cp ttsalert.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable ttsalert
sudo systemctl start ttsalert
```

### 6. 验证

```bash
systemctl status ttsalert
journalctl -u ttsalert -f
```

测试 Webhook:
```bash
curl -X POST http://localhost:8080/webhook/generic \
  -H "Content-Type: application/json" \
  -d '{
    "title": "测试告警",
    "message": "这是一条测试消息",
    "host": "test-server",
    "phone_numbers": ["13800138000"]
  }'
```

## 服务管理

```bash
# 启动
sudo systemctl start ttsalert

# 停止
sudo systemctl stop ttsalert

# 重启
sudo systemctl restart ttsalert

# 查看状态
sudo systemctl status ttsalert

# 查看日志
sudo journalctl -u ttsalert -f

# 开机自启
sudo systemctl enable ttsalert

# 禁用开机自启
sudo systemctl disable ttsalert
```

## 卸载

```bash
curl -fsSL https://raw.githubusercontent.com/ttsalert/ttsalert/main/scripts/uninstall.sh | bash -
```

或手动卸载:
```bash
sudo systemctl stop ttsalert
sudo systemctl disable ttsalert
sudo rm /etc/systemd/system/ttsalert.service
sudo rm /usr/local/bin/ttsalert
sudo rm -rf /etc/ttsalert
sudo rm -rf /var/lib/ttsalert
sudo userdel ttsalert
```

## 目录结构

| 路径 | 说明 |
|------|------|
| `/usr/local/bin/ttsalert` | 主程序 |
| `/etc/ttsalert/config.yaml` | 配置文件 |
| `/var/lib/ttsalert/audio` | 音频文件存储 |
| `/var/log/journal/` | 系统日志 (通过 journalctl 查看) |

## 防火墙配置

如需从外部访问 Webhook:

```bash
# firewalld (CentOS/RHEL)
sudo firewall-cmd --permanent --add-port=8080/tcp
sudo firewall-cmd --reload

# ufw (Ubuntu/Debian)
sudo ufw allow 8080/tcp
sudo ufw reload

# iptables
sudo iptables -A INPUT -p tcp --dport 8080 -j ACCEPT
```

SIP 端口 (如果使用本地 SIP):
```bash
sudo firewall-cmd --permanent --add-port=5060/udp
sudo firewall-cmd --reload
```

## 常见问题

### 服务无法启动

查看日志:
```bash
sudo journalctl -u ttsalert -n 50 --no-pager
```

检查配置:
```bash
sudo ttsalert --config /etc/ttsalert/config.yaml
```

### EdgeTTS 无法使用

检查安装:
```bash
edge-tts --version
```

重新安装:
```bash
pip3 uninstall edge-tts
pip3 install edge-tts
```

### SIP 呼叫失败

1. 检查 SIP 服务器连接
2. 验证 SIP 账号密码
3. 检查防火墙是否放行 5060 端口
4. 查看日志中的 SIP 错误信息
