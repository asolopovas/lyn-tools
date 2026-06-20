#!/usr/bin/env bash
set -u

tool=${1:-}

have() {
  command -v "$1" >/dev/null 2>&1
}

missing() {
  printf '%s is not installed on this system.\n' "$1"
}

case "$tool" in
monitor)
  if have htop; then
    htop
  elif have top; then
    top
  else
    missing htop
  fi
  ;;
logs)
  if have journalctl; then
    journalctl -xe
  else
    missing journalctl
  fi
  ;;
services)
  if have systemctl; then
    systemctl status
  else
    missing systemctl
  fi
  ;;
network)
  if have nmtui; then
    nmtui
  else
    missing nmtui
  fi
  ;;
firewall)
  if have ufw; then
    sudo ufw status verbose
  else
    missing ufw
  fi
  ;;
disk)
  if have df; then
    df -h
  else
    missing df
  fi
  ;;
*)
  printf 'Unknown system tool: %s\n' "$tool"
  ;;
esac

exec "${SHELL:-bash}" -i
