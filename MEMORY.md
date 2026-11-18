# Memoria del progetto — Support Tools

Registro delle scelte tecniche e delle convenzioni. Ogni voce riporta data e motivazione.

## Decisioni

### 2026-11-18 — Nome e scopo
- Nome del tool: **Support Tools**.
- Nome dell'eseguibile: `SupportTools.exe`.
- Modulo Go: `github.com/aptecstudio/supporttools`.

### 2026-11-18 — Piattaforma di destinazione
- Il programma gira su **Windows 7 e successivi** (x86 e x64).
- Si compila da Linux, Windows e macOS tramite cross-compilazione (`GOOS=windows`).
- Per mantenere la cross-compilazione semplice si evita CGO (`CGO_ENABLED=0`) dove possibile.

### 2026-11-18 — Toolchain Go
- **Go 1.20.x** (ultima release: 1.20.14) è la versione da usare per le build di rilascio: è l'ultima versione di Go che supporta ufficialmente Windows 7. Dalla 1.21 in poi gli eseguibili non partono su Windows 7.
- Nel `go.mod` la direttiva è `go 1.20`; le dipendenze scelte devono essere compatibili con Go 1.20.

### 2026-11-18 — Privilegi
- Le funzioni di gestione utenti e copia profili richiedono l'esecuzione come amministratore (manifest con `requireAdministrator`).

### 2026-11-18 — Interfaccia grafica
- Libreria GUI: **[lxn/walk](https://github.com/lxn/walk)**, controlli nativi Win32, senza CGO, compatibile con Windows 7 e cross-compilabile da qualsiasi sistema.
- Testi dell'interfaccia **solo in italiano**, scritti direttamente nel codice.

### 2026-11-18 — Avanzamento delle operazioni lunghe
- Ogni tool che esegue operazioni lente (backup, ripristino, eliminazione profili, copie, ecc.) mostra una finestra o sezione di avanzamento con:
  - **barra di avanzamento** con **percentuale**;
  - **tempo rimanente stimato**;
  - sotto, una **sezione "Dettagli" espandibile** con l'elenco delle operazioni **concluse** e di quella **in lavorazione**.
- È un componente unico e riutilizzabile da tutti i moduli, così che ogni tool lo usi allo stesso modo.
- Le operazioni lente girano fuori dal thread dell'interfaccia, che resta reattiva; gli aggiornamenti arrivano alla GUI tramite il meccanismo di sincronizzazione di walk.
- Implementazione (package `internal/ui/progress`):
  - `progress.Run(owner, titolo, op)` apre una finestra modale ed esegue `op(r *progress.Reporter) error` in una goroutine; restituisce l'errore di `op`;
  - il `Reporter` (utilizzabile da qualsiasi goroutine) offre `SetTotal`/`Add`/`SetDone` per l'avanzamento in un'unità a scelta (byte, file, passaggi), `Step(nome)` per iniziare una nuova operazione nei dettagli (la precedente diventa completata), `SetStatus` per il dettaglio corrente (es. il file in copia) e `Context()` per l'annullamento;
  - i dettagli elencano passaggi di alto livello (cartelle, utenti, browser), **non i singoli file**, che vanno in `SetStatus`;
  - la finestra legge lo stato ogni 200 ms, indipendentemente da quanto spesso l'operazione lo aggiorna; con totale zero la barra è indeterminata;
  - tempo rimanente stimato a velocità costante, mostrato dopo 3 secondi e almeno l'1% di avanzamento, arrotondato per non oscillare;
  - pulsante **Annulla** (anche X ed Esc) con richiesta di conferma: annulla il contesto e la finestra resta aperta finché l'operazione non si ferma; a fine operazione diventa **Chiudi**; in caso di errore i dettagli si aprono da soli;
  - un panic nell'operazione diventa un errore, senza chiudere il programma.

### 2026-11-18 — Distribuzione
- La distribuzione avviene **direttamente da GitHub** (`aptecstudio/SupportTools`).
- Ogni rilascio pubblica su **GitHub Releases** l'eseguibile pronto all'uso (`SupportTools.exe`) insieme ai sorgenti, che restano scaricabili e compilabili da chiunque.

### 2026-11-18 — Licenza
- Il progetto è distribuito con licenza **GNU GPL v3.0 o successive** (SPDX: `GPL-3.0-or-later`, file `LICENSE`).
- Le dipendenze devono avere licenze compatibili con la GPL v3.0. Compatibili: BSD, MIT, Apache-2.0, LGPL. Incompatibili: GPL-2.0-only, licenze proprietarie o con restrizioni d'uso.
- Verifica di compatibilità (2026-11-18):
  - Toolchain, runtime e libreria standard di Go: BSD-3-Clause, più una concessione di brevetti che non aggiunge vincoli. Compatibili. Il compilatore non impone clausole sul codice compilato.
  - lxn/walk e lxn/win: BSD-3-Clause. Compatibili.
  - golang.org/x/sys: BSD-3-Clause. Compatibile.
  - DLL di sistema di Windows: rientrano nell'eccezione "System Libraries" della GPL v3, nessun obbligo.
- Obblighi a ogni rilascio dell'eseguibile:
  - allegare `LICENSE` e `THIRD_PARTY_NOTICES.md` (le licenze BSD richiedono di riprodurre gli avvisi di copyright anche nelle distribuzioni binarie);
  - rendere disponibili i sorgenti corrispondenti alla versione rilasciata (assolto dal tag Git e dall'archivio sorgenti di GitHub Releases).
- Ogni nuova dipendenza va verificata e il suo avviso di licenza aggiunto a `THIRD_PARTY_NOTICES.md`.

### 2026-11-18 — Dipendenze e ambiente di sviluppo
- Uno script di installazione dell'ambiente di sviluppo per ogni sistema, tutti idempotenti e con le stesse versioni fissate:
  - `scripts/setup-linux.sh` e `scripts/setup-macos.sh` (bash): Go 1.20.14 in `/usr/local/go` (con sudo); se è presente un'altra versione si fermano, salvo `--force`. Il `PATH` (`/usr/local/go/bin` e `~/go/bin`) va in `~/.profile` su Linux e nel profilo di login della shell su macOS (`~/.zprofile` per zsh, `~/.bash_profile` per bash). Su macOS si richiedono gli Xcode Command Line Tools e l'architettura si rileva con `sysctl` (corretta anche sotto Rosetta).
  - `scripts/setup-windows.ps1` (PowerShell 5.1+): Go 1.20.14 in `%LOCALAPPDATA%\Programs\Go`, **solo per l'utente corrente e senza privilegi di amministratore**; salvo `-Force`. Aggiunge Go e `%USERPROFILE%\go\bin` al `PATH` dell'utente preservando le voci esistenti (REG_EXPAND_SZ) e avvisa se un altro Go nel `PATH` di sistema ha la precedenza.
  - tutti verificano il checksum ufficiale dell'archivio di Go;
  - installano `rsrc` v0.10.2 (MIT, solo strumento di build: incorpora manifest e icona nell'eseguibile) e `goimports` da `golang.org/x/tools` v0.24.0 (ultima versione compatibile con Go 1.20);
  - scaricano e verificano i moduli di `go.mod`.
- **Wine** è una dipendenza di sviluppo obbligatoria su Linux e macOS (2026-11-18), per provare l'interfaccia senza Windows:
  - Linux: dal gestore di pacchetti (apt, dnf, pacman, zypper); su Debian/Ubuntu amd64 si abilita `i386` e si installa `wine32:i386` per la versione x86. Su architetture diverse da x86 viene saltato con un avviso.
  - macOS: Homebrew ha disattivato il cask `wine-stable` il 2026-09-01 (non supera Gatekeeper), quindi lo script scarica la build ufficiale `wine-stable-11.0_1-osx64.tar.xz` da `Gcenx/macOS_Wine_builds` su GitHub, ne verifica lo SHA-256, la installa in `/Applications/Wine Stable.app`, toglie l'attributo di quarantena solo a quell'app e aggiunge il suo `bin` al `PATH`. Su Apple Silicon installa Rosetta 2. Richiede macOS 10.15+.
- I file `.ps1` sono salvati in **UTF-8 con BOM**: senza BOM Windows PowerShell 5.1 li legge come ANSI e storpia le lettere accentate dei messaggi. Terminatori di riga LF come il resto del progetto (PowerShell li accetta).
- Moduli Go adottati: `lxn/walk` (v0.0.0-20210112085537-c389da54e794) e `lxn/win` (v0.0.0-20210218163916-a377121e959e), ultime versioni disponibili; portano con sé `golang.org/x/sys` (v0.0.0-20201018230417-eeed37f84f13) e `gopkg.in/Knetic/govaluate.v3` v3.0.0 (MIT, compatibile con la GPL v3).
- Dal 2026-11-18 il codice importa walk e win: `go mod tidy` si può eseguire liberamente (le due librerie sono dipendenze dirette, `x/sys` e `govaluate` indirette).
- Variabile `GOTOOLCHAIN=local` per evitare che gli strumenti richiedano una versione di Go più recente.

### 2026-11-18 — Struttura del progetto
- Layout Go standard con `cmd` + `internal`:
  - `cmd/supporttools/`: punto di ingresso (`main.go`), elenco dei tool inclusi (`modules.go`, un import per tool) e manifest dell'applicazione (`supporttools.manifest`); qui va anche l'eventuale icona `supporttools.ico`, che il build incorpora se presente;
  - `internal/ui/`: finestra principale e componenti dell'interfaccia condivisi (qui andrà il componente di avanzamento);
  - `internal/modules/`: registro dei tool (`Register`, `All`, gruppi `Group`); ogni tool è un sottopackage `internal/modules/<tool>` che si registra in `init()` e fornisce una funzione `Build(parent walk.Container) error`;
  - `internal/version/`: versione del programma, impostata in build con `-ldflags -X` dal file `VERSION` (vale `dev` nelle build manuali);
  - `build/`: programma di build portabile.
- I package che usano walk hanno il vincolo `//go:build windows`: sugli altri sistemi `./...` li salta invece di fallire.
- `declarative` di walk si importa con l'alias `decl` (niente dot-import, come da Go Code Review Comments).

### 2026-11-18 — Finestra principale
- **Barra laterale** a sinistra (TreeView a larghezza fissa) con i tool raggruppati per area (Utenti locali, Backup, Ripristino, Generale); a destra la pagina del tool selezionato.
- Le pagine dei tool si creano alla prima apertura e poi restano in memoria (nascoste), così ogni tool conserva il proprio stato quando si cambia sezione.
- Primo tool: **Informazioni** (gruppo Generale) con versione, avviso di licenza GPL e collegamento al repository.

### 2026-11-18 — Build
- Programma di build in Go, identico su tutti i sistemi: `go run ./build [-arch all|x64|x86] [-skip-checks]`.
- Verifica che la toolchain sia Go 1.20.x, esegue `gofmt -l` e `go vet` (GOOS=windows, entrambe le architetture), incorpora manifest e icona con `rsrc` (file `rsrc_windows_<goarch>.syso`, ignorati da Git) e compila con `CGO_ENABLED=0`, `-trimpath`, `-H windowsgui`.
- Output: `dist/x64/SupportTools.exe` e `dist/x86/SupportTools.exe`, ciascuno con `LICENSE` e `THIRD_PARTY_NOTICES.md` accanto.
- Opzione `-tags`: tag di build aggiuntivi; l'output va in `dist/<arch>-<tag>/`. Il tag `demo` include i tool di prova (`internal/modules/demo`, importati da `cmd/supporttools/modules_demo.go`), **mai presenti nei rilasci**.
- Opzione `-run`: avvia l'eseguibile appena compilato (x64, salvo `-arch x86`). Su Windows tramite `cmd /c start`, che rispetta la richiesta di elevazione; su Linux e macOS con Wine, in un prefisso dedicato nella cartella di cache dell'utente (`<UserCacheDir>/SupportTools/wine`), con `WINEDLLOVERRIDES=mscoree,mshtml=` (niente richieste di installare Mono/Gecko) e `WINEDEBUG=-all`.
- La logica non grafica (es. stima dei tempi) sta in file senza vincolo `windows`, così i test girano con `go test ./...` su qualsiasi sistema.

### 2026-11-18 — Manifest
- `requireAdministrator`, Common Controls 6 (necessari a walk per gli stili visuali), `supportedOS` da Windows 7 a Windows 10/11, DPI `PerMonitorV2, PerMonitor` con fallback `dpiAware`.

### 2026-11-18 — Strategia di test
- **Wine** per le prove rapide dell'interfaccia durante lo sviluppo (`go run ./build -run`).
- Wine non sostituisce Windows: API di gestione utenti, profili, registro, DPAPI e privilegi di amministratore si comportano diversamente. Le funzionalità vanno provate su **Windows 7 e Windows 10/11** reali o in macchina virtuale prima di ogni rilascio.

### 2026-11-18 — Gestione utenti locali (Fase 1)
- **Pagina unica** "Gestione utenti" (gruppo Utenti locali): elenco degli utenti locali e pulsanti Nuovo utente, Cambia password, Rimuovi password, Elimina utente, Aggiorna.
- Elenco: **tutti** gli account locali, compresi quelli di sistema e disattivati, con nome, nome completo, tipo (Amministratore/Utente) e stato.
- Creazione: nome utente, nome completo, password e conferma, tipo (Utente/Amministratore), più le opzioni **"La password non scade mai"** (attiva di default) e **"Cambia la password al primo accesso"**, che si escludono a vicenda (come in Windows).
- Eliminazione completa con **conferma digitando il nome dell'utente**, dopo un riepilogo di cosa verrà cancellato. Avviene nel componente di avanzamento: verifica, cancellazione della cartella del profilo con avanzamento file per file, rimozione delle voci di registro del profilo (`DeleteProfileW`), eliminazione dell'account (`NetUserDel`). L'account si elimina per ultimo, così un errore a metà permette di riprovare.
- Protezioni: non si elimina l'utente con cui si sta usando il programma, un utente con il profilo caricato (connesso) né l'ultimo amministratore attivo. La cartella del profilo si cancella a mano solo se è una sottocartella diretta della cartella dei profili (`ProfilesDirectory`); altrimenti se ne occupa solo `DeleteProfileW`. Le giunzioni e i collegamenti (reparse point) si rimuovono senza mai seguirli.
- Gruppi Administrators e Users individuati tramite SID noti (S-1-5-32-544, S-1-5-32-545), perché il nome cambia con la lingua di Windows.
- Cambio e rimozione della password con avviso: reimpostare la password da amministratore fa perdere all'utente l'accesso a file cifrati EFS, credenziali salvate e password dei browser protette da DPAPI.
- Chiamate NetAPI (`netapi32.dll`) e `userenv.dll` dichiarate nel package `internal/winapi`, perché la versione di `golang.org/x/sys` in uso non le espone.

## Convenzioni

### Git
- Branch unico: `main`.
- Messaggi di commit nel formato [Conventional Commits](https://www.conventionalcommits.org/it/): tipo in inglese (`feat`, `fix`, `docs`, `refactor`, `chore`, `build`, `test`), descrizione in italiano. Esempio: `feat(users): aggiunta creazione utente locale`.
- Versioning SemVer con tag `vX.Y.Z`; versione corrente nel file `VERSION`.

### Codice
- Codice e nomenclature in inglese, commenti in italiano.
- Ogni file sorgente inizia con l'intestazione `// SPDX-License-Identifier: GPL-3.0-or-later`.
- Formattazione con `gofmt`, controllo con `go vet` prima di ogni commit.

## Da decidere

- Formato del pacchetto di backup (cartella, archivio ZIP, cifratura).
