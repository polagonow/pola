#!/bin/sh
# Pola installer — downloads the latest release binary for this platform.
#
#   curl -fsSL https://raw.githubusercontent.com/polagonow/pola/main/install.sh | sh
#
# Options via environment variables:
#   POLA_VERSION  release tag to install (default: latest, e.g. v0.6.0)
#   POLA_INSTALL  install directory (default: /usr/local/bin, falls back to
#                 ~/.local/bin when /usr/local/bin is not writable)
#
# Installing via curl+tar deliberately avoids the com.apple.quarantine
# attribute that browsers set on downloads, so macOS Gatekeeper does not
# flag the binary even when the release is unsigned.
set -eu

REPO="polagonow/pola"
BIN="pola"

os=$(uname -s)
case "$os" in
Darwin) os="darwin" ;;
Linux) os="linux" ;;
*)
	echo "error: unsupported OS: $os (use the .zip release asset on Windows)" >&2
	exit 1
	;;
esac

arch=$(uname -m)
case "$arch" in
x86_64 | amd64) arch="amd64" ;;
arm64 | aarch64) arch="arm64" ;;
*)
	echo "error: unsupported architecture: $arch" >&2
	exit 1
	;;
esac

version="${POLA_VERSION:-}"
if [ -z "$version" ]; then
	version=$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" |
		grep '"tag_name"' | head -n1 | cut -d'"' -f4)
fi
if [ -z "$version" ]; then
	echo "error: could not determine the latest release of $REPO" >&2
	exit 1
fi

asset="$BIN-$version-$os-$arch.tar.gz"
base_url="https://github.com/$REPO/releases/download/$version"

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

echo "→ downloading $asset ($version)"
curl -fsSL -o "$tmp/$asset" "$base_url/$asset"
curl -fsSL -o "$tmp/checksums.txt" "$base_url/checksums.txt"

echo "→ verifying checksum"
expected=$(grep " $asset\$" "$tmp/checksums.txt" | cut -d' ' -f1)
if [ -z "$expected" ]; then
	echo "error: $asset not found in checksums.txt" >&2
	exit 1
fi
if command -v sha256sum >/dev/null 2>&1; then
	actual=$(sha256sum "$tmp/$asset" | cut -d' ' -f1)
else
	actual=$(shasum -a 256 "$tmp/$asset" | cut -d' ' -f1)
fi
if [ "$expected" != "$actual" ]; then
	echo "error: checksum mismatch for $asset" >&2
	echo "  expected: $expected" >&2
	echo "  actual:   $actual" >&2
	exit 1
fi

tar -xzf "$tmp/$asset" -C "$tmp"

install_dir="${POLA_INSTALL:-/usr/local/bin}"
if [ ! -w "$install_dir" ] && [ -z "${POLA_INSTALL:-}" ]; then
	install_dir="$HOME/.local/bin"
fi
mkdir -p "$install_dir"

echo "→ installing to $install_dir/$BIN"
mv "$tmp/$BIN" "$install_dir/$BIN"
chmod 0755 "$install_dir/$BIN"

case ":$PATH:" in
*":$install_dir:"*) ;;
*) echo "note: $install_dir is not on your PATH" ;;
esac

echo "✓ installed $("$install_dir/$BIN" --version)"
