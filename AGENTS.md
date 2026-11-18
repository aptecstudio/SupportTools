# AGENTS.md

Istruzioni vincolanti per qualsiasi agente AI che lavora sul progetto **Support Tools**.

## 1. Protezione di questo file

- L'agente AI **non deve modificare** `AGENTS.md` senza previa autorizzazione o esplicito consenso dell'umano.
- Se ritiene utile una modifica, l'agente la propone all'umano e attende il consenso prima di applicarla.

## 2. File di riferimento

| File | Contenuto | Chi lo aggiorna |
|------|-----------|-----------------|
| `ROADMAP.md` | Roadmap del progetto: funzionalità, fasi e stato di avanzamento (spuntato). | L'agente, a ogni avanzamento concreto. |
| `MEMORY.md` | Scelte tecniche, decisioni e convenzioni adottate. | L'agente, ogni volta che viene presa o cambiata una decisione. |
| `CHANGELOG.md` | Storico delle versioni rilasciate. | L'agente, a ogni rilascio. |

- Prima di iniziare un lavoro l'agente legge `ROADMAP.md` e `MEMORY.md`.
- Ogni decisione presa con l'umano va registrata in `MEMORY.md`; ogni attività completata va spuntata in `ROADMAP.md`.

## 3. Lingua

- I file `.md` e le conversazioni con gli umani sono **esclusivamente in italiano**.
- Il codice sorgente e tutte le nomenclature (package, file, tipi, funzioni, variabili, costanti, identificatori, chiavi di configurazione) sono **in inglese**.
- **Eccezione:** i commenti nel codice sono in **italiano**.
- I testi mostrati all'utente finale nell'interfaccia seguono quanto deciso in `MEMORY.md`.

## 4. Stile del codice

- Lo stile di ogni linguaggio deve rispecchiare lo standard ufficiale di quel linguaggio (per Go: `gofmt`/`goimports`, [Effective Go](https://go.dev/doc/effective_go), [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments)).
- Vanno rispettati: commenti (in italiano, secondo le convenzioni del linguaggio, es. godoc per gli identificatori esportati), spaziature, indentazione, righe vuote, nuove linee (fine file con newline, terminatori di riga LF) e punteggiatura.
- Nessun codice viene committato senza essere formattato e senza superare `go vet`.

## 5. Dubbi e domande

- Quando l'agente AI ha dubbi **deve chiedere all'utente tramite sondaggi interattivi** (domande a scelta multipla) **in italiano**.
- L'agente non prende decisioni architetturali o di prodotto rilevanti senza conferma dell'umano.

## 6. Git e versioning

- Si lavora **solo sul branch `main`**: niente branch secondari, salvo diversa indicazione dell'umano.
- Commit e push si fanno su richiesta dell'umano.
- Il versioning segue [Semantic Versioning](https://semver.org/lang/it/) (`MAJOR.MINOR.PATCH`) e viene gestito dall'agente a ogni commit/push di un rilascio: aggiornamento del file `VERSION`, di `CHANGELOG.md` e creazione del tag Git `vX.Y.Z`.
- La **prima versione** (`v0.1.0`) si rilascia solo quando il programma ha un primo funzionamento concreto. Fino ad allora non si creano tag.
- Il formato dei messaggi di commit è definito in `MEMORY.md`.
