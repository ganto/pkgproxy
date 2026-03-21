#!/bin/bash
# test-dnf.sh — Install packages via dnf through pkgproxy.
# Usage: test-dnf.sh <proxy-address> <package> [package...]
#
# Expects a .repo file to be mounted into /etc/yum.repos.d/ by the caller.
# The repo config controls GPG settings (gpgcheck=1 for standard repos,
# gpgcheck=0 for COPR).
set -euo pipefail

PROXY_ADDR="$1"; shift
PACKAGES=("$@")

echo "==> Proxy: ${PROXY_ADDR}"
echo "==> Packages: ${PACKAGES[*]}"

# Remove all default repo files so only the mounted pkgproxy repo is used.
find /etc/yum.repos.d/ -name '*.repo' ! -name 'pkgproxy-*' -delete

echo "==> Repo files:"
for f in /etc/yum.repos.d/pkgproxy-*.repo; do
  echo "--- ${f} ---"
  cat "${f}"
done

dnf makecache
dnf install -y "${PACKAGES[@]}"

echo "==> Done"
