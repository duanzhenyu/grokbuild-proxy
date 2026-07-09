#!/usr/bin/env sh
set -eu

REPO="${GROKBUILD_REPO:-GreyGunG/grokbuild-proxy}"
VERSION="${GROKBUILD_VERSION:-latest}"
INSTALL_DIR="${INSTALL_DIR:-${HOME}/.local/bin}"
CONFIG_DIR="${CONFIG_DIR:-${HOME}/.config/grokbuild-proxy}"

fail() {
  printf 'grokbuild-proxy installer: %s\n' "$*" >&2
  exit 1
}

command -v curl >/dev/null 2>&1 || fail "curl is required"
command -v tar >/dev/null 2>&1 || fail "tar is required"

case "$(uname -s)" in
  Linux) os="Linux" ;;
  Darwin) os="Darwin" ;;
  *) fail "unsupported operating system: $(uname -s)" ;;
esac

case "$(uname -m)" in
  x86_64 | amd64) arch="x86_64" ;;
  arm64 | aarch64) arch="arm64" ;;
  *) fail "unsupported architecture: $(uname -m)" ;;
esac

archive="grokbuild-proxy_${os}_${arch}.tar.gz"
if [ "$VERSION" = "latest" ]; then
  base_url="https://github.com/${REPO}/releases/latest/download"
else
  base_url="https://github.com/${REPO}/releases/download/${VERSION}"
fi

tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT INT TERM

printf 'Downloading %s...\n' "$archive"
curl --fail --location --silent --show-error \
  "${base_url}/${archive}" \
  -o "${tmp_dir}/${archive}"
curl --fail --location --silent --show-error \
  "${base_url}/checksums.txt" \
  -o "${tmp_dir}/checksums.txt"

expected="$(
  awk -v file="$archive" '$2 == file { print $1; exit }' \
    "${tmp_dir}/checksums.txt"
)"
[ -n "$expected" ] || fail "checksum entry not found for ${archive}"

if command -v sha256sum >/dev/null 2>&1; then
  actual="$(sha256sum "${tmp_dir}/${archive}" | awk '{print $1}')"
elif command -v shasum >/dev/null 2>&1; then
  actual="$(shasum -a 256 "${tmp_dir}/${archive}" | awk '{print $1}')"
else
  fail "sha256sum or shasum is required"
fi

[ "$actual" = "$expected" ] || fail "checksum verification failed"

tar -xzf "${tmp_dir}/${archive}" -C "$tmp_dir"
mkdir -p "$INSTALL_DIR" "$CONFIG_DIR"
install -m 0755 "${tmp_dir}/grokbuild-proxy" \
  "${INSTALL_DIR}/grokbuild-proxy"

if [ ! -f "${CONFIG_DIR}/config.yaml" ]; then
  install -m 0600 "${tmp_dir}/config.example.yaml" \
    "${CONFIG_DIR}/config.yaml"
fi

printf '\nInstalled: %s\n' "${INSTALL_DIR}/grokbuild-proxy"
printf 'Config:    %s\n' "${CONFIG_DIR}/config.yaml"
printf '\nRun:\n  %s -config %s\n' \
  "${INSTALL_DIR}/grokbuild-proxy" \
  "${CONFIG_DIR}/config.yaml"

case ":${PATH}:" in
  *":${INSTALL_DIR}:"*) ;;
  *) printf '\nAdd %s to PATH to run grokbuild-proxy directly.\n' "$INSTALL_DIR" ;;
esac
