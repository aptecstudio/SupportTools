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

## Convenzioni

### Git
- Branch unico: `main`.
- Messaggi di commit nel formato [Conventional Commits](https://www.conventionalcommits.org/it/): tipo in inglese (`feat`, `fix`, `docs`, `refactor`, `chore`, `build`, `test`), descrizione in italiano. Esempio: `feat(users): aggiunta creazione utente locale`.
- Versioning SemVer con tag `vX.Y.Z`; versione corrente nel file `VERSION`.

### Codice
- Codice e nomenclature in inglese, commenti in italiano.
- Formattazione con `gofmt`, controllo con `go vet` prima di ogni commit.

## Da decidere

- Libreria per l'interfaccia grafica.
- Lingua dei testi dell'interfaccia utente (solo italiano o multilingua).
- Formato del pacchetto di backup (cartella, archivio ZIP, cifratura).
