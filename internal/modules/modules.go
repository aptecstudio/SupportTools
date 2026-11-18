// SPDX-License-Identifier: GPL-3.0-or-later

//go:build windows

// Package modules è il registro dei tool di Support Tools.
//
// Ogni tool vive in un proprio sottopackage e si registra con Register in una
// funzione init. La finestra principale mostra i tool registrati nella barra
// laterale, raggruppati per Group. Per aggiungere un tool basta importare il
// suo package in cmd/supporttools/modules.go.
package modules

import (
	"fmt"
	"sort"

	"github.com/lxn/walk"
)

// Group è l'area funzionale a cui appartiene un tool. L'ordine delle costanti
// è l'ordine con cui i gruppi compaiono nella barra laterale.
type Group int

// Gruppi disponibili.
const (
	GroupUsers Group = iota
	GroupBackup
	GroupRestore
	GroupGeneral
)

// Title restituisce il nome del gruppo mostrato all'utente.
func (g Group) Title() string {
	switch g {
	case GroupUsers:
		return "Utenti locali"
	case GroupBackup:
		return "Backup"
	case GroupRestore:
		return "Ripristino"
	case GroupGeneral:
		return "Generale"
	default:
		return fmt.Sprintf("Gruppo %d", int(g))
	}
}

// Module descrive un tool mostrato nella barra laterale.
type Module struct {
	// Group è il gruppo in cui compare il tool.
	Group Group
	// Title è il nome del tool nella barra laterale.
	Title string
	// Order stabilisce la posizione del tool all'interno del gruppo
	// (valori più bassi vengono prima; a parità si ordina per titolo).
	Order int
	// Build crea l'interfaccia del tool dentro parent. Viene chiamata una
	// sola volta, la prima volta che l'utente apre il tool; parent ha già un
	// layout verticale senza margini.
	Build func(parent walk.Container) error
}

var registry []Module

// Register aggiunge un tool al registro. Va chiamata solo da funzioni init:
// il registro non è protetto per l'accesso concorrente.
func Register(m Module) {
	if m.Title == "" {
		panic("modules: tool senza titolo")
	}
	if m.Build == nil {
		panic(fmt.Sprintf("modules: il tool %q non ha una funzione Build", m.Title))
	}
	registry = append(registry, m)
}

// All restituisce i tool registrati, ordinati per gruppo, ordine e titolo.
func All() []Module {
	mods := make([]Module, len(registry))
	copy(mods, registry)
	sort.SliceStable(mods, func(i, j int) bool {
		a, b := mods[i], mods[j]
		if a.Group != b.Group {
			return a.Group < b.Group
		}
		if a.Order != b.Order {
			return a.Order < b.Order
		}
		return a.Title < b.Title
	})
	return mods
}
