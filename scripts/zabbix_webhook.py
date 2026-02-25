#!/usr/bin/env python3
"""
Zabbix Webhook Script for TTS Alert
用于 Zabbix 告警媒介类型的 Python 脚本
"""

import sys
import json
import urllib.request
import urllib.error

def main():
    if len(sys.argv) < 7:
        print("Usage: zabbix_webhook.py <url> <eventid> <title> <message> <severity> <host> <phone>")
        sys.exit(1)

    url = sys.argv[1]
    eventid = sys.argv[2]
    title = sys.argv[3]
    message = sys.argv[4]
    severity = sys.argv[5]
    host = sys.argv[6]
    phone = sys.argv[7] if len(sys.argv) > 7 else ""

    payload = {
        "eventid": eventid,
        "title": title,
        "message": message,
        "severity": severity,
        "host": host,
        "phone_number": phone
    }

    data = json.dumps(payload).encode('utf-8')
    
    req = urllib.request.Request(
        url,
        data=data,
        headers={'Content-Type': 'application/json'},
        method='POST'
    )

    try:
        with urllib.request.urlopen(req, timeout=30) as response:
            result = json.loads(response.read().decode('utf-8'))
            print(f"Success: {result.get('status', 'queued')}")
            sys.exit(0)
    except urllib.error.HTTPError as e:
        print(f"HTTP Error: {e.code} - {e.reason}")
        sys.exit(1)
    except urllib.error.URLError as e:
        print(f"URL Error: {e.reason}")
        sys.exit(1)
    except Exception as e:
        print(f"Error: {e}")
        sys.exit(1)

if __name__ == "__main__":
    main()
