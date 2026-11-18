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

### 2026-11-18 — Distribuzione
- La distribuzione avviene **direttamente da GitHub** (`aptecstudio/SupportTools`).
- Ogni rilascio pubblica su **GitHub Releases** l'eseguibile pronto all'uso (`SupportTools.exe`) insieme ai sorgenti, che restano scaricabili e compilabili da chiunque.

### 2026-11-18 — Licenza
- Il progetto è distribuito con licenza **GNU GPL v3.0** (file `LICENSE`).
- Le dipendenze devono avere licenze compatibili con la GPL v3.0 (lxn/walk è BSD-3-Clause: compatibile).

## Convenzioni

### Git
- Branch unico: `main`.
- Messaggi di commit nel formato [Conventional Commits](https://www.conventionalcommits.org/it/): tipo in inglese (`feat`, `fix`, `docs`, `refactor`, `chore`, `build`, `test`), descrizione in italiano. Esempio: `feat(users): aggiunta creazione utente locale`.
- Versioning SemVer con tag `vX.Y.Z`; versione corrente nel file `VERSION`.

### Codice
- Codice e nomenclature in inglese, commenti in italiano.
- Formattazione con `gofmt`, controllo con `go vet` prima di ogni commit.

## Da decidere

- Formato del pacchetto di backup (cartella, archivio ZIP, cifratura).
