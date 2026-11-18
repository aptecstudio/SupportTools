// SPDX-License-Identifier: GPL-3.0-or-later

//go:build windows

// Il comando supporttools avvia Support Tools, il tool di supporto tecnico
// per Windows.
//
// Va compilato con il programma di build (go run ./build), che incorpora il
// manifest con i privilegi di amministratore e gli stili visuali.
package main

import (
	"os"

	"github.com/lxn/walk"

	"github.com/aptecstudio/supporttools/internal/ui"
)

func main() {
	if err := ui.Run(); err != nil {
		walk.MsgBox(nil, ui.AppName, "Impossibile avviare il programma:\r\n"+err.Error(),
			walk.MsgBoxIconError)
		os.Exit(1)
	}
}
