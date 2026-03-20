#!/bin/bash
# test-pacman.sh — Install packages via pacman through pkgproxy.
# Usage: test-pacman.sh <proxy-address> <package> [package...]
set -euo pipefail

PROXY_ADDR="$1"; shift
PACKAGES=("$@")

echo "==> Proxy: ${PROXY_ADDR}"
echo "==> Packages: ${PACKAGES[*]}"

# Configure pacman mirror to use pkgproxy.
echo "Server = http://${PROXY_ADDR}/archlinux/\$repo/os/\$arch" > /etc/pacman.d/mirrorlist

pacman -Sy
pacman -S --noconfirm "${PACKAGES[@]}"

echo "==> Done"
