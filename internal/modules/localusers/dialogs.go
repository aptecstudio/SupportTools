// SPDX-License-Identifier: GPL-3.0-or-later

//go:build windows

package localusers

import (
	"errors"
	"fmt"
	"strings"

	"github.com/lxn/walk"
	decl "github.com/lxn/walk/declarative"

	"github.com/aptecstudio/supporttools/internal/ui/progress"
	"github.com/aptecstudio/supporttools/internal/users"
)

// resetWarning spiega cosa perde un utente quando un amministratore gli
// reimposta o toglie la password.
const resetWarning = "Attenzione: reimpostando la password da qui, l'utente perde l'accesso " +
	"ai file cifrati con EFS, alle credenziali salvate in Windows e alle password " +
	"memorizzate nei browser. Se l'utente conosce la password attuale, è meglio che la " +
	"cambi lui stesso (Ctrl+Alt+Canc > Cambia password)."

func showError(owner walk.Form, title string, err error) {
	walk.MsgBox(owner, title, err.Error(), walk.MsgBoxIconError)
}

// runNewUserDialog mostra il dialog di creazione di un utente e restituisce
// true se l'utente è stato creato.
func runNewUserDialog(owner walk.Form) bool {
	const title = "Nuovo utente"
	var (
		dlg                      *walk.Dialog
		okButton, cancelButton   *walk.PushButton
		name, fullName           *walk.LineEdit
		password, confirm        *walk.LineEdit
		kind                     *walk.ComboBox
		neverExpires, mustChange *walk.CheckBox
	)

	onOK := func() {
		nu := users.NewUser{
			Name:                 name.Text(),
			FullName:             strings.TrimSpace(fullName.Text()),
			Password:             password.Text(),
			Admin:                kind.CurrentIndex() == 1,
			PasswordNeverExpires: neverExpires.Checked(),
			MustChangePassword:   mustChange.Checked(),
		}
		if err := users.ValidateUserName(nu.Name); err != nil {
			showError(dlg, title, err)
			name.SetFocus()
			return
		}
		if err := users.ValidatePasswords(nu.Password, confirm.Text()); err != nil {
			showError(dlg, title, err)
			confirm.SetFocus()
			return
		}
		if nu.Password == "" {
			answer := walk.MsgBox(dlg, title,
				"Non hai inserito una password: chiunque potrà accedere a questo account. Continuare?",
				walk.MsgBoxYesNo|walk.MsgBoxIconWarning|walk.MsgBoxDefButton2)
			if answer != walk.DlgCmdYes {
				password.SetFocus()
				return
			}
		}
		if err := users.Create(nu); err != nil {
			showError(dlg, title, err)
			return
		}
		dlg.Accept()
	}

	err := decl.Dialog{
		AssignTo:      &dlg,
		Title:         title,
		DefaultButton: &okButton,
		CancelButton:  &cancelButton,
		MinSize:       decl.Size{Width: 440},
		Layout:        decl.VBox{},
		Children: []decl.Widget{
			decl.Composite{
				Layout: decl.Grid{Columns: 2, MarginsZero: true},
				Children: []decl.Widget{
					decl.Label{Text: "Nome utente:"},
					decl.LineEdit{AssignTo: &name, MaxLength: users.MaxUserNameLength},
					decl.Label{Text: "Nome completo:"},
					decl.LineEdit{AssignTo: &fullName, CueBanner: "facoltativo"},
					decl.Label{Text: "Password:"},
					decl.LineEdit{AssignTo: &password, PasswordMode: true},
					decl.Label{Text: "Conferma password:"},
					decl.LineEdit{AssignTo: &confirm, PasswordMode: true},
					decl.Label{Text: "Tipo di account:"},
					decl.ComboBox{
						AssignTo:     &kind,
						Model:        []string{"Utente standard", "Amministratore"},
						CurrentIndex: 0,
					},
				},
			},
			decl.CheckBox{
				AssignTo: &neverExpires,
				Text:     "La password non scade mai",
				Checked:  true,
				OnCheckedChanged: func() {
					// Le due opzioni si escludono a vicenda, come in Windows.
					if neverExpires != nil && mustChange != nil && neverExpires.Checked() {
						mustChange.SetChecked(false)
					}
				},
			},
			decl.CheckBox{
				AssignTo: &mustChange,
				Text:     "Cambia la password al primo accesso",
				OnCheckedChanged: func() {
					if neverExpires != nil && mustChange != nil && mustChange.Checked() {
						neverExpires.SetChecked(false)
					}
				},
			},
			decl.VSpacer{Size: 8},
			buttonRow(&okButton, "Crea", onOK, &cancelButton, func() { dlg.Cancel() }),
		},
	}.Create(owner)
	if err != nil {
		showError(owner, title, err)
		return false
	}
	return dlg.Run() == walk.DlgCmdOK
}

// runPasswordDialog mostra il dialog di cambio password di u e restituisce
// true se la password è stata cambiata.
func runPasswordDialog(owner walk.Form, u users.User) bool {
	title := "Cambia password di " + u.Name
	var (
		dlg                    *walk.Dialog
		okButton, cancelButton *walk.PushButton
		password, confirm      *walk.LineEdit
	)

	onOK := func() {
		if err := users.ValidatePasswords(password.Text(), confirm.Text()); err != nil {
			showError(dlg, title, err)
			confirm.SetFocus()
			return
		}
		if password.Text() == "" {
			showError(dlg, title, errors.New("inserisci la nuova password; per togliere "+
				"la password usa \"Rimuovi password\""))
			password.SetFocus()
			return
		}
		if err := users.SetPassword(u.Name, password.Text()); err != nil {
			showError(dlg, title, err)
			return
		}
		walk.MsgBox(dlg, title, "Password cambiata.", walk.MsgBoxIconInformation)
		dlg.Accept()
	}

	err := decl.Dialog{
		AssignTo:      &dlg,
		Title:         title,
		DefaultButton: &okButton,
		CancelButton:  &cancelButton,
		MinSize:       decl.Size{Width: 440},
		Layout:        decl.VBox{},
		Children: []decl.Widget{
			decl.Composite{
				Layout: decl.Grid{Columns: 2, MarginsZero: true},
				Children: []decl.Widget{
					decl.Label{Text: "Nuova password:"},
					decl.LineEdit{AssignTo: &password, PasswordMode: true},
					decl.Label{Text: "Conferma password:"},
					decl.LineEdit{AssignTo: &confirm, PasswordMode: true},
				},
			},
			decl.TextLabel{Text: resetWarning},
			decl.VSpacer{Size: 8},
			buttonRow(&okButton, "Cambia password", onOK, &cancelButton, func() { dlg.Cancel() }),
		},
	}.Create(owner)
	if err != nil {
		showError(owner, title, err)
		return false
	}
	return dlg.Run() == walk.DlgCmdOK
}

// confirmRemovePassword chiede conferma e toglie la password di u.
// Restituisce true se la password è stata rimossa.
func confirmRemovePassword(owner walk.Form, u users.User) bool {
	title := "Rimuovi password di " + u.Name
	msg := fmt.Sprintf("Vuoi togliere la password all'utente %s? Chiunque potrà accedere "+
		"al suo account senza password.\r\n\r\n%s", u.Name, resetWarning)
	if walk.MsgBox(owner, title, msg,
		walk.MsgBoxYesNo|walk.MsgBoxIconWarning|walk.MsgBoxDefButton2) != walk.DlgCmdYes {
		return false
	}
	if err := users.RemovePassword(u.Name); err != nil {
		showError(owner, title, err)
		return false
	}
	walk.MsgBox(owner, title, "Password rimossa.", walk.MsgBoxIconInformation)
	return true
}

// runDelete verifica che u si possa eliminare, chiede conferma facendo
// digitare il nome ed esegue l'eliminazione con la finestra di avanzamento.
func runDelete(owner walk.Form, u users.User) {
	title := "Elimina utente " + u.Name
	plan, err := users.PlanDelete(u.Name)
	if err != nil {
		showError(owner, title, err)
		return
	}
	if !confirmDelete(owner, title, plan) {
		return
	}

	err = progress.Run(owner, title, func(r *progress.Reporter) error {
		return plan.Execute(r.Context(), r)
	})
	if err == nil {
		walk.MsgBox(owner, title, "L'utente "+u.Name+" è stato eliminato.", walk.MsgBoxIconInformation)
	}
}

// confirmDelete riepiloga cosa verrà cancellato e chiede di digitare il nome
// dell'utente. Restituisce true se l'utente ha confermato.
func confirmDelete(owner walk.Form, title string, plan *users.DeletePlan) bool {
	name := plan.User.Name
	var b strings.Builder
	b.WriteString("Verranno eliminati definitivamente:\r\n\r\n")
	account := "l'account " + name
	if plan.User.Admin {
		account += " (amministratore)"
	}
	b.WriteString("•  " + account + "\r\n")
	if plan.ProfilePath != "" {
		b.WriteString("•  la cartella del profilo " + plan.ProfilePath + " con tutti i suoi file " +
			"(Desktop, Documenti, Immagini, Download, impostazioni dei programmi…)\r\n")
		b.WriteString("•  le voci di registro del profilo\r\n")
		if !plan.ManualFolderDelete {
			b.WriteString("\r\nLa cartella non si trova nella posizione standard: la eliminerà " +
				"Windows, senza indicazione dell'avanzamento.\r\n")
		}
	} else {
		b.WriteString("\r\nL'utente non ha mai effettuato l'accesso: non c'è una cartella del profilo.\r\n")
	}
	b.WriteString("\r\nL'operazione non si può annullare: fai prima un backup dei dati da conservare.")

	var (
		dlg                        *walk.Dialog
		deleteButton, cancelButton *walk.PushButton
		typed                      *walk.LineEdit
	)
	err := decl.Dialog{
		AssignTo:     &dlg,
		Title:        title,
		CancelButton: &cancelButton,
		MinSize:      decl.Size{Width: 480},
		Layout:       decl.VBox{},
		Children: []decl.Widget{
			decl.TextLabel{Text: b.String()},
			decl.VSpacer{Size: 8},
			decl.Label{Text: "Per confermare digita il nome dell'utente (" + name + "):"},
			decl.LineEdit{
				AssignTo: &typed,
				OnTextChanged: func() {
					if deleteButton != nil {
						deleteButton.SetEnabled(strings.EqualFold(typed.Text(), name))
					}
				},
			},
			decl.VSpacer{Size: 8},
			buttonRow(&deleteButton, "Elimina", func() { dlg.Accept() },
				&cancelButton, func() { dlg.Cancel() }),
		},
	}.Create(owner)
	if err != nil {
		showError(owner, title, err)
		return false
	}
	deleteButton.SetEnabled(false)
	// L'azione distruttiva non deve partire con Invio per errore: il
	// pulsante predefinito è Annulla.
	dlg.SetDefaultButton(cancelButton)
	typed.SetFocus()
	return dlg.Run() == walk.DlgCmdOK
}

// buttonRow crea la riga dei pulsanti di conferma e annullamento, allineati
// a destra.
func buttonRow(ok **walk.PushButton, okText string, onOK walk.EventHandler,
	cancel **walk.PushButton, onCancel walk.EventHandler) decl.Composite {
	return decl.Composite{
		Layout: decl.HBox{MarginsZero: true},
		Children: []decl.Widget{
			decl.HSpacer{},
			decl.PushButton{AssignTo: ok, Text: okText, OnClicked: onOK},
			decl.PushButton{AssignTo: cancel, Text: "Annulla", OnClicked: onCancel},
		},
	}
}
