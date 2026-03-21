#!/bin/bash
# test-apt.sh — Install packages via apt through pkgproxy.
# Usage: test-apt.sh <package> [package...]
#
# Expects a sources.list to be mounted into /etc/apt/sources.list by the caller.
set -euo pipefail

PACKAGES=("$@")

echo "==> Packages: ${PACKAGES[*]}"

# Remove any sources.list.d entries that might interfere.
rm -f /etc/apt/sources.list.d/*.sources /etc/apt/sources.list.d/*.list

echo "==> sources.list:"
echo "--- /etc/apt/sources.list ---"
cat /etc/apt/sources.list

export DEBIAN_FRONTEND=noninteractive

apt-get update
apt-get install -y "${PACKAGES[@]}"

echo "==> Done"
