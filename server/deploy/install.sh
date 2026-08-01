#!/usr/bin/env bash
set -euo pipefail

# Simple install script for SimplePresent server (requires sudo)
# Installs binary to /usr/local/bin, config to /etc/simplepresent, creates data dir and systemd unit.

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

BIN="${SCRIPT_DIR}/simplepresent-server"
INSTALL_BIN=/usr/local/bin/simplepresent-server

DATA_DIR=/var/lib/simplepresent

ETC_DIR=/etc/simplepresent
ETC_CONFIG=${ETC_DIR}/config.json

SYSTEMD_UNIT="${SCRIPT_DIR}/simplepresent-server.service"
SYSTEMD_TARGET=/etc/systemd/system/simplepresent-server.service

if [ "$(id -u)" -ne 0 ]; then
  echo "This script must be run as root (or with sudo)" >&2
  exit 1
fi

if [ ! -f "$BIN" ]; then
  echo "Binary $BIN not found." >&2
  exit 1
fi

if [ ! -f "${SCRIPT_DIR}/config.json.example" ]; then
  echo "Example config ${SCRIPT_DIR}/config.json.example not found." >&2
  exit 1
fi

if [ ! -f "$SYSTEMD_UNIT" ]; then
  echo "Systemd unit $SYSTEMD_UNIT not found." >&2
  exit 1
fi

echo "Installing binary to ${INSTALL_BIN}"
if [ -f "$INSTALL_BIN" ]; then
  echo "Found existing binary at ${INSTALL_BIN} -> backing up"
  mv "$INSTALL_BIN" "${INSTALL_BIN}.bak.$(date +%s)" || true
fi
install -m 0755 "$BIN" "$INSTALL_BIN"

echo "Creating user/group 'simplepresent' if missing"
if ! id -u simplepresent >/dev/null 2>&1; then
  useradd --system --no-create-home --shell /usr/sbin/nologin simplepresent || true
fi

echo "Creating data dir ${DATA_DIR}"
mkdir -p "$DATA_DIR"
chown simplepresent:simplepresent "$DATA_DIR"
chmod 0755 "$DATA_DIR"

echo "Installing config to ${ETC_CONFIG} (only if missing)"
mkdir -p "$ETC_DIR"
if [ ! -f "$ETC_CONFIG" ]; then
  install -m 0644 "${SCRIPT_DIR}/config.json.example" "$ETC_CONFIG"
  JWT_SECRET="$(openssl rand -hex 32)"
  sed -i "s/change-me-in-production/${JWT_SECRET}/g" "$ETC_CONFIG"
  echo "Wrote example config to $ETC_CONFIG - edit as needed"
else
  echo "$ETC_CONFIG already exists - leaving it in place -> check and edit as needed"
fi

echo "Would you like to edit ${ETC_CONFIG} now before the service starts? [y/N]"
read -r edit_config_answer || true
case "${edit_config_answer:-}" in
  y|Y|yes|YES)
    if ! command -v nano >/dev/null 2>&1; then
      echo "nano is not installed, cannot edit ${ETC_CONFIG}" >&2
      exit 1
    fi
    nano "$ETC_CONFIG"
    ;;
esac

echo "Installing systemd unit to ${SYSTEMD_TARGET}"
if [ -f "$SYSTEMD_TARGET" ] || systemctl list-unit-files --type=service | grep -q "^simplepresent-server.service" 2>/dev/null; then
  echo "Detected existing systemd unit for simplepresent-server. Stopping and disabling service."
  systemctl stop simplepresent-server.service 2>/dev/null || true
  systemctl disable simplepresent-server.service 2>/dev/null || true
  if [ -f "$SYSTEMD_TARGET" ]; then
    echo "Backing up existing unit to ${SYSTEMD_TARGET}.bak.$(date +%s)"
    mv "$SYSTEMD_TARGET" "${SYSTEMD_TARGET}.bak.$(date +%s)" || true
  fi
fi
install -m 0644 "$SYSTEMD_UNIT" "$SYSTEMD_TARGET"

echo "Reloading systemd and enabling service"
systemctl daemon-reload
systemctl enable --now simplepresent-server.service

echo "If this was an update, previous binary/unit were backed up with .bak.<ts> suffix. Config at ${ETC_CONFIG} and data at ${DATA_DIR} are preserved."

echo "Install complete. Check 'journalctl -u simplepresent-server -f' for logs."
