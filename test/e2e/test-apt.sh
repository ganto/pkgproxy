#!/bin/bash
# test-apt.sh — Install packages via apt through pkgproxy.
# Usage: test-apt.sh <proxy-address> <release-codename> <package> [package...]
set -euo pipefail

PROXY_ADDR="$1"; shift
RELEASE="$1"; shift
PACKAGES=("$@")

echo "==> Proxy: ${PROXY_ADDR}"
echo "==> Release: ${RELEASE}"
echo "==> Packages: ${PACKAGES[*]}"

# Write sources.list pointing at pkgproxy for both debian and debian-security.
cat > /etc/apt/sources.list <<EOF
deb http://${PROXY_ADDR}/debian          ${RELEASE}           main contrib non-free non-free-firmware
deb http://${PROXY_ADDR}/debian          ${RELEASE}-updates   main contrib non-free non-free-firmware
deb http://${PROXY_ADDR}/debian-security ${RELEASE}-security  main contrib non-free non-free-firmware
EOF

# Remove any sources.list.d entries that might interfere.
rm -f /etc/apt/sources.list.d/*.sources /etc/apt/sources.list.d/*.list

export DEBIAN_FRONTEND=noninteractive

apt-get update
apt-get install -y "${PACKAGES[@]}"

echo "==> Done"
