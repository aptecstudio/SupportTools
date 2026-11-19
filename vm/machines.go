// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// machine descrive una macchina virtuale di test.
type machine struct {
	name  string
	title string
	// goarch è l'architettura degli eseguibili da provare (amd64).
	goarch string
	// unattendArch è l'architettura nei componenti di autounattend.xml.
	unattendArch string
	// modern indica Windows 10 o successivo (opzioni OOBE aggiuntive).
	modern   bool
	memoryMB int
	diskGB   int
	// imageKey e imageValue scelgono l'edizione da installare dalla ISO.
	imageKey   string
	imageValue string
	productKey string
}

// machines sono le macchine virtuali disponibili.
var machines = map[string]machine{
	"win10": {
		name:         "win10",
		title:        "Windows 10 x64",
		goarch:       "amd64",
		unattendArch: "amd64",
		modern:       true,
		memoryMB:     3072,
		diskGB:       40,
		imageKey:     "/IMAGE/NAME",
		imageValue:   "Windows 10 Pro",
		// Chiave generica di installazione pubblicata da Microsoft per
		// Windows 10 Pro: permette l'installazione ma non attiva Windows.
		productKey: "VK7JG-NPHTM-C97JM-9MPGT-3V66T",
	},
	"win7": {
		name:         "win7",
		title:        "Windows 7 SP1 x64",
		goarch:       "amd64",
		unattendArch: "amd64",
		memoryMB:     2048,
		diskGB:       32,
		// Le ISO di Windows 7 contengono spesso una sola edizione.
		imageKey:   "/IMAGE/INDEX",
		imageValue: "1",
	},
}

// Credenziali dell'utente amministratore creato nelle macchine di test.
// Sono macchine usa e getta, senza dati reali.
const (
	testUser     = "tester"
	testPassword = "SupportTools1!"
)

func lookupMachine(name string) (machine, error) {
	m, ok := machines[name]
	if !ok {
		return machine{}, fmt.Errorf("macchina sconosciuta %q: disponibili %v", name, machineNames())
	}
	return m, nil
}

func machineNames() []string {
	names := make([]string, 0, len(machines))
	for n := range machines {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// vmRoot è la cartella delle macchine virtuali: SUPPORTTOOLS_VM_DIR o
// ~/VM/SupportTools.
func vmRoot() (string, error) {
	if dir := os.Getenv("SUPPORTTOOLS_VM_DIR"); dir != "" {
		return dir, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "VM", "SupportTools"), nil
}

func (m machine) dir() (string, error) {
	root, err := vmRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, m.name), nil
}

func (m machine) disk() (string, error) {
	dir, err := m.dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "disk.qcow2"), nil
}

// installedMarker è il file creato a installazione verificata.
func (m machine) installedMarker() (string, error) {
	dir, err := m.dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "installed"), nil
}

func (m machine) installed() bool {
	marker, err := m.installedMarker()
	return err == nil && fileExists(marker)
}

func (m machine) requireInstalled() error {
	if !m.installed() {
		return errors.New("la macchina " + m.name + " non è installata: esegui go run ./vm install " +
			m.name + " -iso <file.iso>")
	}
	return nil
}

// distArch è il nome dell'architettura usato dal programma di build.
func (m machine) distArch() string {
	return "x64"
}
