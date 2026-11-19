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

L'ambiente di sviluppo (Go 1.20.14, `rsrc`, `goimports`, i moduli Go e, su Linux e macOS, Wine e QEMU) si installa con lo script del proprio sistema:

| Sistema | Comando | Go installato in |
|---------|---------|------------------|
| Linux | `scripts/setup-linux.sh` | `/usr/local/go` (con `sudo`) |
| macOS | `scripts/setup-macos.sh` | `/usr/local/go` (con `sudo`) |
| Windows | `powershell -NoProfile -ExecutionPolicy Bypass -File scripts\setup-windows.ps1` | `%LOCALAPPDATA%\Programs\Go` (senza privilegi di amministratore) |

Se è già presente un'altra versione di Go nella cartella di installazione, lo script si ferma: usa `--force` (Linux e macOS) o `-Force` (Windows) per sostituirla. Su macOS servono gli Xcode Command Line Tools (`xcode-select --install`), su Windows PowerShell 5.1 o successivo e Git.

Wine serve a provare il programma senza Windows:

- **Linux:** installato con il gestore di pacchetti della distribuzione (Debian/Ubuntu e derivate, Fedora, Arch, openSUSE).
- **macOS:** Homebrew non distribuisce più Wine (cask `wine-stable` disattivato a settembre 2026), quindi lo script scarica la build ufficiale per macOS, ne verifica il checksum e la installa in `/Applications/Wine Stable.app`. Su Apple Silicon installa anche Rosetta 2. Serve macOS 10.15 o successivo.

Poi, dalla radice del repository, su qualsiasi sistema:

```sh
go run ./build
```

Il programma controlla formattazione e codice (`gofmt`, `go vet`), incorpora il manifest e compila `dist/x64/SupportTools.exe`, con `LICENSE` e `THIRD_PARTY_NOTICES.md` accanto. Opzioni: `-tags demo` per includere gli strumenti di prova (output in `dist/x64-demo/`, da non distribuire), `-skip-checks` per saltare i controlli.

L'icona (`cmd/supporttools/supporttools.ico`) viene disegnata dal codice: dopo averne modificato forme o colori in `build/icon/main.go` si rigenera con `go run ./build/icon`, aggiungendo `-preview <cartella>` per salvare un'anteprima PNG di ogni dimensione.

I test si eseguono con `go test ./...`.

## Provare il programma

Su qualsiasi sistema, dalla radice del repository:

```sh
go run ./build -run              # compila e avvia la versione x64
go run ./build -run -tags demo   # include gli strumenti di prova
```

Su Windows il programma parte direttamente (con la richiesta dei privilegi di amministratore); su Linux e macOS parte con Wine, in un prefisso dedicato (`~/.cache/SupportTools/wine` su Linux, `~/Library/Caches/SupportTools/wine` su macOS) che non tocca la configurazione Wine dell'utente. Il primo avvio richiede qualche minuto.

Wine è adatto a verificare l'interfaccia, ma non sostituisce Windows: gestione utenti, profili, registro e password dei browser vanno provati su Windows vero, per esempio nelle macchine virtuali di test.

## Macchine virtuali di test

Su Linux e macOS il programma `vm` crea e usa con QEMU due macchine virtuali Windows: `win10` (Windows 10 x64, prova l'eseguibile x64) e `win7` (Windows 7 SP1 x64, prova l'eseguibile x64 e la compatibilità con Windows 7). Le ISO di Windows vanno procurate a parte e non vanno messe nel repository (per esempio in `~/VM/iso/`).

```sh
go run ./vm install win10 -iso ~/VM/iso/win10_x64.iso   # una volta sola, 20-60 minuti
go run ./vm test win10                                   # test di integrazione su Windows
go run ./vm screenshot win10 -tags demo                  # schermata del programma in dist/vm/
go run ./vm start win10                                  # finestra della VM con il programma aperto
go run ./vm list                                         # stato delle macchine
```

- L'installazione è non presidiata: crea l'amministratore `tester` (password `SupportTools1!`, solo per queste macchine usa e getta), attiva l'accesso automatico e installa l'agente che esegue i comandi inviati dall'host. Opzioni: `-lang` (lingua della ISO, predefinita `it-IT`), `-edition` o `-index` (edizione da installare), `-key` (Product Key), `-headless`, `-force`.
- `test`, `screenshot` e `start` partono sempre da Windows appena installato: tutte le modifiche vengono scartate allo spegnimento (salvo `start -persist`).
- Servono l'accelerazione KVM (Linux) o Hypervisor.framework (Mac Intel). Sui Mac Apple Silicon Windows x64 si può solo emulare (`-tcg`), molto lentamente.
- I dischi stanno in `~/VM/SupportTools` (o nella cartella indicata da `SUPPORTTOOLS_VM_DIR`).

I test di integrazione (tag `integration`) creano ed eliminano account e profili reali: vanno eseguiti solo nelle macchine virtuali, mai sul proprio computer.

## Licenza

Support Tools è software libero: puoi redistribuirlo e/o modificarlo secondo i termini della [GNU General Public License](LICENSE), versione 3 o (a tua scelta) qualsiasi versione successiva.

Gli avvisi di licenza dei componenti di terze parti sono in [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md).
