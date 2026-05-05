#!/usr/bin/env sh
set -eu

repo="DrakeAFK/naudia"
bin="naudia"

os="$(uname -s)"
arch="$(uname -m)"

case "$os" in
  Darwin) goos="Darwin" ;;
  Linux) goos="Linux" ;;
  *) echo "Unsupported OS: $os" >&2; exit 1 ;;
esac

case "$arch" in
  x86_64|amd64) goarch="x86_64" ;;
  arm64|aarch64) goarch="arm64" ;;
  *) echo "Unsupported architecture: $arch" >&2; exit 1 ;;
esac

install_dir="${NAUDIA_INSTALL_DIR:-$HOME/.local/bin}"
mkdir -p "$install_dir"

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

asset="${bin}_${goos}_${goarch}.tar.gz"
url="https://github.com/${repo}/releases/latest/download/${asset}"

echo "Downloading $url"
curl -fsSL "$url" -o "$tmp/$asset"
tar -xzf "$tmp/$asset" -C "$tmp"
install "$tmp/$bin" "$install_dir/$bin"

echo "Installed $bin to $install_dir/$bin"
echo "Next: ollama pull llama3.1:8b && ollama pull llama3.2:3b && ollama pull nomic-embed-text && naudia init"
