# Roadmap — Support Tools

Legenda: `[x]` completato · `[ ]` da fare · `[~]` in corso

## Obiettivo

Tool di supporto tecnico per Windows (da Windows 7 in poi), scritto in Go, compilabile da Linux, Windows e macOS, con interfaccia grafica semplice e modulare.

## Fase 0 — Inizializzazione

- [x] Creazione `AGENTS.md`, `ROADMAP.md`, `MEMORY.md`, `CHANGELOG.md`, `README.md`
- [x] Definizione vincoli di compatibilità (Windows 7+) e toolchain Go
- [ ] Scelta della libreria per l'interfaccia grafica
- [ ] Struttura del progetto Go (moduli, cartelle, sistema a moduli dell'interfaccia)
- [ ] Script di build cross-platform (Linux, Windows, macOS → eseguibile Windows)
- [ ] Manifest dell'applicazione (privilegi di amministratore, stili visuali)

## Fase 1 — Gestione utenti locali

- [ ] Creazione utente locale (nome, nome completo, password, gruppo: Utenti/Amministratori)
- [ ] Eliminazione completa di un utente locale (account, profilo, cartella utente, voci di registro del profilo)
- [ ] Cambio password di un utente locale
- [ ] Rimozione della password di un utente locale

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
