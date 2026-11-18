// SPDX-License-Identifier: GPL-3.0-or-later

//go:build windows

// Package demo è un tool di prova che simula un'operazione lunga per
// verificare il componente di avanzamento. È incluso nell'eseguibile solo
// compilando con il tag "demo" (go run ./build -tags demo), mai nei rilasci.
package demo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/lxn/walk"
	decl "github.com/lxn/walk/declarative"

	"github.com/aptecstudio/supporttools/internal/modules"
	"github.com/aptecstudio/supporttools/internal/ui/progress"
)

func init() {
	modules.Register(modules.Module{
		Group: modules.GroupGeneral,
		Title: "Prova avanzamento",
		Order: 100,
		Build: build,
	})
}

func build(parent walk.Container) error {
	var owner *walk.Composite
	return decl.Composite{
		AssignTo: &owner,
		Layout:   decl.VBox{},
		Children: []decl.Widget{
			decl.Label{Text: "Prova del componente di avanzamento", Font: decl.Font{PointSize: 14, Bold: true}},
			decl.TextLabel{Text: "Simula un'operazione di circa 30 secondi divisa in passaggi. " +
				"Strumento presente solo nelle build di prova."},
			decl.Composite{
				Layout: decl.HBox{MarginsZero: true},
				Children: []decl.Widget{
					decl.PushButton{Text: "Simula operazione", OnClicked: func() { simulate(owner, false) }},
					decl.PushButton{Text: "Simula errore", OnClicked: func() { simulate(owner, true) }},
					decl.PushButton{Text: "Simula durata sconosciuta", OnClicked: func() { simulateIndeterminate(owner) }},
					decl.HSpacer{},
				},
			},
			decl.VSpacer{},
		},
	}.Create(decl.NewBuilder(parent))
}

var folders = []string{"Desktop", "Documenti", "Immagini", "Musica", "Video", "Download"}

func simulate(owner *walk.Composite, fail bool) {
	const filesPerFolder = 50
	err := progress.Run(owner.Form(), "Prova avanzamento", func(r *progress.Reporter) error {
		r.SetTotal(int64(len(folders) * filesPerFolder))
		for i, folder := range folders {
			r.Step("Copia della cartella " + folder)
			for f := 1; f <= filesPerFolder; f++ {
				if fail && i == 3 && f == 20 {
					return errors.New("accesso negato a " + folder + `\file020.dat`)
				}
				r.SetStatus(fmt.Sprintf("file%03d.dat", f))
				if err := sleep(r.Context(), 100*time.Millisecond); err != nil {
					return err
				}
				r.Add(1)
			}
		}
		return nil
	})
	report(owner, err)
}

func simulateIndeterminate(owner *walk.Composite) {
	err := progress.Run(owner.Form(), "Prova avanzamento", func(r *progress.Reporter) error {
		for _, step := range []string{"Ricerca dei profili", "Lettura del registro", "Analisi dei browser"} {
			r.Step(step)
			if err := sleep(r.Context(), 3*time.Second); err != nil {
				return err
			}
		}
		return nil
	})
	report(owner, err)
}

// sleep attende d o l'annullamento del contesto.
func sleep(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-t.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func report(owner *walk.Composite, err error) {
	msg := "Esito: completata."
	switch {
	case errors.Is(err, context.Canceled):
		msg = "Esito: annullata dall'utente."
	case err != nil:
		msg = "Esito: errore (" + err.Error() + ")."
	}
	walk.MsgBox(owner.Form(), "Prova avanzamento", msg, walk.MsgBoxIconInformation)
}
