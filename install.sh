#!/bin/sh
# Downloads the fastygo 0.1.0 binary for Linux amd64 or macOS arm64.
# The GitHub repository is private, so this script works only with access to that release.
# Windows and everyone else: npx fastygo, which carries the binary in the npm package.

set -eu
version=0.1.0
repo=fastygo/cli

os=$(uname -s)
arch=$(uname -m)
case "$os-$arch" in
  Linux-x86_64) file=fastygo_${version}_linux_amd64.tar.gz; bin=fastygo ;;
  Darwin-arm64) file=fastygo_${version}_darwin_arm64.tar.gz; bin=fastygo ;;
  *)
    echo "fastygo ${version} shell install supports Linux amd64 and macOS arm64 ($os-$arch)." >&2
    echo "go install github.com/fastygo/cli/cmd/fastygo@v${version}" >&2
    exit 1
    ;;
esac

dest=${HOME}/.cache/fastygo/${version}
mkdir -p "$dest"
base=https://github.com/${repo}/releases/download/v${version}
curl -fsSL "$base/$file" -o "$dest/$file"
curl -fsSL "$base/SHA256SUMS" -o "$dest/SHA256SUMS"
want=$(awk -v name="$file" '$2 == name { print $1 }' "$dest/SHA256SUMS")
if command -v sha256sum >/dev/null 2>&1; then
  got=$(sha256sum "$dest/$file" | awk '{ print $1 }')
else
  got=$(shasum -a 256 "$dest/$file" | awk '{ print $1 }')
fi
if [ "$want" != "$got" ] || [ -z "$want" ]; then
  echo "checksum mismatch for $file" >&2
  exit 1
fi
tar -xzf "$dest/$file" -C "$dest"
chmod 755 "$dest/$bin"
echo "$dest/$bin"
echo "Add that directory to PATH, or run the binary by its path."
