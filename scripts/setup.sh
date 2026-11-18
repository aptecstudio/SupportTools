#!/usr/bin/env bash
# SPDX-License-Identifier: GPL-3.0-or-later
#
# Installa le dipendenze di sviluppo di Support Tools su Linux e macOS:
#   - toolchain Go 1.20.14 in /usr/local/go (richiede sudo);
#   - strumenti di sviluppo: rsrc (manifest/icona) e goimports;
#   - moduli Go elencati in go.mod.
#
# Lo script è idempotente: può essere rieseguito senza effetti collaterali.
# Uso: scripts/setup.sh [--force]
#   --force  sostituisce un'eventuale versione di Go diversa già presente
#            in /usr/local/go.

set -euo pipefail

# Versioni fissate: Go 1.20.x è l'ultima release che supporta Windows 7.
readonly GO_VERSION="1.20.14"
readonly GO_ROOT="/usr/local/go"
readonly RSRC_VERSION="v0.10.2"
# Ultima versione di x/tools compatibile con Go 1.20.
readonly GOIMPORTS_VERSION="v0.24.0"

# Marcatore usato per riconoscere la riga aggiunta al profilo della shell.
readonly PROFILE_MARKER="# Support Tools: toolchain Go"

force=0
tmp=""

info() { printf '\033[1;34m==>\033[0m %s\n' "$*"; }
warn() { printf '\033[1;33m[!]\033[0m %s\n' "$*" >&2; }
fail() {
	printf '\033[1;31m[x]\033[0m %s\n' "$*" >&2
	exit 1
}

parse_args() {
	for arg in "$@"; do
		case "$arg" in
		--force) force=1 ;;
		-h | --help)
			sed -n '3,13p' "$0" | sed 's/^# \{0,1\}//'
			exit 0
			;;
		*) fail "Argomento sconosciuto: $arg" ;;
		esac
	done
}

# go_archive stampa "nomefile sha256" dell'archivio Go adatto al sistema.
go_archive() {
	local os arch
	case "$(uname -s)" in
	Linux) os="linux" ;;
	Darwin) os="darwin" ;;
	*) fail "Sistema operativo non supportato: $(uname -s). Su Windows installa Go ${GO_VERSION} da https://go.dev/dl/." ;;
	esac
	case "$(uname -m)" in
	x86_64 | amd64) arch="amd64" ;;
	aarch64 | arm64) arch="arm64" ;;
	i386 | i686) arch="386" ;;
	*) fail "Architettura non supportata: $(uname -m)" ;;
	esac

	local file="go${GO_VERSION}.${os}-${arch}.tar.gz"
	local sum
	# Checksum ufficiali pubblicati su https://go.dev/dl/
	case "$file" in
	go1.20.14.linux-amd64.tar.gz) sum="ff445e48af27f93f66bd949ae060d97991c83e11289009d311f25426258f9c44" ;;
	go1.20.14.linux-arm64.tar.gz) sum="2096507509a98782850d1f0669786c09727053e9fe3c92b03c0d96f48700282b" ;;
	go1.20.14.linux-386.tar.gz) sum="9c0acad376b41292c6e9e5534e26d9432f92a214d6c40a7e4c024b0235cc30e8" ;;
	go1.20.14.darwin-amd64.tar.gz) sum="754363489e2244e72cb49b4ec6ddfd6a2c60b0700f8c4876e11befb1913b11c5" ;;
	go1.20.14.darwin-arm64.tar.gz) sum="6da3f76164b215053daf730a9b8f1d673dbbaa4c61031374a6744b75cb728641" ;;
	*) fail "Nessun archivio Go disponibile per ${os}/${arch}" ;;
	esac
	printf '%s %s\n' "$file" "$sum"
}

sha256() {
	if command -v sha256sum >/dev/null 2>&1; then
		sha256sum "$1" | cut -d' ' -f1
	else
		shasum -a 256 "$1" | cut -d' ' -f1
	fi
}

check_requirements() {
	local missing=()
	for cmd in curl tar git sudo; do
		command -v "$cmd" >/dev/null 2>&1 || missing+=("$cmd")
	done
	if ! command -v sha256sum >/dev/null 2>&1 && ! command -v shasum >/dev/null 2>&1; then
		missing+=("sha256sum")
	fi
	if ((${#missing[@]} > 0)); then
		fail "Comandi mancanti: ${missing[*]}. Installali e rilancia lo script."
	fi
}

install_go() {
	local current=""
	if [[ -x "${GO_ROOT}/bin/go" ]]; then
		current="$("${GO_ROOT}/bin/go" env GOVERSION 2>/dev/null || true)"
	fi

	if [[ "$current" == "go${GO_VERSION}" ]]; then
		info "Go ${GO_VERSION} già installato in ${GO_ROOT}"
		return
	fi
	if [[ -n "$current" && "$force" -ne 1 ]]; then
		fail "In ${GO_ROOT} è presente ${current}. Rilancia con --force per sostituirlo con Go ${GO_VERSION}."
	fi

	local file sum
	read -r file sum < <(go_archive)

	tmp="$(mktemp -d)"
	trap 'rm -rf "${tmp:-}"' EXIT

	info "Download di ${file}"
	curl -fL --progress-bar -o "${tmp}/${file}" "https://go.dev/dl/${file}"

	info "Verifica del checksum"
	if [[ "$(sha256 "${tmp}/${file}")" != "$sum" ]]; then
		fail "Checksum non valido per ${file}: download corrotto o manomesso."
	fi

	info "Installazione in ${GO_ROOT} (richiede sudo)"
	sudo rm -rf "$GO_ROOT"
	sudo tar -C "$(dirname "$GO_ROOT")" -xzf "${tmp}/${file}"
}

# configure_path rende disponibili go e gli strumenti installati, sia per
# questa esecuzione sia per le shell future (tramite ~/.profile).
configure_path() {
	export PATH="${GO_ROOT}/bin:${PATH}"
	local gobin
	gobin="$(go env GOPATH)/bin"
	export PATH="${gobin}:${PATH}"

	local profile="${HOME}/.profile"
	if [[ -f "$profile" ]] && grep -qF "$PROFILE_MARKER" "$profile"; then
		return
	fi
	info "Aggiunta di ${GO_ROOT}/bin e ${gobin} al PATH in ${profile}"
	{
		printf '\n%s\n' "$PROFILE_MARKER"
		# shellcheck disable=SC2016 # espansione voluta al login
		printf 'export PATH="%s/bin:$HOME/go/bin:$PATH"\n' "$GO_ROOT"
	} >>"$profile"
}

install_tools() {
	# I moduli degli strumenti non devono richiedere una versione di Go più recente.
	export GOTOOLCHAIN=local

	info "Installazione di rsrc ${RSRC_VERSION}"
	go install "github.com/akavel/rsrc@${RSRC_VERSION}"

	info "Installazione di goimports ${GOIMPORTS_VERSION}"
	go install "golang.org/x/tools/cmd/goimports@${GOIMPORTS_VERSION}"
}

download_modules() {
	local root
	root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

	info "Download dei moduli Go del progetto"
	(cd "$root" && go mod download && go mod verify)
}

main() {
	parse_args "$@"
	check_requirements
	install_go
	configure_path
	install_tools
	download_modules

	info "Completato: $(go version)"
	echo "Apri un nuovo terminale (o esegui 'source ~/.profile') per avere go nel PATH."
}

main "$@"
