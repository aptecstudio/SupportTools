#!/usr/bin/env bash
# SPDX-License-Identifier: GPL-3.0-or-later
#
# Installa le dipendenze di sviluppo di Support Tools su Linux:
#   - toolchain Go 1.20.14 in /usr/local/go (richiede sudo);
#   - strumenti di sviluppo: rsrc (manifest/icona) e goimports;
#   - moduli Go elencati in go.mod;
#   - Wine, per provare l'eseguibile senza Windows (go run ./build -run).
#
# Lo script è idempotente: può essere rieseguito senza effetti collaterali.
# Uso: scripts/setup-linux.sh [--force]
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
			# Stampa il commento iniziale dello script (dalla terza riga).
			awk 'NR > 2 && /^#/ { sub(/^# ?/, ""); print; next } NR > 2 { exit }' "$0"
			exit 0
			;;
		*) fail "Argomento sconosciuto: $arg" ;;
		esac
	done
}

# go_archive stampa "nomefile sha256" dell'archivio Go adatto al sistema.
go_archive() {
	local os="linux" arch
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
	*) fail "Nessun archivio Go disponibile per ${os}/${arch}" ;;
	esac
	printf '%s %s\n' "$file" "$sum"
}

sha256() {
	sha256sum "$1" | cut -d' ' -f1
}

check_requirements() {
	if [[ "$(uname -s)" != "Linux" ]]; then
		fail "Questo script è per Linux: su macOS usa scripts/setup-macos.sh, su Windows scripts/setup-windows.ps1."
	fi

	local missing=()
	for cmd in curl tar git sudo sha256sum; do
		command -v "$cmd" >/dev/null 2>&1 || missing+=("$cmd")
	done
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

# install_wine installa Wine con il gestore di pacchetti della distribuzione.
install_wine() {
	if command -v wine >/dev/null 2>&1; then
		info "Wine già installato: $(wine --version 2>/dev/null || echo 'versione sconosciuta')"
		return
	fi
	case "$(uname -m)" in
	x86_64 | amd64 | i386 | i686) ;;
	*)
		warn "Wine non installato: su $(uname -m) non può eseguire programmi Windows per x86."
		return
		;;
	esac

	local id="" like=""
	if [[ -r /etc/os-release ]]; then
		# shellcheck disable=SC1091 # file di sistema
		id="$(. /etc/os-release && echo "${ID:-}")"
		# shellcheck disable=SC1091
		like="$(. /etc/os-release && echo "${ID_LIKE:-}")"
	fi

	info "Installazione di Wine (richiede sudo)"
	case " ${id} ${like} " in
	*" debian "* | *" ubuntu "*)
		if [[ "$(dpkg --print-architecture)" == "amd64" ]]; then
			# La parte a 32 bit di Wine serve per provare l'eseguibile x86.
			if ! dpkg --print-foreign-architectures | grep -qx i386; then
				sudo dpkg --add-architecture i386
			fi
			sudo apt-get update
			sudo apt-get install -y wine wine64 wine32:i386
		else
			sudo apt-get update
			sudo apt-get install -y wine
		fi
		;;
	*" fedora "*) sudo dnf install -y wine ;;
	*" arch "*) sudo pacman -S --needed --noconfirm wine ;;
	*" suse "* | *" opensuse "*) sudo zypper --non-interactive install wine ;;
	*) fail "Distribuzione non riconosciuta (${id:-sconosciuta}): installa Wine con il gestore di pacchetti e rilancia lo script." ;;
	esac
}

main() {
	parse_args "$@"
	check_requirements
	install_go
	configure_path
	install_tools
	download_modules
	install_wine

	info "Completato: $(go version)"
	echo "Apri un nuovo terminale (o esegui 'source ~/.profile') per avere go nel PATH."
}

main "$@"
