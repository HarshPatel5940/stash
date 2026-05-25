#!/usr/bin/env bash
set -euo pipefail

REPO="harshpatel5940/stash"
BIN="stash"
TAP="harshpatel5940/tap"

log() {
  printf "==> %s\n" "$1"
}

warn() {
  printf "⚠️  %s\n" "$1" >&2
}

die() {
  printf "✗ %s\n" "$1" >&2
  exit 1
}

if [[ "$(uname -s)" != "Darwin" ]]; then
  die "This installer supports macOS only."
fi

if command -v brew >/dev/null 2>&1; then
  log "Homebrew detected. Installing via brew..."
  if brew install "${TAP}/${BIN}"; then
    log "Installed ${BIN} via Homebrew."
    log "Next: run \`${BIN} init\`"
    exit 0
  fi
  warn "brew install failed. Falling back to direct binary install."
fi

arch="$(uname -m)"
case "${arch}" in
  arm64) arch="arm64" ;;
  x86_64) arch="amd64" ;;
  *) die "Unsupported architecture: ${arch}" ;;
esac

install_dir="${STASH_INSTALL_DIR:-}"
if [[ -z "${install_dir}" ]]; then
  if [[ "${arch}" == "arm64" && -d "/opt/homebrew/bin" ]]; then
    install_dir="/opt/homebrew/bin"
  else
    install_dir="/usr/local/bin"
  fi
fi

for cmd in curl tar shasum; do
  if ! command -v "${cmd}" >/dev/null 2>&1; then
    die "Missing required command: ${cmd}"
  fi
done

log "Fetching latest release metadata..."
tag="$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | awk -F '"' '/tag_name/{print $4; exit}')"
if [[ -z "${tag}" ]]; then
  die "Unable to determine the latest release tag."
fi

version="${tag#v}"
asset="${BIN}_${version}_darwin_${arch}.tar.gz"
base_url="https://github.com/${REPO}/releases/download/${tag}"

tmpdir="$(mktemp -d)"
cleanup() { rm -rf "${tmpdir}"; }
trap cleanup EXIT

log "Downloading ${asset}..."
curl -fsSL -o "${tmpdir}/${asset}" "${base_url}/${asset}"
curl -fsSL -o "${tmpdir}/checksums.txt" "${base_url}/checksums.txt"

log "Verifying checksums..."
if ! grep " ${asset}$" "${tmpdir}/checksums.txt" | shasum -a 256 -c -; then
  die "Checksum verification failed."
fi

log "Extracting archive..."
tar -xzf "${tmpdir}/${asset}" -C "${tmpdir}"

bin_path="${tmpdir}/${BIN}"
if [[ ! -f "${bin_path}" ]]; then
  bin_path="$(find "${tmpdir}" -maxdepth 2 -type f -name "${BIN}" | head -n 1 || true)"
fi
if [[ -z "${bin_path}" || ! -f "${bin_path}" ]]; then
  die "stash binary not found in archive."
fi

sudo_cmd=""
if [[ ! -w "${install_dir}" ]]; then
  sudo_cmd="sudo"
fi

log "Installing to ${install_dir}/${BIN}..."
${sudo_cmd} install -m 0755 "${bin_path}" "${install_dir}/${BIN}"

log "Installed ${BIN}."
log "Next: run \`${BIN} init\`"
