// SPDX-License-Identifier: GPL-3.0-or-later

//go:build windows

package progress

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/lxn/walk"
	decl "github.com/lxn/walk/declarative"
)

const (
	// refreshInterval è ogni quanto la finestra legge l'avanzamento: gli
	// aggiornamenti non dipendono da quanto spesso l'operazione li invia.
	refreshInterval = 200 * time.Millisecond
	// detailsHeight è l'altezza aggiunta alla finestra quando si aprono i
	// dettagli, a 96 DPI.
	detailsHeight = 220
)

// Run apre una finestra modale di avanzamento con titolo title, esegue op in
// una goroutine separata e attende che l'utente chiuda la finestra.
//
// Restituisce l'errore di op; se l'utente ha annullato e op ha restituito
// l'errore del contesto, il risultato soddisfa errors.Is(err, context.Canceled).
// Un panic in op viene convertito in errore, senza chiudere il programma.
func Run(owner walk.Form, title string, op func(r *Reporter) error) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	d := &dialog{
		reporter: newReporter(ctx),
		cancel:   cancel,
		model:    new(stepsModel),
		running:  true,
		stop:     make(chan struct{}),
	}

	err := decl.Dialog{
		AssignTo:     &d.dlg,
		Title:        title,
		MinSize:      decl.Size{Width: 480},
		Size:         decl.Size{Width: 520},
		Layout:       decl.VBox{},
		CancelButton: &d.button,
		Children: []decl.Widget{
			decl.Label{AssignTo: &d.status, Text: "Preparazione…"},
			decl.ProgressBar{AssignTo: &d.bar, MaxValue: barMax, MarqueeMode: true},
			decl.Composite{
				Layout: decl.HBox{MarginsZero: true},
				Children: []decl.Widget{
					decl.Label{AssignTo: &d.percent},
					decl.HSpacer{},
					decl.Label{AssignTo: &d.remaining},
				},
			},
			decl.Composite{
				Layout: decl.HBox{MarginsZero: true},
				Children: []decl.Widget{
					decl.PushButton{
						AssignTo:  &d.toggle,
						Text:      "Mostra dettagli",
						OnClicked: func() { d.setDetailsVisible(!d.list.Visible()) },
					},
					decl.HSpacer{},
				},
			},
			decl.ListBox{
				AssignTo: &d.list,
				Model:    d.model,
				Visible:  false,
				MinSize:  decl.Size{Height: detailsHeight - 20},
			},
			decl.Composite{
				Layout: decl.HBox{MarginsZero: true},
				Children: []decl.Widget{
					decl.HSpacer{},
					decl.PushButton{AssignTo: &d.button, Text: "Annulla", OnClicked: d.onButton},
				},
			},
		},
	}.Create(owner)
	if err != nil {
		return fmt.Errorf("creazione della finestra di avanzamento: %w", err)
	}

	// Mentre l'operazione è in corso la finestra non si chiude: la X e il
	// tasto Esc chiedono l'annullamento.
	d.dlg.Closing().Attach(func(canceled *bool, _ walk.CloseReason) {
		if d.running {
			*canceled = true
			d.requestCancel()
		}
	})
	d.dlg.Starting().Attach(func() { d.start(op) })

	d.dlg.Run()
	d.closed = true
	return d.err
}

// dialog è lo stato della finestra di avanzamento. Tutti i campi, tranne
// reporter, si usano solo dal thread dell'interfaccia.
type dialog struct {
	dlg       *walk.Dialog
	status    *walk.Label
	bar       *walk.ProgressBar
	percent   *walk.Label
	remaining *walk.Label
	toggle    *walk.PushButton
	list      *walk.ListBox
	button    *walk.PushButton

	reporter *Reporter
	cancel   context.CancelFunc
	model    *stepsModel
	stepsRev uint64
	started  time.Time
	marquee  bool

	running   bool
	canceling bool
	closed    bool
	stop      chan struct{}
	err       error
}

// start avvia l'operazione e l'aggiornamento periodico della finestra.
func (d *dialog) start(op func(r *Reporter) error) {
	d.started = time.Now()
	d.marquee = true

	go func() {
		ticker := time.NewTicker(refreshInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				d.dlg.Synchronize(d.refresh)
			case <-d.stop:
				return
			}
		}
	}()

	go func() {
		err := safeRun(op, d.reporter)
		d.dlg.Synchronize(func() { d.finish(err) })
	}()
}

// safeRun esegue op convertendo un eventuale panic in errore.
func safeRun(op func(r *Reporter) error, r *Reporter) (err error) {
	defer func() {
		if p := recover(); p != nil {
			err = fmt.Errorf("errore interno: %v", p)
		}
	}()
	return op(r)
}

// refresh aggiorna la finestra con lo stato attuale del Reporter.
func (d *dialog) refresh() {
	if d.closed {
		return
	}
	s := d.reporter.snapshot(d.stepsRev)

	if s.total > 0 {
		d.setMarquee(false)
		frac := fraction(s.done, s.total)
		d.bar.SetValue(int(frac * barMax))
		d.percent.SetText(fmt.Sprintf("%d%%", int(frac*100)))
		if rem, ok := estimateRemaining(time.Since(d.started), frac); ok {
			d.remaining.SetText("Tempo rimanente: circa " + formatRemaining(rem))
		} else {
			d.remaining.SetText("Calcolo del tempo rimanente…")
		}
	} else {
		d.setMarquee(true)
		d.percent.SetText("")
		d.remaining.SetText("")
	}

	if d.running && !d.canceling {
		text := "Preparazione…"
		if s.current != "" {
			text = s.current
			if s.status != "" {
				text += ": " + s.status
			}
		}
		d.status.SetText(text)
	}

	if s.steps != nil {
		d.stepsRev = s.stepsRev
		d.model.update(s.steps)
		if n := len(s.steps); n > 0 {
			d.list.EnsureItemVisible(n - 1)
		}
	}
}

func (d *dialog) setMarquee(on bool) {
	if d.marquee == on {
		return
	}
	d.marquee = on
	d.bar.SetMarqueeMode(on)
	if !on {
		// Uscendo dalla modalità indeterminata il controllo perde il range.
		d.bar.SetRange(0, barMax)
	}
}

// finish mostra l'esito dell'operazione e abilita la chiusura.
func (d *dialog) finish(err error) {
	if d.closed {
		return
	}
	d.running = false
	close(d.stop)
	d.err = err

	canceled := err != nil && (errors.Is(err, context.Canceled) || d.reporter.Context().Err() != nil)
	switch {
	case err == nil:
		d.reporter.finish(StepDone)
	case canceled:
		d.reporter.finish(StepCanceled)
	default:
		d.reporter.finish(StepFailed)
	}
	d.refresh()

	switch {
	case err == nil:
		d.setMarquee(false)
		d.bar.SetValue(barMax)
		d.percent.SetText("100%")
		d.status.SetText("Operazione completata.")
	case canceled:
		d.status.SetText("Operazione annullata.")
	default:
		d.status.SetText("Operazione non riuscita: " + err.Error())
		d.setDetailsVisible(true)
	}
	if err != nil {
		d.setMarquee(false)
	}
	d.remaining.SetText("")

	d.button.SetText("Chiudi")
	d.button.SetEnabled(true)
	d.dlg.SetDefaultButton(d.button)
	d.button.SetFocus()
}

// onButton annulla l'operazione in corso o, a operazione finita, chiude.
func (d *dialog) onButton() {
	if d.running {
		d.requestCancel()
		return
	}
	d.dlg.Accept()
}

// requestCancel chiede conferma e annulla l'operazione. La finestra resta
// aperta finché l'operazione non si ferma davvero.
func (d *dialog) requestCancel() {
	if d.canceling {
		return
	}
	answer := walk.MsgBox(d.dlg, d.dlg.Title(), "Vuoi annullare l'operazione in corso?",
		walk.MsgBoxYesNo|walk.MsgBoxIconQuestion|walk.MsgBoxDefButton2)
	// L'operazione può essere finita mentre la domanda era aperta.
	if answer != walk.DlgCmdYes || !d.running {
		return
	}
	d.canceling = true
	d.cancel()
	d.button.SetEnabled(false)
	d.status.SetText("Annullamento in corso…")
}

// setDetailsVisible mostra o nasconde i dettagli, ridimensionando la finestra.
func (d *dialog) setDetailsVisible(visible bool) {
	if d.list.Visible() == visible {
		return
	}
	size := d.dlg.Size()
	if visible {
		size.Height += detailsHeight
		d.toggle.SetText("Nascondi dettagli")
	} else {
		size.Height -= detailsHeight
		d.toggle.SetText("Mostra dettagli")
	}
	d.list.SetVisible(visible)
	d.dlg.SetSize(size)
}

// stepsModel è il modello dell'elenco dei dettagli.
type stepsModel struct {
	walk.ListModelBase
	steps []Step
}

func (m *stepsModel) ItemCount() int {
	return len(m.steps)
}

func (m *stepsModel) Value(i int) interface{} {
	return m.steps[i].String()
}

// update sostituisce l'elenco notificando solo le righe cambiate: le
// operazioni si aggiungono in coda e cambia stato solo l'ultima.
func (m *stepsModel) update(steps []Step) {
	old := m.steps
	m.steps = steps
	if len(steps) < len(old) {
		m.PublishItemsReset()
		return
	}
	for i := range old {
		if old[i] != steps[i] {
			m.PublishItemChanged(i)
		}
	}
	if len(steps) > len(old) {
		m.PublishItemsInserted(len(old), len(steps)-1)
	}
}
