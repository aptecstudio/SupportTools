// SPDX-License-Identifier: GPL-3.0-or-later

//go:build windows

// Package localusers è il tool "Gestione utenti": elenco degli account
// locali, creazione, cambio e rimozione della password, eliminazione
// completa con il profilo.
package localusers

import (
	"github.com/lxn/walk"
	decl "github.com/lxn/walk/declarative"

	"github.com/aptecstudio/supporttools/internal/modules"
	"github.com/aptecstudio/supporttools/internal/users"
)

func init() {
	modules.Register(modules.Module{
		Group: modules.GroupUsers,
		Title: "Gestione utenti",
		Build: build,
	})
}

// page è lo stato della pagina "Gestione utenti".
type page struct {
	root    *walk.Composite
	table   *walk.TableView
	model   *userModel
	message *walk.Label

	// Pulsanti che richiedono un utente selezionato.
	selectionButtons []*walk.PushButton
}

func build(parent walk.Container) error {
	p := &page{model: new(userModel)}
	var changePwd, removePwd, del *walk.PushButton

	err := decl.Composite{
		AssignTo: &p.root,
		Layout:   decl.VBox{},
		Children: []decl.Widget{
			decl.Label{Text: "Gestione utenti", Font: decl.Font{PointSize: 14, Bold: true}},
			decl.Label{Text: "Account locali di questo computer."},
			decl.TableView{
				AssignTo:            &p.table,
				Model:               p.model,
				AlternatingRowBG:    true,
				LastColumnStretched: true,
				Columns: []decl.TableViewColumn{
					{Title: "Nome utente", Width: 160},
					{Title: "Nome completo", Width: 200},
					{Title: "Tipo", Width: 120},
					{Title: "Stato", Width: 160},
				},
				OnCurrentIndexChanged: p.updateButtons,
			},
			decl.Label{AssignTo: &p.message, Visible: false, TextColor: walk.RGB(0xC0, 0, 0)},
			decl.Composite{
				Layout: decl.HBox{MarginsZero: true},
				Children: []decl.Widget{
					decl.PushButton{Text: "Nuovo utente…", OnClicked: p.create},
					decl.PushButton{AssignTo: &changePwd, Text: "Cambia password…", OnClicked: p.changePassword},
					decl.PushButton{AssignTo: &removePwd, Text: "Rimuovi password…", OnClicked: p.removePassword},
					decl.PushButton{AssignTo: &del, Text: "Elimina utente…", OnClicked: p.delete},
					decl.HSpacer{},
					decl.PushButton{Text: "Aggiorna", OnClicked: p.refresh},
				},
			},
		},
	}.Create(decl.NewBuilder(parent))
	if err != nil {
		return err
	}
	p.selectionButtons = []*walk.PushButton{changePwd, removePwd, del}
	p.refresh()
	return nil
}

// refresh rilegge l'elenco degli utenti mantenendo, se possibile, la selezione.
func (p *page) refresh() {
	selected := ""
	if u, ok := p.selected(); ok {
		selected = u.Name
	}

	list, err := users.List()
	if err != nil {
		p.message.SetText("Impossibile leggere gli utenti: " + err.Error())
		p.message.SetVisible(true)
		list = nil
	} else {
		p.message.SetVisible(false)
	}
	p.model.setUsers(list)

	for i, u := range list {
		if u.Name == selected {
			p.table.SetCurrentIndex(i)
			break
		}
	}
	p.updateButtons()
}

// selected restituisce l'utente selezionato nella tabella.
func (p *page) selected() (users.User, bool) {
	i := p.table.CurrentIndex()
	if i < 0 || i >= len(p.model.users) {
		return users.User{}, false
	}
	return p.model.users[i], true
}

func (p *page) updateButtons() {
	_, ok := p.selected()
	for _, b := range p.selectionButtons {
		b.SetEnabled(ok)
	}
}

func (p *page) create() {
	if runNewUserDialog(p.root.Form()) {
		p.refresh()
	}
}

func (p *page) changePassword() {
	u, ok := p.selected()
	if !ok {
		return
	}
	if runPasswordDialog(p.root.Form(), u) {
		p.refresh()
	}
}

func (p *page) removePassword() {
	u, ok := p.selected()
	if !ok {
		return
	}
	if confirmRemovePassword(p.root.Form(), u) {
		p.refresh()
	}
}

func (p *page) delete() {
	u, ok := p.selected()
	if !ok {
		return
	}
	runDelete(p.root.Form(), u)
	p.refresh()
}

// userModel è il modello della tabella degli utenti.
type userModel struct {
	walk.TableModelBase
	users []users.User
}

func (m *userModel) setUsers(list []users.User) {
	m.users = list
	m.PublishRowsReset()
}

func (m *userModel) RowCount() int {
	return len(m.users)
}

func (m *userModel) Value(row, col int) interface{} {
	u := m.users[row]
	switch col {
	case 0:
		return u.Name
	case 1:
		return u.FullName
	case 2:
		if u.Admin {
			return "Amministratore"
		}
		return "Utente standard"
	case 3:
		state := "Attivo"
		if u.Disabled {
			state = "Disattivato"
		}
		if u.Current {
			state += " (in uso)"
		}
		return state
	}
	return nil
}
