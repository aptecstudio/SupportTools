# Support Tools

Tool di supporto tecnico per Windows (da Windows 7 in poi), scritto in Go, con interfaccia grafica semplice e modulare.

> Progetto in fase di inizializzazione: nessuna versione ancora rilasciata.

## Funzionalità previste

- Creazione di utenti locali
- Eliminazione completa di utenti locali
- Cambio e rimozione della password di un utente
- Backup di file e impostazioni utente (browser, password, impostazioni di sistema) su un target specifico
- Ripristino dei backup

Dettagli e stato di avanzamento in [ROADMAP.md](ROADMAP.md); scelte tecniche in [MEMORY.md](MEMORY.md).

## Download

La distribuzione avviene direttamente da GitHub:

- **Eseguibile pronto all'uso:** scarica `SupportTools.exe` dalla pagina [Releases](https://github.com/aptecstudio/SupportTools/releases) (disponibile dalla prima versione).
- **Sorgenti:** clona il repository o scarica l'archivio dei sorgenti dalla stessa pagina e compila in autonomia.

## Build dai sorgenti

Requisiti:

- Go 1.20.x (necessario per la compatibilità con Windows 7)
- Qualsiasi sistema operativo tra Linux, Windows e macOS

L'ambiente di sviluppo (Go 1.20.14, `rsrc`, `goimports`, i moduli Go e, su Linux e macOS, Wine) si installa con lo script del proprio sistema:

| Sistema | Comando | Go installato in |
|---------|---------|------------------|
| Linux | `scripts/setup-linux.sh` | `/usr/local/go` (con `sudo`) |
| macOS | `scripts/setup-macos.sh` | `/usr/local/go` (con `sudo`) |
| Windows | `powershell -NoProfile -ExecutionPolicy Bypass -File scripts\setup-windows.ps1` | `%LOCALAPPDATA%\Programs\Go` (senza privilegi di amministratore) |

Se è già presente un'altra versione di Go nella cartella di installazione, lo script si ferma: usa `--force` (Linux e macOS) o `-Force` (Windows) per sostituirla. Su macOS servono gli Xcode Command Line Tools (`xcode-select --install`), su Windows PowerShell 5.1 o successivo e Git.

Wine serve a provare il programma senza Windows:

- **Linux:** installato con il gestore di pacchetti della distribuzione (Debian/Ubuntu e derivate, Fedora, Arch, openSUSE); su Debian e Ubuntu a 64 bit viene abilitata l'architettura `i386` per eseguire anche la versione x86.
- **macOS:** Homebrew non distribuisce più Wine (cask `wine-stable` disattivato a settembre 2026), quindi lo script scarica la build ufficiale per macOS, ne verifica il checksum e la installa in `/Applications/Wine Stable.app`. Su Apple Silicon installa anche Rosetta 2. Serve macOS 10.15 o successivo.

Poi, dalla radice del repository, su qualsiasi sistema:

```sh
go run ./build
```

Il programma controlla formattazione e codice (`gofmt`, `go vet`), incorpora il manifest e compila `dist/x64/SupportTools.exe` e `dist/x86/SupportTools.exe`, con `LICENSE` e `THIRD_PARTY_NOTICES.md` accanto. Opzioni: `-arch x64` o `-arch x86` per compilare una sola architettura, `-tags demo` per includere gli strumenti di prova (output in `dist/<arch>-demo/`, da non distribuire), `-skip-checks` per saltare i controlli.

I test si eseguono con `go test ./...`.

## Provare il programma

Su qualsiasi sistema, dalla radice del repository:

```sh
go run ./build -run              # compila e avvia la versione x64
go run ./build -run -tags demo   # include gli strumenti di prova
```

Su Windows il programma parte direttamente (con la richiesta dei privilegi di amministratore); su Linux e macOS parte con Wine, in un prefisso dedicato (`~/.cache/SupportTools/wine` su Linux, `~/Library/Caches/SupportTools/wine` su macOS) che non tocca la configurazione Wine dell'utente. Il primo avvio richiede qualche minuto.

Wine è adatto a verificare l'interfaccia, ma non sostituisce Windows: gestione utenti, profili, registro e password dei browser vanno provati su Windows 7 e Windows 10/11 reali o in macchina virtuale.

## Licenza

Support Tools è software libero: puoi redistribuirlo e/o modificarlo secondo i termini della [GNU General Public License](LICENSE), versione 3 o (a tua scelta) qualsiasi versione successiva.

Gli avvisi di licenza dei componenti di terze parti sono in [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md).
