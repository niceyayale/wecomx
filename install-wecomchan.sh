#!/usr/bin/env bash
set -euo pipefail

APP_DIR=/opt/wecomchan
BIN_SRC=${1:-./wecomchan}
ENV_FILE=/etc/wecomchan.env
SERVICE_FILE=/etc/systemd/system/wecomchan.service

if [[ $EUID -ne 0 ]]; then
  echo "Run with sudo: sudo $0 /path/to/wecomchan"
  exit 1
fi

if [[ ! -f "$BIN_SRC" ]]; then
  echo "Binary not found: $BIN_SRC"
  exit 1
fi

install -d -m 0755 "$APP_DIR"

if ! id -u wecomchan >/dev/null 2>&1; then
  useradd --system --home "$APP_DIR" --shell /usr/sbin/nologin wecomchan
fi

install -m 0755 "$BIN_SRC" "$APP_DIR/wecomchan"
chown -R root:root "$APP_DIR"

if [[ ! -f "$ENV_FILE" ]]; then
  install -m 0600 /dev/null "$ENV_FILE"
  echo "Created $ENV_FILE. Configure it before starting the service."
else
  chmod 0600 "$ENV_FILE"
fi

install -m 0644 "$(dirname "$0")/wecomchan.service" "$SERVICE_FILE"

systemctl daemon-reload
systemctl enable wecomchan

if grep -q '^SENDKEY=$\|^SENDKEY=change-me' "$ENV_FILE" 2>/dev/null; then
  echo "Configure $ENV_FILE, then run: sudo systemctl restart wecomchan"
else
  systemctl restart wecomchan
fi

echo
systemctl --no-pager --full status wecomchan || true
echo
echo "Health: curl http://127.0.0.1:8080/health"
