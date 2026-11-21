# Memoria del progetto — Support Tools

Registro delle scelte tecniche e delle convenzioni. Ogni voce riporta data e motivazione.

## Decisioni

### 2026-11-18 — Nome e scopo
- Nome del tool: **Support Tools**.
- Nome dell'eseguibile: `SupportTools.exe`.
- Modulo Go: `github.com/aptecstudio/supporttools`.

### 2026-11-18 — Piattaforma di destinazione
- Il programma gira su **Windows 7 e successivi**, **solo a 64 bit (x64)** (vedi la decisione del 2026-11-19 sull'abbandono del 32 bit).
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
  - Linux: dal gestore di pacchetti (apt, dnf, pacman, zypper). Su architetture diverse da x86_64 viene saltato con un avviso.
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
- Programma di build in Go, identico su tutti i sistemi: `go run ./build [-arch all|x64] [-skip-checks]`.
- Verifica che la toolchain sia Go 1.20.x, esegue `gofmt -l` e `go vet` (GOOS=windows, x64), incorpora manifest e icona con `rsrc` (file `rsrc_windows_<goarch>.syso`, ignorati da Git) e compila con `CGO_ENABLED=0`, `-trimpath`, `-H windowsgui`.
- Output: `dist/x64/SupportTools.exe`, con `LICENSE` e `THIRD_PARTY_NOTICES.md` accanto.
- Opzione `-tags`: tag di build aggiuntivi; l'output va in `dist/<arch>-<tag>/`. Il tag `demo` include i tool di prova (`internal/modules/demo`, importati da `cmd/supporttools/modules_demo.go`), **mai presenti nei rilasci**.
- Opzione `-run`: avvia l'eseguibile appena compilato (x64). Su Windows tramite `cmd /c start`, che rispetta la richiesta di elevazione; su Linux e macOS con Wine, in un prefisso dedicato nella cartella di cache dell'utente (`<UserCacheDir>/SupportTools/wine`), con `WINEDLLOVERRIDES=mscoree,mshtml=` (niente richieste di installare Mono/Gecko) e `WINEDEBUG=-all`.
- La logica non grafica (es. stima dei tempi) sta in file senza vincolo `windows`, così i test girano con `go test ./...` su qualsiasi sistema.

### 2026-11-18 — Manifest
- `requireAdministrator`, Common Controls 6 (necessari a walk per gli stili visuali), `supportedOS` da Windows 7 a Windows 10/11, DPI `PerMonitorV2, PerMonitor` con fallback `dpiAware`.

### 2026-11-18 — Icona
- Chiave inglese bianca in diagonale su un quadrato blu arrotondato con sfumatura verticale (`#2F86E8` → `#174A9A`) e leggera ombra.
- Generata dal codice con `go run ./build/icon`: le forme sono geometriche e ogni dimensione viene disegnata alla propria risoluzione, con supersampling 8×8. Il file prodotto, `cmd/supporttools/supporttools.ico`, è versionato.
- Dimensioni: 16, 20, 24, 32, 40, 48, 64 e 256 pixel. Fino a 64 sono bitmap a 32 bit con maschera, compatibili con tutte le versioni di Windows; 256 è PNG, supportato da Vista in poi. Il foro del manico compare solo da 32 pixel in su.
- rsrc assegna l'ID 1 al manifest e l'ID 2 al gruppo dell'icona: la finestra principale carica l'icona con l'ID 2 (`iconResourceID`). Senza icona incorporata resta quella predefinita di Windows.
- Può essere sostituita in qualsiasi momento con un file `.ico` disegnato a mano, con lo stesso nome.

### 2026-11-18 — Strategia di test
- **Wine** per le prove rapide dell'interfaccia durante lo sviluppo (`go run ./build -run`).
- Wine non sostituisce Windows: API di gestione utenti, profili, registro, DPAPI e privilegi di amministratore si comportano diversamente. Le funzionalità vanno provate su **Windows 7 e Windows 10/11** reali o in macchina virtuale prima di ogni rilascio.
- Macchine virtuali di test con **QEMU**, su Linux (accelerazione KVM) e su macOS (accelerazione HVF sui Mac Intel). Scelto al posto di VirtualBox (2026-11-18) perché su Linux è già supportato da KVM, è nei repository delle distribuzioni e permette di automatizzare anche tastiera e mouse dell'interfaccia (QMP).
- Macchine di test (2026-11-18, aggiornato 2026-11-19): **`win10`** (Windows 10 x64, 3 GB di RAM, prova l'eseguibile x64) e **`win7`** (Windows 7 SP1 x64, 2 GB, prova l'eseguibile x64 e la compatibilità con Windows 7). Una alla volta, per i limiti di RAM. Windows 11 solo come verifica finale prima dei rilasci, su un PC con più risorse. Le ISO le procura l'umano e restano fuori dal repository.
- Implementazione (programma `vm/`, `go run ./vm …`):
  - hardware virtuale compatibile con i driver inclusi in Windows 7 e 10, senza virtio: chipset i440fx, dischi IDE, rete e1000 in modalità utente, tavoletta USB, scheda video standard;
  - installazione non presidiata con `autounattend.xml` e `stsetup.cmd` (modelli in `vm/templates/`, convertiti in CRLF) su un secondo CD creato con xorriso (Linux) o hdiutil (macOS); crea l'amministratore `tester`, attiva l'accesso automatico permanente, disattiva Windows Update e la sospensione, esclude `C:\stagent` da Defender;
  - **agente proprio** (`vm/agent`, Go per Windows, compilato per l'architettura della macchina) invece del guest agent di QEMU, i cui driver virtio non supportano più Windows 7: avviato all'accesso di `tester` da un'attività pianificata con privilegi più elevati nella sessione interattiva, chiede i lavori all'host via HTTP (`10.0.2.2:47800`, che QEMU inoltra a `127.0.0.1` dell'host), scarica i file, esegue e invia l'esito. Protocollo in `vm/internal/protocol`: cambiarlo richiede di reinstallare le macchine;
  - controllo della macchina via QMP (socket Unix): schermate PNG, spegnimento e **invio di clic e tasti all'interfaccia** (`input-send-event`). Il puntatore usa la tavoletta USB con coordinate assolute (0..32767 = intero schermo, indipendenti dalla risoluzione); i tasti usano i qcode QEMU. Comando `go run ./vm input <macchina> [-launch] <azione>…` con azioni scriptabili: `move:x,y`, `click[:pulsante]:x,y` (x,y in percentuale), `key:ctrl+a`, `type:testo`, `wait:500ms`, `shot[:nome]`. La mappatura caratteri→qcode è affidabile per lettere, cifre e spazio; i simboli presuppongono il layout di tastiera US del guest. Primitivi in `vm/qmp.go`, mappatura e parsing in `vm/input.go` (testati a unità). **Pressione e rilascio** di un pulsante o tasto vanno inviati in **due eventi distinti** con una pausa (`pressDelay`, 60 ms): insieme, Windows li vede come pressione di durata zero e li ignora. **Verificato dal vivo su Windows 10 x64 (2026-11-21)**: clic, navigazione dell'albero e tasti funzionano. **Attenzione**: la posizione a video della finestra dell'app varia tra un avvio e l'altro, quindi i clic in percentuale di schermo vanno ricalibrati sulla schermata del momento (o la finestra va prima massimizzata); i clic su elementi di sistema a posizione fissa (es. flyout "Reti" in basso a destra) sono invece stabili;
  - `test`, `screenshot` e `start` usano `-snapshot` di QEMU: ogni esecuzione parte da Windows pulito;
  - Windows 10 si installa con la chiave generica pubblica di Windows 10 Pro (non attiva Windows); Windows 7 senza chiave, dall'edizione con indice 1 della ISO.
- **Test di integrazione** con tag di build `integration` (es. `internal/users/integration_windows_test.go`): agiscono su account e profili reali, quindi si eseguono solo nelle macchine virtuali con `go run ./vm test <macchina>`; l'elenco dei package è in `vm/commands.go`.

### 2026-11-18 — Gestione utenti locali (Fase 1)
- **Pagina unica** "Gestione utenti" (gruppo Utenti locali): elenco degli utenti locali e pulsanti Nuovo utente, Cambia password, Rimuovi password, Elimina utente, Aggiorna.
- Elenco: **tutti** gli account locali, compresi quelli di sistema e disattivati, con nome, nome completo, tipo (Amministratore/Utente) e stato.
- Creazione: nome utente, nome completo, password e conferma, tipo (Utente/Amministratore), più le opzioni **"La password non scade mai"** (attiva di default) e **"Cambia la password al primo accesso"**, che si escludono a vicenda (come in Windows).
- Eliminazione completa con **conferma digitando il nome dell'utente**, dopo un riepilogo di cosa verrà cancellato. Avviene nel componente di avanzamento: verifica, cancellazione della cartella del profilo con avanzamento file per file, rimozione delle voci di registro del profilo (`DeleteProfileW`), eliminazione dell'account (`NetUserDel`). L'account si elimina per ultimo, così un errore a metà permette di riprovare.
- Protezioni: non si elimina l'utente con cui si sta usando il programma, un utente con il profilo caricato (connesso) né l'ultimo amministratore attivo. La cartella del profilo si cancella a mano solo se è una sottocartella diretta della cartella dei profili (`ProfilesDirectory`); altrimenti se ne occupa solo `DeleteProfileW`. Le giunzioni e i collegamenti (reparse point) si rimuovono senza mai seguirli.
- Gruppi Administrators e Users individuati tramite SID noti (S-1-5-32-544, S-1-5-32-545), perché il nome cambia con la lingua di Windows.
- Cambio e rimozione della password con avviso: reimpostare la password da amministratore fa perdere all'utente l'accesso a file cifrati EFS, credenziali salvate e password dei browser protette da DPAPI.
- Chiamate NetAPI (`netapi32.dll`) e `userenv.dll` dichiarate nel package `internal/winapi`, perché la versione di `golang.org/x/sys` in uso non le espone.

### 2026-11-19 — Backup di file e impostazioni (Fase 2)
- **Formato del backup**: una cartella per backup (es. `Backup_<utente>_<data>`) con i file copiati così come sono (non compressi) e un `manifest.json`. Scelto per poter sfogliare e recuperare i file da Esplora risorse anche senza il programma, reggere file grandi e riprendere dopo un'interruzione.
- **Manifest** (`manifest.json`): contenuto del backup, versione del formato, data, utente sorgente, computer e voci incluse.
- **Destinazione**: una cartella qualsiasi, su disco interno, USB o percorso di rete `\\server\condivisione`; per le condivisioni che le richiedono si chiedono nome utente e password. Prima di iniziare si controlla lo spazio libero.
- **File dell'utente**, cartelle selezionabili:
  - cartelle standard con la dimensione accanto: Desktop, Documenti, Immagini, Musica, Video, Download, Preferiti;
  - cartelle qualsiasi aggiunte a mano, del profilo o del disco;
  - dischi fissi fisici diversi dal disco di sistema, per intero;
  - cartelle nella radice del disco di sistema che non sono cartelle di sistema (es. `C:\Dati`).
  - La copia prosegue dopo l'errore sul singolo file e riepiloga i file non copiati.
- **Browser**: nella prima versione Chrome, Edge (Chromium) e Firefox. Per ciascun browser e profilo si scelgono le singole voci: segnalibri, password, cronologia, estensioni, impostazioni. Cache e file temporanei sempre esclusi.
- **Password dei browser**: le password dei browser Chromium sono legate all'utente Windows. Si leggono nel contesto dell'utente sorgente e nel backup vengono protette da una **password del backup** scelta dal tecnico, per poterle ripristinare altrove nella Fase 3. Firefox usa un formato già portabile. Da verificare su Windows reale il comportamento con le versioni recenti di Chrome/Edge, che aggiungono una protezione a livello di sistema.
- **Utente sorgente senza password**: l'accesso ai suoi dati protetti è limitato da un criterio di Windows. Si sblocca temporaneamente quel criterio, si leggono le password e lo si **ripristina subito**; lo stato iniziale va ripristinato anche se il programma si interrompe a metà (controllo e recupero al riavvio).
- **Impostazioni di sistema** nella prima versione: reti Wi-Fi salvate con le password; stampanti dell'utente (con la predefinita) e unità di rete collegate; sfondo e aspetto (immagine e colore dello sfondo, tema chiaro/scuro, colore principale); lingua, layout di tastiera e formati di data, ora e numeri. Le altre voci restano per una versione successiva.

### 2026-11-19 — Recupero offline delle password dei browser (Windows non avviabile)
- **Scenario**: recuperare le password dei browser di un cliente quando il suo Windows **non si avvia**, da ambiente di ripristino. È uno scenario distinto dalla Fase 2 (che legge le password nel contesto dell'utente in esecuzione). Si fa **solo previa autorizzazione del cliente** e con la **sua password Windows**, che il cliente fornisce.
- **Dove gira (entrambi gli scenari, scelta del 2026-11-19)**:
  - disco del cliente **montato su un PC Windows funzionante** (adattatore USB o come secondo disco): `SupportTools.exe` gira normalmente e legge file e hive dal volume montato — è il metodo standard del data recovery;
  - dentro un ambiente **WinPE** avviato da USB sul PC del cliente: da verificare il funzionamento della GUI walk e delle API disponibili in WinPE.
- **Copertura (scelta del 2026-11-19)**: tutte le generazioni di cifratura dei browser Chromium, **incluso App-Bound Encryption (ABE, "v20")** di Chrome ≥127 ed Edge recenti.
- **Catena di decifratura offline** (dal disco montato, nessuna chiamata a Windows avviato):
  - *livello utente* (DPAPI diretto e chiave AES in `Local State`, prefissi `v10`/`v11`): servono `Login Data`, `Local State`, i masterkey DPAPI dell'utente (`…\Microsoft\Protect\<SID>\`) e la **password Windows dell'utente** per sbloccare il masterkey;
  - *livello ABE* (prefisso `v20`): la chiave app-bound è wrappata due volte, con DPAPI utente **e** DPAPI di macchina. Serve in più il masterkey **di macchina** (`…\Microsoft\Protect\S-1-5-18\`), che si sblocca con il LSA secret `DPAPI_SYSTEM`, ricavato dagli hive di registro `SYSTEM` (bootkey) e `SECURITY` presenti sul disco. Nessuna password utente per questo livello.
  - Offline la **path-validation** del servizio ABE non si applica (non si passa per il servizio SYSTEM): avendo entrambi i masterkey si ottiene la chiave app-bound e si decifra con AES-GCM. I vecchi strumenti "avvia e scarica" falliscono su Chrome 127+ perché estraggono solo blob cifrati.
- **Implementazione (scelta del 2026-11-19)**: catena DPAPI offline con una **libreria/port Go esistente**, non riscritta a mano.
  - **Valutazione librerie (2026-11-19)**:
    - [`wat4r/dpapitk`](https://pkg.go.dev/github.com/wat4r/dpapitk) — Apache-2.0 (compatibile GPL-3.0), `go 1.19` (**ok con Go 1.20**), unica dipendenza `golang.org/x/crypto`. Copre: decifratura masterkey con password/hash/PVK (chiave di backup di dominio), blob DPAPI e **DPAPI-NG/CNG**. **Non** copre: contesto SYSTEM/di macchina, LSA secret, parsing degli hive di registro, né Chrome/ABE. Progetto piccolo (ultimo commit dic 2024).
    - [`mandiant/gopacket`](https://pkg.go.dev/github.com/mandiant/gopacket/pkg/dpapi) (pkg `pkg/dpapi`, reimplementazione Go di Impacket) — Apache-2.0 (+ file `NOTICE` da riprodurre), molto più completa: masterkey offline, cronologia credenziali (hash NT), vault, chiave di backup di dominio/BKRP, `ParsePrivateKeyData` da **LSA secret**, e un package `pkg/registry` per gli **hive di registro**. **Ma richiede Go ≥1.24/1.25**: incompatibile con il vincolo Go 1.20 / Windows 7, e trascina molte dipendenze (LDAP/SMB/Kerberos/MSRPC). Nessun DPAPI-NG/CNG.
    - Nessuna delle due copre l'ABE/`v20` (atteso).
    - I wrapper della Windows API (`billgraziano/dpapi`, `jghiloni/dpapi`) **non servono** (richiedono Windows avviato).
  - **Scelta (2026-11-19): opzione A.** Si resta a **un solo binario** Go 1.20 / Windows 7. Si usa **`wat4r/dpapitk`** per i primitivi DPAPI/CNG (decifratura masterkey con password/hash/PVK, blob DPAPI e DPAPI-NG/CNG) e si **implementano in proprio** i pezzi mancanti, compatibili con Go 1.20:
    - parsing degli hive di registro `SYSTEM` e `SECURITY` dal disco → bootkey → LSA secret `DPAPI_SYSTEM` → sblocco del masterkey DPAPI **di macchina** (`…\S-1-5-18\`);
    - strato **ABE/`v20`**: scartare i due livelli DPAPI (utente + macchina) e AES-GCM finale.
    - Dove utile si **porta** il codice Apache-2.0 dei package `pkg/registry`/`pkg/dpapi` di `mandiant/gopacket` in un nostro package Go 1.20 (attribuzione Apache in `THIRD_PARTY_NOTICES.md`). Scartata l'opzione B (binario separato con Go recente) per non avere due toolchain, dipendenze pesanti e più avvisi di licenza.
    - L'avviso di licenza di `wat4r/dpapitk` (Apache-2.0) va aggiunto a `THIRD_PARTY_NOTICES.md` all'introduzione della dipendenza.
  - Prima di adottare una libreria vanno verificate la **compatibilità della licenza con GPL-3.0** e la **compatibilità con Go 1.20**, e il relativo avviso va aggiunto a `THIRD_PARTY_NOTICES.md`.
  - Lo **strato ABE (parsing del blob `v20`)** non ha una libreria Go pronta: va costruito sopra i primitivi DPAPI (scartare i due livelli + AES-GCM). Riferimenti (C++/Python) per il formato: progetti tipo `xaitax/Chrome-App-Bound-Encryption-Decryption`.
- **Firefox**: usa il proprio archivio NSS (`key4.db` + `logins.json`), già portabile e indipendente da DPAPI; si gestisce a parte.
- Lo strato di decodifica DPAPI/ABE è condivisibile con la Fase 2 (lettura nel contesto utente vivo): stessa logica di decifratura, sorgente dei masterkey diversa.
- **Lettura del database `Login Data` (SQLite) senza CGO (scelta 2026-11-19)**: libreria **`github.com/alicebob/sqlittle`** (MIT, compatibile GPL; `go 1.12`, verificata su Go 1.20). È in **sola lettura** e **non supporta i file WAL**: se accanto a `Login Data` c'è un `-wal` non ancora unito, le scritture recenti non si vedono → va **rilevato e segnalato** (o gestito a parte). Dipendenze indirette `x/exp`, `x/sys` (BSD).
- **Verifica dipendenze (2026-11-19)**: `wat4r/dpapitk@v1.0.1` compila e gira su **Go 1.20** (in modulo isolato); `go mod tidy` risolve `golang.org/x/crypto v0.17.0` (BSD). Gli avvisi di licenza (dpapitk Apache-2.0, sqlittle MIT + relative dipendenze) vanno aggiunti a `THIRD_PARTY_NOTICES.md` quando le dipendenze entrano nel `go.mod`.
- **Avanzamento codice (2026-11-19)**: avviato il package `internal/browserpw` (decodifica Chromium, cross-platform, testabile senza Windows). Fatto il primo tassello del livello utente, senza dipendenze esterne: `EncryptedKeyFromLocalState` (estrae dal file `Local State` il blob DPAPI della chiave AES) e `DecryptPassword` (AES-256-GCM per i valori `v10`/`v11`; i valori legacy senza prefisso restituiscono `ErrLegacyDPAPIValue`), con test unitari. Prossimo: sblocco DPAPI della chiave (dpapitk) e lettura di `Login Data` (sqlittle), da provare col profilo Chrome della VM win10.

### 2026-11-19 — Solo architettura x64 (abbandono del 32 bit)
- Il prodotto è **solo a 64 bit (x64)**. Motivazione (decisa con l'umano): le macchine a 32 bit sono ormai rare, non vale la pena mantenere il doppio target.
- Resta il supporto a **Windows 7 (x64)** e successivi, quindi resta il vincolo **Go 1.20** (ultima versione che genera eseguibili per Windows 7).
- Modifiche applicate:
  - `build`: rimosso il target `x86`/`386`; `-arch` accetta `all` o `x64`; output solo `dist/x64/`.
  - macchine di test: `win7` convertita a **x64** (`goarch amd64`, `unattendArch amd64`, 2 GB di RAM, disco 32 GB); l'ISO di Windows 7 deve essere **x64** e, se in inglese, va passato `-lang en-US` all'installazione.
  - `autounattend.xml` (Win7, ramo `else` del blocco `.Modern`): aggiunto `<SkipMachineOOBE>true</SkipMachineOOBE>`. Senza, l'installazione non presidiata di Windows 7 si bloccava sulla pagina OOBE "Type your product key" (Win7 Ultimate non ha una chiave generica; la pagina ha un pulsante *Skip* ma va saltata in automatico). Verificato il 2026-11-19: win10 e win7 si installano entrambe correttamente.
  - `scripts/setup-linux.sh`: niente più Wine a 32 bit (`wine32:i386`, `dpkg --add-architecture i386`); si installa solo `wine` (x64). Il rilevamento di un host Linux a 32 bit per scaricare Go resta (riguarda la macchina di sviluppo, non il target).
  - documenti (`MEMORY.md`, `ROADMAP.md`, `README.md`) aggiornati.
- L'icona "a 32 bit" nei commenti è la **profondità colore** (BGRA), non la CPU: invariata.

### 2026-11-19 — Macchine di test da un editor impacchettato come snap
- Lanciando `go run ./vm …` da un editor snap (es. VS Code snap), QEMU apre la finestra GTK con i moduli dello snap e trascina librerie incompatibili (la `libpthread` di `core20`), fallendo con `undefined symbol __libc_pthread_init`.
- Soluzione nel programma `vm` (`hostEnv` in `vm/util.go`): i processi QEMU vengono avviati con un ambiente ripulito dalle variabili che redirigono il caricamento dentro lo snap (`LD_LIBRARY_PATH` filtrato; `GTK_PATH`, `GIO_MODULE_DIR`, `GDK_PIXBUF_*`, `GSETTINGS_SCHEMA_DIR`, `LOCPATH`, ecc. rimosse). Gli snap stanno sia in `/snap/...` sia nei dati utente `~/snap/...`, quindi il riconoscimento cerca il segmento `/snap/` ovunque nel percorso.

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

- Nome e posizione predefiniti della cartella di backup e regole in caso di cartella già esistente.
- Come proteggere la password del backup nel `manifest.json` (algoritmo e verifica all'apertura in Fase 3).
