// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// integrationPackages sono i package con test di integrazione da eseguire
// su Windows (tag di build "integration").
var integrationPackages = []string{"./internal/users"}

// testTimeout è la durata massima di ogni eseguibile di test.
const testTimeout = 20 * time.Minute

// runTests compila i test di integrazione e li esegue nella macchina.
func runTests(m machine, o sessionOptions) error {
	if err := m.requireInstalled(); err != nil {
		return err
	}
	tmp, err := os.MkdirTemp("", "supporttools-vm-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)

	var exes []string
	for _, pkg := range integrationPackages {
		exe := filepath.Join(tmp, filepath.Base(pkg)+".test.exe")
		info("Compilazione dei test di %s (%s)", pkg, m.goarch)
		if err := runCmd(windowsEnv(m.goarch), "go", "test", "-c", "-tags", "integration",
			"-o", exe, pkg); err != nil {
			return err
		}
		exes = append(exes, exe)
	}

	info("Avvio di %s", m.title)
	s, err := startSession(m, o)
	if err != nil {
		return err
	}
	defer s.close()
	if err := s.waitReady(); err != nil {
		return err
	}

	failed := 0
	for i, exe := range exes {
		info("Esecuzione dei test di %s", integrationPackages[i])
		res, err := s.exec([]string{exe}, filepath.Base(exe),
			[]string{"-test.v", "-test.timeout", testTimeout.String()}, testTimeout)
		if err != nil {
			return err
		}
		fmt.Print(res.Output)
		if !strings.HasSuffix(res.Output, "\n") {
			fmt.Println()
		}
		if res.Error != "" {
			fmt.Fprintln(os.Stderr, "[x]", res.Error)
		}
		if res.ExitCode != 0 || res.Error != "" {
			failed++
		}
	}
	if failed > 0 {
		return fmt.Errorf("test non superati su %s: %d package su %d", m.title, failed, len(exes))
	}
	info("Tutti i test superati su %s", m.title)
	return nil
}

// screenshot avvia Support Tools nella macchina e ne salva una schermata in
// dist/vm.
func screenshot(m machine, tags string, wait time.Duration, o sessionOptions) error {
	if err := m.requireInstalled(); err != nil {
		return err
	}
	exe, err := buildApp(m, tags)
	if err != nil {
		return err
	}
	info("Avvio di %s", m.title)
	s, err := startSession(m, o)
	if err != nil {
		return err
	}
	defer s.close()
	if err := s.waitReady(); err != nil {
		return err
	}
	if err := s.launch([]string{exe}, filepath.Base(exe)); err != nil {
		return err
	}
	info("Attesa di %s prima della schermata", wait)
	time.Sleep(wait)

	out, err := filepath.Abs(filepath.Join("dist", "vm",
		fmt.Sprintf("%s-%s.png", m.name, time.Now().Format("20060102-150405"))))
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		return err
	}
	if err := s.qmp.screendump(out); err != nil {
		return err
	}
	info("Schermata salvata in %s", out)
	return nil
}

// start avvia la macchina con la finestra visibile e Support Tools già
// aperto, per le prove manuali; termina quando si chiude la finestra.
func start(m machine, tags string, o sessionOptions) error {
	if err := m.requireInstalled(); err != nil {
		return err
	}
	exe, err := buildApp(m, tags)
	if err != nil {
		return err
	}
	if !o.snapshot {
		info("Attenzione: con -persist le modifiche restano sul disco della macchina")
	}
	info("Avvio di %s", m.title)
	s, err := startSession(m, o)
	if err != nil {
		return err
	}
	defer s.close()
	if err := s.waitReady(); err != nil {
		return err
	}
	if err := s.launch([]string{exe}, filepath.Base(exe)); err != nil {
		return err
	}
	info("Support Tools avviato: chiudi la finestra della macchina per terminare")
	s.wait()
	return nil
}

// buildApp compila Support Tools per l'architettura della macchina con il
// programma di build e restituisce il percorso dell'eseguibile.
func buildApp(m machine, tags string) (string, error) {
	args := []string{"run", "./build", "-arch", m.distArch(), "-skip-checks"}
	dir := m.distArch()
	if tags != "" {
		args = append(args, "-tags", tags)
		dir += "-" + strings.ReplaceAll(tags, ",", "-")
	}
	if err := runCmd(nil, "go", args...); err != nil {
		return "", err
	}
	return filepath.Abs(filepath.Join("dist", dir, "SupportTools.exe"))
}
