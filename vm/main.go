// SPDX-License-Identifier: GPL-3.0-or-later

// Il programma vm crea e usa le macchine virtuali Windows con cui provare
// Support Tools, tramite QEMU, su Linux e macOS.
//
// Va eseguito dalla radice del repository:
//
//	go run ./vm list
//	go run ./vm install <macchina> -iso <file.iso> [opzioni]
//	go run ./vm test <macchina> [-show]
//	go run ./vm screenshot <macchina> [-tags demo] [-wait 20s] [-show]
//	go run ./vm start <macchina> [-tags demo] [-persist]
//
// Le macchine disponibili sono win10 (Windows 10 x64) e win7 (Windows 7
// SP1 x64); i dischi stanno in ~/VM/SupportTools (o in SUPPORTTOOLS_VM_DIR).
//
// L'installazione è non presidiata: crea l'amministratore "tester", attiva
// l'accesso automatico e installa l'agente che esegue i comandi inviati
// dall'host. test, screenshot e start avviano la macchina con -snapshot di
// QEMU, quindi ogni esecuzione parte da Windows appena installato e tutte le
// modifiche vengono scartate allo spegnimento (salvo start -persist).
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"time"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "[x]", err)
		os.Exit(1)
	}
}

func usage() error {
	return errors.New("uso: go run ./vm list | install <macchina> -iso <file.iso> | " +
		"test <macchina> | screenshot <macchina> | start <macchina> (aggiungi -h per le opzioni)")
}

func run(args []string) error {
	if len(args) == 0 {
		return usage()
	}
	if _, err := os.Stat("go.mod"); err != nil {
		return errors.New("esegui il programma dalla radice del repository (go run ./vm …)")
	}
	command, args := args[0], args[1:]
	if command == "list" {
		return list()
	}
	if len(args) == 0 || len(args[0]) == 0 || args[0][0] == '-' {
		return usage()
	}
	m, err := lookupMachine(args[0])
	if err != nil {
		return err
	}
	fs := flag.NewFlagSet(command, flag.ExitOnError)
	allowTCG := fs.Bool("tcg", false, "usa l'emulazione software se manca l'accelerazione hardware (molto lenta)")

	switch command {
	case "install":
		var o installOptions
		fs.StringVar(&o.iso, "iso", "", "ISO di installazione di Windows")
		fs.StringVar(&o.lang, "lang", "it-IT", "lingua della ISO (es. it-IT, en-US)")
		fs.StringVar(&o.timeZone, "timezone", "W. Europe Standard Time", "fuso orario di Windows")
		fs.StringVar(&o.edition, "edition", "", "nome dell'edizione da installare (es. \"Windows 10 Pro\")")
		fs.IntVar(&o.index, "index", 0, "numero dell'edizione da installare nella ISO")
		fs.StringVar(&o.key, "key", "", "codice Product Key (\"-\" per non indicarlo)")
		fs.BoolVar(&o.headless, "headless", false, "non mostrare la finestra durante l'installazione")
		fs.BoolVar(&o.force, "force", false, "reinstalla una macchina già installata")
		fs.Parse(args[1:])
		o.allowTCG = *allowTCG
		return install(m, o)

	case "test":
		show := fs.Bool("show", false, "mostra la finestra della macchina")
		fs.Parse(args[1:])
		return runTests(m, sessionOptions{snapshot: true, show: *show, allowTCG: *allowTCG})

	case "screenshot":
		tags := fs.String("tags", "", "tag di build del programma (es. demo)")
		wait := fs.Duration("wait", 15*time.Second, "attesa tra l'avvio del programma e la schermata")
		show := fs.Bool("show", false, "mostra la finestra della macchina")
		fs.Parse(args[1:])
		return screenshot(m, *tags, *wait, sessionOptions{snapshot: true, show: *show, allowTCG: *allowTCG})

	case "start":
		tags := fs.String("tags", "", "tag di build del programma (es. demo)")
		persist := fs.Bool("persist", false, "conserva le modifiche al disco allo spegnimento")
		fs.Parse(args[1:])
		return start(m, *tags, sessionOptions{snapshot: !*persist, show: true, allowTCG: *allowTCG})
	}
	return usage()
}

func list() error {
	root, err := vmRoot()
	if err != nil {
		return err
	}
	fmt.Printf("Macchine in %s:\n", root)
	for _, name := range machineNames() {
		m := machines[name]
		state := "non installata"
		if m.installed() {
			state = "installata"
		}
		fmt.Printf("  %-6s %-20s %s\n", m.name, m.title, state)
	}
	return nil
}
