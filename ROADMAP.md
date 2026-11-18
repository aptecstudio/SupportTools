# Roadmap — Support Tools

Legenda: `[x]` completato · `[ ]` da fare · `[~]` in corso

## Obiettivo

Tool di supporto tecnico per Windows (da Windows 7 in poi), scritto in Go, compilabile da Linux, Windows e macOS, con interfaccia grafica semplice e modulare.

## Fase 0 — Inizializzazione

- [x] Creazione `AGENTS.md`, `ROADMAP.md`, `MEMORY.md`, `CHANGELOG.md`, `README.md`
- [x] Definizione vincoli di compatibilità (Windows 7+) e toolchain Go
- [x] Scelta della libreria per l'interfaccia grafica (lxn/walk) e della lingua dell'interfaccia (italiano)
- [x] Struttura del progetto Go (moduli, cartelle, sistema a moduli dell'interfaccia)
- [x] Finestra principale con barra laterale e primo tool "Informazioni"
- [x] Script di build cross-platform (Linux, Windows, macOS → eseguibile Windows): `go run ./build`
- [x] Manifest dell'applicazione (privilegi di amministratore, stili visuali)
- [ ] Verifica dell'avvio su Windows 7 e Windows 10/11 (x86 e x64), compreso il componente di avanzamento (build `-tags demo`)
- [ ] Icona dell'applicazione (`cmd/supporttools/supporttools.ico`)
- [x] Componente riutilizzabile di avanzamento per le operazioni lente: barra con percentuale, tempo rimanente, sezione "Dettagli" espandibile con le operazioni concluse e in lavorazione
- [x] Licenza GPL v3.0 o successive e verifica di compatibilità con Go
- [x] File `THIRD_PARTY_NOTICES.md` con gli avvisi di licenza di terze parti
- [x] Aggiunta degli avvisi di lxn/walk, lxn/win, golang.org/x/sys e govaluate all'introduzione delle dipendenze
- [x] Wine tra le dipendenze di Linux e macOS e avvio di prova con `go run ./build -run`
- [x] Script di installazione delle dipendenze di sviluppo per Linux, macOS e Windows (`scripts/setup-linux.sh`, `scripts/setup-macos.sh`, `scripts/setup-windows.ps1`)
- [x] Definizione del canale di distribuzione (GitHub Releases)
- [ ] Pubblicazione automatica dell'eseguibile su GitHub Releases alla creazione di un tag `vX.Y.Z`

## Fase 1 — Gestione utenti locali

- [x] Pagina unica "Gestione utenti" con l'elenco di tutti gli account locali (nome, nome completo, tipo, stato)
- [~] Creazione utente locale (nome, nome completo, password, gruppo: Utenti/Amministratori, "password senza scadenza", "cambio al primo accesso") — implementata, da verificare su Windows
- [~] Eliminazione completa di un utente locale (account, profilo, cartella utente, voci di registro del profilo) con conferma digitando il nome e avanzamento — implementata, da verificare su Windows
- [~] Cambio password di un utente locale — implementato, da verificare su Windows
- [~] Rimozione della password di un utente locale — implementata, da verificare su Windows
- [ ] Verifica completa su Windows 7 e Windows 10/11 (x86 e x64), compresi Windows in lingue diverse dall'italiano
- [ ] Valutare la presa di possesso dei file protetti del profilo che l'amministratore non può eliminare (oggi vengono segnalati)

→ **Rilascio `v0.1.0`** al primo funzionamento concreto.

## Fase 2 — Backup di file e impostazioni utente

- [ ] Selezione dell'utente sorgente e del target di destinazione (cartella, disco esterno, percorso di rete)
- [ ] Copia dei file utente (Desktop, Documenti, Immagini, Musica, Video, Download, ecc.) con selezione delle cartelle
- [ ] Rilevamento dei browser installati sull'account (Chrome, Edge, Firefox, Opera, Brave, ecc.)
- [ ] Copia opzionale, per singolo browser, di impostazioni/profili e password salvate
- [ ] Copia delle impostazioni personali di sistema, con selezione delle voci (es. sfondo, stampanti, unità di rete, Wi-Fi, impostazioni regionali, …)
- [ ] File manifest del backup (contenuto, versione, data, utente sorgente)

## Fase 3 — Ripristino

- [ ] Lettura di un backup creato nella Fase 2
- [ ] Selezione dell'utente di destinazione e delle voci da ripristinare
- [ ] Ripristino di file, impostazioni dei browser, password e impostazioni di sistema

## Punti di attenzione tecnici

- Le password dei browser basati su Chromium e le credenziali Windows sono cifrate con DPAPI e legate all'utente/macchina: per copiarle su un altro utente o PC vanno decifrate nel contesto dell'utente sorgente e ricifrate in quello di destinazione (o esportate in formato portabile).
- Le operazioni sugli utenti richiedono privilegi di amministratore.
- Su Windows 7 alcune API e percorsi differiscono da Windows 10/11: ogni funzione va verificata su entrambi.
