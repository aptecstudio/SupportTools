// SPDX-License-Identifier: GPL-3.0-or-later

//go:build windows

// Package ui contiene la finestra principale di Support Tools e i componenti
// dell'interfaccia condivisi dai tool.
package ui

import (
	"fmt"

	"github.com/lxn/walk"
	decl "github.com/lxn/walk/declarative"

	"github.com/aptecstudio/supporttools/internal/modules"
	"github.com/aptecstudio/supporttools/internal/version"
)

// AppName è il nome del programma mostrato all'utente.
const AppName = "Support Tools"

// iconResourceID è l'ID della risorsa dell'icona incorporata da rsrc, che
// assegna 1 al manifest e 2 al gruppo dell'icona.
const iconResourceID = 2

// sidebarWidth è la larghezza fissa della barra laterale, a 96 DPI.
const sidebarWidth = 230

// mainWindow è la finestra principale: barra laterale con i tool a sinistra,
// pagina del tool selezionato a destra.
type mainWindow struct {
	*walk.MainWindow
	tree    *walk.TreeView
	content *walk.Composite
	welcome *walk.Composite
	model   *sidebarModel
	current *walk.Composite
}

// Run crea la finestra principale con i tool registrati e resta in attesa
// finché l'utente non la chiude.
func Run() error {
	w := &mainWindow{model: newSidebarModel(modules.All())}

	err := decl.MainWindow{
		AssignTo: &w.MainWindow,
		Title:    AppName + " " + version.Version,
		MinSize:  decl.Size{Width: 760, Height: 480},
		Size:     decl.Size{Width: 1000, Height: 640},
		Layout:   decl.HBox{},
		Children: []decl.Widget{
			decl.TreeView{
				AssignTo:             &w.tree,
				Model:                w.model,
				MinSize:              decl.Size{Width: sidebarWidth},
				MaxSize:              decl.Size{Width: sidebarWidth},
				OnCurrentItemChanged: w.onCurrentItemChanged,
			},
			decl.Composite{
				AssignTo: &w.content,
				Layout:   decl.VBox{MarginsZero: true},
				Children: []decl.Widget{
					decl.Composite{
						AssignTo: &w.welcome,
						Layout:   decl.VBox{},
						Children: []decl.Widget{
							decl.Label{Text: AppName, Font: decl.Font{PointSize: 16, Bold: true}},
							decl.Label{Text: "Seleziona uno strumento dall'elenco a sinistra."},
							decl.VSpacer{},
						},
					},
				},
			},
		},
	}.Create()
	if err != nil {
		return fmt.Errorf("creazione della finestra principale: %w", err)
	}

	// Senza icona incorporata (file .ico assente al momento della build) la
	// finestra mantiene quella predefinita di Windows.
	if icon, err := walk.NewIconFromResourceId(iconResourceID); err == nil {
		if err := w.SetIcon(icon); err != nil {
			return fmt.Errorf("impostazione dell'icona: %w", err)
		}
	}

	for _, g := range w.model.roots {
		if err := w.tree.SetExpanded(g, true); err != nil {
			return fmt.Errorf("apertura del gruppo %q: %w", g.Text(), err)
		}
	}
	w.current = w.welcome

	w.MainWindow.Run()
	return nil
}

// onCurrentItemChanged mostra la pagina del tool selezionato; selezionando
// un gruppo torna alla pagina iniziale.
func (w *mainWindow) onCurrentItemChanged() {
	item, ok := w.tree.CurrentItem().(*moduleItem)
	if !ok {
		w.showPage(w.welcome)
		return
	}
	if item.page == nil {
		page, err := newModulePage(w.content, item.module)
		if err != nil {
			walk.MsgBox(w, AppName,
				fmt.Sprintf("Impossibile aprire %q:\n%v", item.module.Title, err),
				walk.MsgBoxIconError)
			w.showPage(w.welcome)
			return
		}
		item.page = page
	}
	w.showPage(item.page)
}

// showPage rende visibile page e nasconde quella mostrata in precedenza.
func (w *mainWindow) showPage(page *walk.Composite) {
	if page == w.current {
		return
	}
	if w.current != nil {
		w.current.SetVisible(false)
	}
	page.SetVisible(true)
	w.current = page
}

// newModulePage crea, nascosta, la pagina di un tool dentro parent.
func newModulePage(parent walk.Container, mod modules.Module) (*walk.Composite, error) {
	page, err := walk.NewComposite(parent)
	if err != nil {
		return nil, err
	}
	page.SetVisible(false)

	layout := walk.NewVBoxLayout()
	if err := layout.SetMargins(walk.Margins{}); err != nil {
		page.Dispose()
		return nil, err
	}
	if err := page.SetLayout(layout); err != nil {
		page.Dispose()
		return nil, err
	}
	if err := mod.Build(page); err != nil {
		page.Dispose()
		return nil, err
	}
	return page, nil
}
