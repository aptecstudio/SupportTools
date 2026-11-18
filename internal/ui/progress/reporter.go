// SPDX-License-Identifier: GPL-3.0-or-later

// Package progress è il componente condiviso per le operazioni lunghe.
//
// Run apre una finestra con barra di avanzamento, percentuale, tempo
// rimanente stimato e una sezione "Dettagli" espandibile con l'elenco delle
// operazioni concluse e di quella in lavorazione. L'operazione gira in una
// goroutine separata e comunica l'avanzamento tramite un *Reporter.
//
// Esempio:
//
//	err := progress.Run(owner, "Backup", func(r *progress.Reporter) error {
//		r.SetTotal(totalBytes)
//		for _, dir := range dirs {
//			r.Step("Copia di " + dir.Name)
//			if err := copyDir(r.Context(), dir, r.Add); err != nil {
//				return err
//			}
//		}
//		return nil
//	})
//
// Le operazioni elencate nei dettagli sono passaggi di alto livello (una
// cartella, un utente, un browser), non i singoli file: per mostrare il file
// in lavorazione si usa SetStatus.
package progress

import (
	"context"
	"sync"
)

// StepState è lo stato di un'operazione elencata nei dettagli.
type StepState int

// Stati possibili di un'operazione.
const (
	StepRunning StepState = iota
	StepDone
	StepFailed
	StepCanceled
)

// Step è un'operazione elencata nei dettagli.
type Step struct {
	Name  string
	State StepState
}

// String restituisce la riga mostrata nei dettagli.
func (s Step) String() string {
	switch s.State {
	case StepRunning:
		return "In corso – " + s.Name
	case StepDone:
		return "Completata – " + s.Name
	case StepFailed:
		return "Non riuscita – " + s.Name
	case StepCanceled:
		return "Annullata – " + s.Name
	default:
		return s.Name
	}
}

// Reporter riceve l'avanzamento dall'operazione in corso. I suoi metodi si
// possono chiamare da qualsiasi goroutine.
type Reporter struct {
	ctx context.Context

	mu     sync.Mutex
	total  int64
	done   int64
	status string
	steps  []Step
	// stepsRev cambia a ogni modifica di steps, per aggiornare l'elenco dei
	// dettagli solo quando serve.
	stepsRev uint64
}

func newReporter(ctx context.Context) *Reporter {
	return &Reporter{ctx: ctx}
}

// Context restituisce il contesto dell'operazione, annullato quando l'utente
// preme "Annulla". Le operazioni lunghe devono controllarlo periodicamente e
// restituire ctx.Err() appena viene annullato.
func (r *Reporter) Context() context.Context {
	return r.ctx
}

// SetTotal imposta la quantità totale di lavoro (byte, file, passaggi: l'unità
// la sceglie l'operazione). Con totale zero la barra resta indeterminata.
func (r *Reporter) SetTotal(total int64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if total < 0 {
		total = 0
	}
	r.total = total
}

// Add aggiunge n unità di lavoro completate.
func (r *Reporter) Add(n int64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.done += n
}

// SetDone imposta il lavoro completato fino a ora.
func (r *Reporter) SetDone(done int64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.done = done
}

// Step inizia una nuova operazione, mostrata nei dettagli come "in corso";
// l'operazione precedente, se ancora in corso, diventa completata.
func (r *Reporter) Step(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.closeRunning(StepDone)
	r.steps = append(r.steps, Step{Name: name, State: StepRunning})
	r.status = ""
	r.stepsRev++
}

// SetStatus imposta un testo aggiuntivo sull'operazione in corso (per esempio
// il file in copia). Non compare nei dettagli e si azzera a ogni Step.
func (r *Reporter) SetStatus(text string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.status = text
}

// finish chiude l'operazione in corso con lo stato indicato.
func (r *Reporter) finish(state StepState) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.closeRunning(state)
	r.status = ""
	r.stepsRev++
}

// closeRunning va chiamata con r.mu bloccato.
func (r *Reporter) closeRunning(state StepState) {
	if n := len(r.steps); n > 0 && r.steps[n-1].State == StepRunning {
		r.steps[n-1].State = state
	}
}

// snapshot è una copia coerente dello stato del Reporter.
type snapshot struct {
	total, done int64
	status      string
	current     string // nome dell'operazione in corso, se c'è
	steps       []Step // valorizzato solo se stepsRev è diverso da quello chiesto
	stepsRev    uint64
}

// snapshot copia lo stato; l'elenco delle operazioni viene copiato solo se è
// cambiato rispetto a knownRev.
func (r *Reporter) snapshot(knownRev uint64) snapshot {
	r.mu.Lock()
	defer r.mu.Unlock()
	s := snapshot{total: r.total, done: r.done, status: r.status, stepsRev: r.stepsRev}
	if n := len(r.steps); n > 0 && r.steps[n-1].State == StepRunning {
		s.current = r.steps[n-1].Name
	}
	if r.stepsRev != knownRev {
		s.steps = append([]Step(nil), r.steps...)
	}
	return s
}
