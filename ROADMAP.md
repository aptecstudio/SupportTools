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
- [ ] Verifica dell'avvio su Windows 7 e Windows 10/11 (x64), compreso il componente di avanzamento (build `-tags demo`)
- [x] Icona dell'applicazione (`cmd/supporttools/supporttools.ico`, generata con `go run ./build/icon`) e icona della finestra principale
- [x] Componente riutilizzabile di avanzamento per le operazioni lente: barra con percentuale, tempo rimanente, sezione "Dettagli" espandibile con le operazioni concluse e in lavorazione
- [x] Licenza GPL v3.0 o successive e verifica di compatibilità con Go
- [x] File `THIRD_PARTY_NOTICES.md` con gli avvisi di licenza di terze parti
- [x] Aggiunta degli avvisi di lxn/walk, lxn/win, golang.org/x/sys e govaluate all'introduzione delle dipendenze
- [x] Wine tra le dipendenze di Linux e macOS e avvio di prova con `go run ./build -run`
- [x] Macchine virtuali di test con QEMU (`go run ./vm`): installazione non presidiata di Windows 10 x64 e Windows 7 x64, agente, test di integrazione e schermate — **win10 e win7 installate e verificate il 2026-11-19** (correzione env snap per QEMU e `SkipMachineOOBE` per l'OOBE di Win7)
- [ ] Invio di clic e tasti all'interfaccia nelle macchine virtuali (QMP), per automatizzare le prove dell'interfaccia
- [x] Script di installazione delle dipendenze di sviluppo per Linux, macOS e Windows (`scripts/setup-linux.sh`, `scripts/setup-macos.sh`, `scripts/setup-windows.ps1`)
- [x] Definizione del canale di distribuzione (GitHub Releases)
- [ ] Pubblicazione automatica dell'eseguibile su GitHub Releases alla creazione di un tag `vX.Y.Z`

## Fase 1 — Gestione utenti locali

- [x] Pagina unica "Gestione utenti" con l'elenco di tutti gli account locali (nome, nome completo, tipo, stato)
- [~] Creazione utente locale (nome, nome completo, password, gruppo: Utenti/Amministratori, "password senza scadenza", "cambio al primo accesso") — implementata, da verificare su Windows
- [~] Eliminazione completa di un utente locale (account, profilo, cartella utente, voci di registro del profilo) con conferma digitando il nome e avanzamento — implementata, da verificare su Windows
- [~] Cambio password di un utente locale — implementato, da verificare su Windows
- [~] Rimozione della password di un utente locale — implementata, da verificare su Windows
- [~] Test di integrazione della gestione utenti (creazione, opzioni, password, eliminazione con profilo, giunzioni, protezioni) — scritti, da eseguire nelle macchine virtuali
- [ ] Verifica completa su Windows 7 e Windows 10/11 (x64), compresi Windows in lingue diverse dall'italiano
- [ ] Valutare la presa di possesso dei file protetti del profilo che l'amministratore non può eliminare (oggi vengono segnalati)

→ **Rilascio `v0.1.0`** al primo funzionamento concreto.

## Fase 2 — Backup di file e impostazioni utente

Formato del backup: una cartella per backup (es. `Backup_<utente>_<data>`) con i file copiati così come sono e un `manifest.json`; sfogliabile da Esplora risorse e recuperabile anche senza il programma. Dettagli in `MEMORY.md`.

### Selezione sorgente e destinazione

- [ ] Selezione dell'utente sorgente tra gli account locali
- [ ] Destinazione: cartella qualsiasi (disco interno, USB, percorso di rete `\\server\condivisione`)
- [ ] Richiesta di credenziali per le condivisioni di rete che le richiedono
- [ ] Controllo dello spazio libero sulla destinazione prima di iniziare

### File dell'utente

- [ ] Cartelle standard con dimensione accanto a ciascuna: Desktop, Documenti, Immagini, Musica, Video, Download, Preferiti
- [ ] Aggiunta di cartelle qualsiasi del profilo o del disco
- [ ] Dischi fissi fisici diversi dal disco di sistema, selezionabili per intero
- [ ] Cartelle nella radice del disco di sistema che non sono cartelle di sistema (es. `C:\Dati`)
- [ ] Copia con avanzamento, continuazione dopo un errore sul singolo file e riepilogo dei file non copiati

### Browser (Chrome, Edge, Firefox)

- [ ] Rilevamento dei browser installati e dei relativi profili sull'account
- [ ] Selezione per voce e per browser: segnalibri, password, cronologia, estensioni, impostazioni
- [ ] Copia dei dati dei browser basati su Chromium (Chrome, Edge), con esclusione di cache e file temporanei
- [ ] Copia del profilo di Firefox (formato proprio, portabile su un altro PC)
- [ ] Password dei browser Chromium: lettura nel contesto dell'utente sorgente e salvataggio nel backup protetto da una password del backup scelta dal tecnico
- [ ] Gestione dell'utente sorgente senza password (vedi `MEMORY.md`: sblocco temporaneo del criterio con ripristino garantito)

### Impostazioni di sistema (prima versione)

- [ ] Reti Wi-Fi salvate, con le relative password
- [ ] Stampanti dell'utente (con la predefinita) e unità di rete collegate (lettera → percorso)
- [ ] Sfondo e aspetto: immagine e colore dello sfondo, tema chiaro/scuro, colore principale
- [ ] Lingua, layout di tastiera e formati di data, ora e numeri

### Manifest

- [ ] `manifest.json` del backup: contenuto, versione del formato, data, utente sorgente, computer, voci incluse

## Fase 2b — Recupero offline delle password dei browser (Windows non avviabile)

Scenario distinto dalla Fase 2: recuperare le password dei browser di un cliente quando il suo Windows non si avvia, da ambiente di ripristino, previa autorizzazione del cliente e con la sua password Windows. Dettagli in `MEMORY.md`.

- [x] Valutazione delle librerie Go e scelta (2026-11-19): `wat4r/dpapitk` (Apache-2.0, Go 1.19, DPAPI + CNG) per i primitivi, pezzi mancanti implementati in proprio — opzione A, un solo binario Go 1.20. Dettagli in `MEMORY.md`.
- [ ] Integrazione di `wat4r/dpapitk` e avviso Apache-2.0 in `THIRD_PARTY_NOTICES.md`
- [ ] Decifratura del livello utente (DPAPI, chiave AES in `Local State`): masterkey utente sbloccato con la password Windows del cliente
- [ ] Parsing degli hive `SYSTEM`/`SECURITY` dal disco → bootkey → LSA secret `DPAPI_SYSTEM` → masterkey di macchina (eventuale port del codice Apache-2.0 da `mandiant/gopacket`)
- [ ] Decifratura del livello ABE/`v20`: scarto dei due livelli DPAPI (utente + macchina) e AES-GCM finale
- [ ] Rilevamento dei profili browser e degli hive di registro da un **disco montato** su un PC Windows funzionante
- [ ] Funzionamento in **WinPE** avviato sul PC del cliente (verifica GUI walk e API disponibili)
- [ ] Recupero delle password di Firefox (archivio NSS `key4.db` + `logins.json`, portabile)
- [ ] Conferma esplicita di autorizzazione del cliente da parte del tecnico prima dell'operazione (coerente con gli avvisi degli altri tool)
- [ ] Riuso dello strato di decodifica DPAPI/ABE condiviso con la Fase 2

## Fase 3 — Ripristino

- [ ] Lettura di un backup creato nella Fase 2
- [ ] Selezione dell'utente di destinazione e delle voci da ripristinare
- [ ] Ripristino di file, impostazioni dei browser, password e impostazioni di sistema

## Punti di attenzione tecnici

- Le password dei browser basati su Chromium sono legate all'utente Windows: vanno lette nel contesto dell'utente sorgente e nel backup vengono protette da una password del backup, così da poterle ripristinare su un altro utente o PC nella Fase 3. Le versioni recenti di Chrome/Edge aggiungono una protezione a livello di sistema: va verificato su Windows reale cosa è accessibile eseguendo come amministratore nella sessione dell'utente. Firefox usa un formato proprio già portabile.
- Se l'utente sorgente non ha una password, l'accesso ai suoi dati protetti è limitato: lo sblocco temporaneo del relativo criterio di sicurezza deve sempre ripristinare lo stato iniziale, anche in caso di interruzione del programma.
- Le operazioni sugli utenti richiedono privilegi di amministratore.
- Su Windows 7 alcune API e percorsi differiscono da Windows 10/11: ogni funzione va verificata su entrambi.
