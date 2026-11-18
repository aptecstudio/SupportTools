// SPDX-License-Identifier: GPL-3.0-or-later

//go:build windows

// Package about è il tool "Informazioni": versione, licenza e collegamento
// al repository dei sorgenti.
package about

import (
	"syscall"

	"github.com/lxn/walk"
	decl "github.com/lxn/walk/declarative"
	"github.com/lxn/win"

	"github.com/aptecstudio/supporttools/internal/modules"
	"github.com/aptecstudio/supporttools/internal/version"
)

// RepositoryURL è l'indirizzo del repository con sorgenti e rilasci.
const RepositoryURL = "https://github.com/aptecstudio/SupportTools"

const licenseNotice = "Support Tools è software libero: puoi redistribuirlo e/o " +
	"modificarlo secondo i termini della GNU General Public License, versione 3 " +
	"o (a tua scelta) qualsiasi versione successiva.\r\n\r\n" +
	"È distribuito nella speranza che sia utile, ma SENZA ALCUNA GARANZIA, " +
	"nemmeno la garanzia implicita di COMMERCIABILITÀ o di IDONEITÀ A UN " +
	"PARTICOLARE SCOPO. I dettagli sono nel file LICENSE distribuito con il " +
	"programma; gli avvisi dei componenti di terze parti sono in " +
	"THIRD_PARTY_NOTICES.md."

func init() {
	modules.Register(modules.Module{
		Group: modules.GroupGeneral,
		Title: "Informazioni",
		Build: build,
	})
}

func build(parent walk.Container) error {
	var owner *walk.Composite
	return decl.Composite{
		AssignTo: &owner,
		Layout:   decl.VBox{},
		Children: []decl.Widget{
			decl.Label{Text: "Support Tools", Font: decl.Font{PointSize: 16, Bold: true}},
			decl.Label{Text: "Versione " + version.Version},
			decl.Label{Text: "Tool di supporto tecnico per Windows."},
			decl.VSpacer{Size: 12},
			decl.TextLabel{Text: licenseNotice},
			decl.VSpacer{Size: 12},
			decl.LinkLabel{
				Text: `Sorgenti e rilasci: <a href="` + RepositoryURL + `">` + RepositoryURL + `</a>`,
				OnLinkActivated: func(link *walk.LinkLabelLink) {
					openURL(owner, link.URL())
				},
			},
			decl.VSpacer{},
		},
	}.Create(decl.NewBuilder(parent))
}

// openURL apre url nel browser predefinito.
func openURL(owner *walk.Composite, url string) {
	verb, _ := syscall.UTF16PtrFromString("open")
	file, err := syscall.UTF16PtrFromString(url)
	if err != nil {
		return
	}
	if !win.ShellExecute(owner.Handle(), verb, file, nil, nil, win.SW_SHOWNORMAL) {
		walk.MsgBox(owner.Form(), "Support Tools", "Impossibile aprire "+url, walk.MsgBoxIconError)
	}
}
