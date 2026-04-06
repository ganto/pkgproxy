#!/bin/bash
# test-gentoo.sh — Fetch a package via emerge --fetchonly through pkgproxy.
# Usage: test-gentoo.sh <proxy-address>
set -euo pipefail

PROXY_ADDR="$1"

echo "==> Proxy: ${PROXY_ADDR}"

# Bootstrap: download portage snapshot directly (bypassing the proxy).
echo "==> Downloading portage snapshot..."
wget -q https://distfiles.gentoo.org/snapshots/portage-latest.tar.xz -O /tmp/portage-latest.tar.xz
echo "==> Unpacking portage snapshot..."
mkdir -p /var/db/repos/gentoo
tar xf /tmp/portage-latest.tar.xz -C /var/db/repos/gentoo --strip-components=1
rm /tmp/portage-latest.tar.xz

# Configure GENTOO_MIRRORS to point at pkgproxy.
echo "GENTOO_MIRRORS=\"http://${PROXY_ADDR}/gentoo\"" >> /etc/portage/make.conf

echo "==> make.conf:"
echo "--- /etc/portage/make.conf ---"
cat /etc/portage/make.conf

# Fetch distfiles for app-text/tree through the proxy.
echo "==> Running emerge --fetchonly app-text/tree..."
emerge --fetchonly app-text/tree

# Exercise the negative cache path: fetch layout.conf through the proxy.
echo "==> Fetching layout.conf through proxy (should not be cached)..."
wget -q "http://${PROXY_ADDR}/gentoo/distfiles/layout.conf" -O /dev/null

echo "==> Done"
