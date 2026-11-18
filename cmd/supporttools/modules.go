// SPDX-License-Identifier: GPL-3.0-or-later

//go:build windows

package main

// Tool inclusi nel programma: ogni package si registra da solo nella sua
// funzione init. Per aggiungere un tool basta aggiungerne qui l'import.
import (
	_ "github.com/aptecstudio/supporttools/internal/modules/about"
)
