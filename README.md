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

Su Linux e macOS l'ambiente di sviluppo (Go 1.20.14, `rsrc`, `goimports` e i moduli Go) si installa con:

```sh
scripts/setup.sh
```

Lo script usa `sudo` per installare Go in `/usr/local/go`; se è già presente un'altra versione si ferma (usa `--force` per sostituirla).

Le istruzioni di compilazione saranno aggiunte insieme allo script di build.

## Licenza

Support Tools è software libero: puoi redistribuirlo e/o modificarlo secondo i termini della [GNU General Public License](LICENSE), versione 3 o (a tua scelta) qualsiasi versione successiva.

Gli avvisi di licenza dei componenti di terze parti sono in [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md).
