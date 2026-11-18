#!/usr/bin/env bash
# SPDX-License-Identifier: GPL-3.0-or-later
#
# Installa le dipendenze di sviluppo di Support Tools su macOS:
#   - toolchain Go 1.20.14 in /usr/local/go (richiede sudo);
#   - strumenti di sviluppo: rsrc (manifest/icona) e goimports;
#   - moduli Go elencati in go.mod;
#   - Wine (build ufficiale per macOS) in /Applications, per provare
#     l'eseguibile senza Windows (go run ./build -run); su Apple Silicon
#     installa anche Rosetta 2, necessaria a Wine.
#
# Lo script è idempotente: può essere rieseguito senza effetti collaterali.
# Uso: scripts/setup-macos.sh [--force]
#   --force  sostituisce un'eventuale versione di Go diversa già presente
#            in /usr/local/go e reinstalla Wine.

set -euo pipefail

# Versioni fissate: Go 1.20.x è l'ultima release che supporta Windows 7.
readonly GO_VERSION="1.20.14"
readonly GO_ROOT="/usr/local/go"
readonly RSRC_VERSION="v0.10.2"
# Ultima versione di x/tools compatibile con Go 1.20.
readonly GOIMPORTS_VERSION="v0.24.0"

# Wine per macOS: Homebrew ha disattivato il cask wine-stable il 2026-09-01,
# quindi si usa direttamente la build ufficiale a cui il cask puntava.
readonly WINE_VERSION="11.0_1"
readonly WINE_ARCHIVE="wine-stable-${WINE_VERSION}-osx64.tar.xz"
readonly WINE_URL="https://github.com/Gcenx/macOS_Wine_builds/releases/download/${WINE_VERSION}/${WINE_ARCHIVE}"
readonly WINE_SHA256="b50dc50ec7f41d58b115a6b685d4d1315ba3c797bd3aa0f49213f2703cb82388"
readonly WINE_APP="/Applications/Wine Stable.app"
readonly WINE_BIN="${WINE_APP}/Contents/Resources/wine/bin"

# Marcatore usato per riconoscere la riga aggiunta al profilo della shell.
readonly PROFILE_MARKER="# Support Tools: toolchain Go"
readonly WINE_PROFILE_MARKER="# Support Tools: Wine"

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
	local os="darwin" arch
	# Su Apple Silicon uname -m restituisce x86_64 se il terminale gira sotto
	# Rosetta: sysctl indica l'hardware reale.
	if [[ "$(sysctl -n hw.optional.arm64 2>/dev/null || echo 0)" == "1" ]]; then
		arch="arm64"
	else
		case "$(uname -m)" in
		x86_64) arch="amd64" ;;
		arm64) arch="arm64" ;;
		*) fail "Architettura non supportata: $(uname -m)" ;;
		esac
	fi

	local file="go${GO_VERSION}.${os}-${arch}.tar.gz"
	local sum
	# Checksum ufficiali pubblicati su https://go.dev/dl/
	case "$file" in
	go1.20.14.darwin-amd64.tar.gz) sum="754363489e2244e72cb49b4ec6ddfd6a2c60b0700f8c4876e11befb1913b11c5" ;;
	go1.20.14.darwin-arm64.tar.gz) sum="6da3f76164b215053daf730a9b8f1d673dbbaa4c61031374a6744b75cb728641" ;;
	*) fail "Nessun archivio Go disponibile per ${os}/${arch}" ;;
	esac
	printf '%s %s\n' "$file" "$sum"
}

# ensure_tmp crea, una sola volta, la cartella temporanea dei download,
# eliminata all'uscita dello script.
ensure_tmp() {
	if [[ -z "$tmp" ]]; then
		tmp="$(mktemp -d)"
		trap 'rm -rf "${tmp:-}"' EXIT
	fi
}

sha256() {
	shasum -a 256 "$1" | cut -d' ' -f1
}

check_requirements() {
	if [[ "$(uname -s)" != "Darwin" ]]; then
		fail "Questo script è per macOS: su Linux usa scripts/setup-linux.sh, su Windows scripts/setup-windows.ps1."
	fi

	local missing=()
	for cmd in curl tar sudo shasum; do
		command -v "$cmd" >/dev/null 2>&1 || missing+=("$cmd")
	done
	# Su macOS git fa parte degli Xcode Command Line Tools.
	if ! xcode-select -p >/dev/null 2>&1; then
		fail "Xcode Command Line Tools assenti: esegui 'xcode-select --install' e rilancia lo script."
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

	ensure_tmp

	info "Download di ${file}"
	curl -fL --progress-bar -o "${tmp}/${file}" "https://go.dev/dl/${file}"

	info "Verifica del checksum"
	if [[ "$(sha256 "${tmp}/${file}")" != "$sum" ]]; then
		fail "Checksum non valido per ${file}: download corrotto o manomesso."
	fi

	info "Installazione in ${GO_ROOT} (richiede sudo)"
	sudo rm -rf "$GO_ROOT"
	sudo mkdir -p "$(dirname "$GO_ROOT")"
	sudo tar -C "$(dirname "$GO_ROOT")" -xzf "${tmp}/${file}"
}

# login_profile stampa il file letto dalla shell di login dell'utente:
# zsh (predefinita da macOS 10.15) legge ~/.zprofile, bash ~/.bash_profile.
login_profile() {
	case "$(basename "${SHELL:-/bin/zsh}")" in
	zsh) printf '%s\n' "${HOME}/.zprofile" ;;
	bash) printf '%s\n' "${HOME}/.bash_profile" ;;
	*) printf '%s\n' "${HOME}/.profile" ;;
	esac
}

# configure_path rende disponibili go e gli strumenti installati, sia per
# questa esecuzione sia per le shell future (tramite il profilo di login).
configure_path() {
	export PATH="${GO_ROOT}/bin:${PATH}"
	local gobin
	gobin="$(go env GOPATH)/bin"
	export PATH="${gobin}:${PATH}"

	local profile
	profile="$(login_profile)"
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

# install_rosetta installa Rosetta 2 su Apple Silicon: la build di Wine per
# macOS è per Intel.
install_rosetta() {
	if [[ "$(sysctl -n hw.optional.arm64 2>/dev/null || echo 0)" != "1" ]]; then
		return
	fi
	if arch -x86_64 /usr/bin/true >/dev/null 2>&1; then
		return
	fi
	info "Installazione di Rosetta 2 (necessaria a Wine, richiede sudo)"
	sudo softwareupdate --install-rosetta --agree-to-license
}

install_wine() {
	install_rosetta

	if [[ -x "${WINE_BIN}/wine" && "$force" -ne 1 ]]; then
		info "Wine già installato: $("${WINE_BIN}/wine" --version 2>/dev/null || echo "$WINE_APP")"
		return
	fi

	ensure_tmp
	info "Download di Wine ${WINE_VERSION}"
	curl -fL --progress-bar -o "${tmp}/${WINE_ARCHIVE}" "$WINE_URL"

	info "Verifica del checksum"
	if [[ "$(sha256 "${tmp}/${WINE_ARCHIVE}")" != "$WINE_SHA256" ]]; then
		fail "Checksum non valido per ${WINE_ARCHIVE}: download corrotto o manomesso."
	fi
	tar -C "$tmp" -xJf "${tmp}/${WINE_ARCHIVE}"

	info "Installazione in ${WINE_APP}"
	local sudo_cmd=()
	if [[ ! -w "$(dirname "$WINE_APP")" ]]; then
		sudo_cmd=(sudo)
	fi
	"${sudo_cmd[@]}" rm -rf "$WINE_APP"
	"${sudo_cmd[@]}" mv "${tmp}/$(basename "$WINE_APP")" "$WINE_APP"
	# Il checksum è già verificato: si toglie l'eventuale attributo di
	# quarantena perché Gatekeeper non blocchi Wine, che non è autenticato
	# da Apple. Riguarda solo questa app, non disattiva Gatekeeper.
	"${sudo_cmd[@]}" xattr -dr com.apple.quarantine "$WINE_APP" 2>/dev/null || true

	local profile
	profile="$(login_profile)"
	if [[ -f "$profile" ]] && grep -qF "$WINE_PROFILE_MARKER" "$profile"; then
		return
	fi
	info "Aggiunta di Wine al PATH in ${profile}"
	{
		printf '\n%s\n' "$WINE_PROFILE_MARKER"
		# shellcheck disable=SC2016 # espansione voluta al login
		printf 'export PATH="%s:$PATH"\n' "$WINE_BIN"
	} >>"$profile"
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
	echo "Apri un nuovo terminale (o esegui 'source $(login_profile)') per avere go nel PATH."
}

main "$@"
