#!/usr/bin/env bash
# install.sh — Build gog from source, ensuring Go is available.
# If Go is not installed or too old, downloads and uses a local Go toolchain.
# Usage: ./scripts/install.sh [from repo root, or run from scripts/]

set -e

# Required Go version (must match go.mod)
MIN_GO_MAJOR=1
MIN_GO_MINOR=24
GO_INSTALL_VERSION="1.24.0"

# Repo root (parent of scripts/)
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]:-$0}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$ROOT_DIR"

# --- Check if Go is already installed and new enough ---
check_go() {
	if ! command -v go &>/dev/null; then
		return 1
	fi
	local v
	v=$(go version 2>/dev/null | sed -n 's/.*go\([0-9]*\)\.\([0-9]*\).*/\1 \2/p')
	local major minor
	read -r major minor <<< "$v"
	[[ -n "$major" && -n "$minor" ]] || return 1
	if [[ "$major" -gt "$MIN_GO_MAJOR" ]]; then
		return 0
	fi
	if [[ "$major" -eq "$MIN_GO_MAJOR" && "$minor" -ge "$MIN_GO_MINOR" ]]; then
		return 0
	fi
	return 1
}

# --- Download and unpack Go into a local directory ---
install_go_local() {
	local go_os go_arch
	case "$(uname -s)" in
		Darwin) go_os=darwin ;;
		Linux)  go_os=linux ;;
		*)
			echo "install.sh: unsupported OS $(uname -s). Please install Go manually (Go ${MIN_GO_MAJOR}.${MIN_GO_MINOR}+)." >&2
			exit 1
			;;
	esac
	case "$(uname -m)" in
		x86_64|amd64) go_arch=amd64 ;;
		aarch64|arm64) go_arch=arm64 ;;
		*)
			echo "install.sh: unsupported arch $(uname -m). Please install Go manually (Go ${MIN_GO_MAJOR}.${MIN_GO_MINOR}+)." >&2
			exit 1
			;;
	esac

	local base="go${GO_INSTALL_VERSION}.${go_os}-${go_arch}"
	local tarball="${base}.tar.gz"
	local url="https://go.dev/dl/${tarball}"
	local install_dir="${HOME}/.local/gogcli-go"
	local goroot="${install_dir}/go"

	if [[ -x "${goroot}/bin/go" ]]; then
		export GOROOT="$goroot"
		export PATH="${goroot}/bin:$PATH"
		if check_go; then
			echo "Using existing Go at ${goroot}" >&2
			return 0
		fi
	fi

	echo "Go not found or version < ${MIN_GO_MAJOR}.${MIN_GO_MINOR}; downloading ${GO_INSTALL_VERSION} to ${install_dir}..." >&2
	mkdir -p "$install_dir"
	local tmpdir
	tmpdir="$(mktemp -d)"
	trap "rm -rf '$tmpdir'" EXIT
	(
		cd "$tmpdir"
		if command -v curl &>/dev/null; then
			curl -fsSL -o "$tarball" "$url"
		elif command -v wget &>/dev/null; then
			wget -q -O "$tarball" "$url"
		else
			echo "install.sh: need curl or wget to download Go." >&2
			exit 1
		fi
		rm -rf "${install_dir}/go"
		tar -xzf "$tarball" -C "$install_dir"
	)
	export GOROOT="$goroot"
	export PATH="${goroot}/bin:$PATH"
	if ! check_go; then
		echo "install.sh: Go download or version check failed." >&2
		exit 1
	fi
	echo "Go ${GO_INSTALL_VERSION} installed at ${goroot}" >&2
}

# --- Main ---
if ! check_go; then
	install_go_local
fi

echo "Building gog..." >&2
make build
echo "Done. Binary: $ROOT_DIR/bin/gog" >&2
