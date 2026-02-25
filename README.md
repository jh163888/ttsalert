# TTS Alert - 语音告警系统

基于 EdgeTTS 和 SIP 的自动语音告警系统，支持 Zabbix 和卓豪 OPM 集成。

## 功能特性

- ✅ 支持 EdgeTTS 高质量中文语音合成
- ✅ SIP 协议自动拨号
- ✅ Webhook 接收 Zabbix/OPM 告警
- ✅ 告警队列和重试机制
- ✅ Docker 容器化部署
- ✅ 多个电话号码轮询

## 快速开始

### 1. 安装依赖

```bash
# 安装 Go 依赖
go mod download

# 安装 EdgeTTS (Python)
pip install edge-tts

# 或安装 espeak (备用)
apt-get install espeak
```

### 2. 配置

```bash
cp configs/config.example.yaml config.yaml
```

编辑 `config.yaml`:

```yaml
server:
  host: "0.0.0.0"
  port: 8080

tts:
  use_edgetts: true
  voice: "zh-CN-XiaoxiaoNeural"
  output_dir: "./audio"

sip:
  server: "sip.example.com"
  port: 5060
  username: "alert_user"
  password: "your_password"
  domain: "example.com"
  local_port: 5060
```

### 3. 运行

```bash
go run ./cmd
```

### 4. Docker 部署

```bash
docker build -t ttsalert .
docker run -d -p 8080:8080 -p 5060:5060/udp \
  -v $(pwd)/config.yaml:/root/config.yaml \
  -v $(pwd)/audio:/root/audio \
  ttsalert
```

## Zabbix 集成

### 创建告警媒介类型

1. 管理 → 告警媒介类型 → 创建
2. 类型：Webhook
3. 脚本:

```javascript
return {
    'url': 'http://ttsalert:8080/webhook/zabbix',
    'query_fields': [
        {'name': 'eventid', 'value': '{EVENT.ID}'},
        {'name': 'title', 'value': '{EVENT.SEVERITY}: {EVENT.NAME}'},
        {'name': 'message', 'value': '{EVENT.OPDATA}'},
        {'name': 'severity', 'value': '{EVENT.SEVERITY}'},
        {'name': 'host', 'value': '{HOST.NAME}'},
        {'name': 'phone_number', 'value': '{ALERT.SENDTO}'}
    ],
    'timeout': '30s'
};
```

### 创建用户告警媒介

1. 管理 → 用户 → 选择用户 → 告警媒介
2. 添加：类型 TTS Alert, 收件人 13800138000

## 卓豪 OPM 集成

### 创建通知配置文件

1. 设置 → 通知配置文件 → 添加
2. URL: `http://ttsalert:8080/webhook/opm`
3. HTTP 方法：POST
4. 内容类型：application/json
5. 自定义正文:

```json
{
    "alert_id": "{AlertId}",
    "subject": "{AlertSubject}",
    "description": "{AlertDescription}",
    "severity": "{Severity}",
    "device": "{DeviceName}",
    "phone_number": "{PhoneNumber}"
}
```

## API 接口

### 通用 Webhook

```bash
curl -X POST http://localhost:8080/webhook/generic \
  -H "Content-Type: application/json" \
  -d '{
    "title": "服务器告警",
    "message": "CPU 使用率超过 90%",
    "severity": "严重",
    "host": "web-server-01",
    "phone_numbers": ["13800138000", "13900139000"]
  }'
```

### 健康检查

```bash
curl http://localhost:8080/health
```

## 配置说明

### TTS 配置

| 参数 | 说明 | 默认值 |
|------|------|--------|
| voice | EdgeTTS 语音 | zh-CN-XiaoxiaoNeural |
| rate | 语速 | +0% |
| volume | 音量 | +0% |
| pitch | 音调 | +0Hz |
| use_edgetts | 使用 EdgeTTS | true |

可用语音:
- zh-CN-XiaoxiaoNeural (女声)
- zh-CN-YunxiNeural (男声)
- zh-CN-XiaoyiNeural (女声)

### SIP 配置

| 参数 | 说明 |
|------|------|
| server | SIP 服务器地址 |
| port | SIP 端口 (默认 5060) |
| username | SIP 用户名 |
| password | SIP 密码 |
| domain | SIP 域 |
| max_call_duration | 最大通话时长 |
| max_retries | 最大重试次数 |

## 项目结构

```
ttsalert/
├── cmd/                    # 主程序入口
├── internal/
│   ├── tts/               # TTS 语音生成
│   ├── sip/               # SIP 拨号
│   ├── handler/           # Webhook 处理
│   └── queue/             # 告警队列
├── configs/               # 配置文件
└── audio/                 # 生成的音频文件
```

## License

MIT
