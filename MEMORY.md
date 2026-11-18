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
- Lo script `scripts/setup.sh` (Linux e macOS) installa l'ambiente di sviluppo ed è idempotente:
  - Go 1.20.14 in `/usr/local/go` (con sudo), con verifica del checksum ufficiale; se è presente un'altra versione si ferma, salvo `--force`;
  - aggiunge `/usr/local/go/bin` e `~/go/bin` al `PATH` in `~/.profile`;
  - installa `rsrc` v0.10.2 (MIT, solo strumento di build: incorpora manifest e icona nell'eseguibile) e `goimports` da `golang.org/x/tools` v0.24.0 (ultima versione compatibile con Go 1.20);
  - scarica e verifica i moduli di `go.mod`.
- Moduli Go adottati: `lxn/walk` (v0.0.0-20210112085537-c389da54e794) e `lxn/win` (v0.0.0-20210218163916-a377121e959e), ultime versioni disponibili; portano con sé `golang.org/x/sys` (v0.0.0-20201018230417-eeed37f84f13) e `gopkg.in/Knetic/govaluate.v3` v3.0.0 (MIT, compatibile con la GPL v3).
- Finché il codice non importa walk, in `go.mod` le dipendenze risultano `// indirect`: **non eseguire `go mod tidy`** prima di averle importate, altrimenti vengono rimosse.
- Variabile `GOTOOLCHAIN=local` per evitare che gli strumenti richiedano una versione di Go più recente.

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
