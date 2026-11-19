// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"bytes"
	"embed"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"text/template"
	"time"
)

//go:embed templates/autounattend.xml.tmpl templates/stsetup.cmd
var templates embed.FS

// installTimeout è il tempo massimo per l'installazione di Windows.
const installTimeout = 3 * time.Hour

type installOptions struct {
	iso      string
	lang     string
	timeZone string
	edition  string
	index    int
	key      string
	headless bool
	force    bool
	allowTCG bool
}

// install crea il disco, prepara il CD dell'installazione non presidiata e
// installa Windows; poi verifica che l'agente risponda.
func install(m machine, o installOptions) error {
	if o.iso == "" {
		return errors.New("indica la ISO di Windows con -iso <file.iso>")
	}
	if !fileExists(o.iso) {
		return fmt.Errorf("ISO non trovata: %s", o.iso)
	}
	iso, err := filepath.Abs(o.iso)
	if err != nil {
		return err
	}
	if m.installed() && !o.force {
		return fmt.Errorf("la macchina %s è già installata: usa -force per reinstallarla", m.name)
	}
	if o.edition != "" {
		m.imageKey, m.imageValue = "/IMAGE/NAME", o.edition
	}
	if o.index > 0 {
		m.imageKey, m.imageValue = "/IMAGE/INDEX", fmt.Sprint(o.index)
	}
	if o.key != "" {
		m.productKey = o.key
	}
	if o.key == "-" {
		m.productKey = ""
	}

	dir, err := m.dir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	marker, _ := m.installedMarker()
	os.Remove(marker)

	disk, _ := m.disk()
	os.Remove(disk)
	qemuImg, err := findQEMUImg()
	if err != nil {
		return err
	}
	info("Creazione del disco %s (%d GB)", disk, m.diskGB)
	if err := runCmd(hostEnv(), qemuImg, "create", "-q", "-f", "qcow2", disk, fmt.Sprintf("%dG", m.diskGB)); err != nil {
		return err
	}

	stage := filepath.Join(dir, "setup")
	os.RemoveAll(stage)
	if err := os.MkdirAll(stage, 0o755); err != nil {
		return err
	}
	info("Compilazione dell'agente (%s)", m.goarch)
	if err := runCmd(windowsEnv(m.goarch), "go", "build", "-trimpath",
		"-ldflags", "-s -w -H windowsgui",
		"-o", filepath.Join(stage, "agent.exe"), "./vm/agent"); err != nil {
		return err
	}
	if err := writeSetupFiles(m, o, stage); err != nil {
		return err
	}
	setupISO := filepath.Join(dir, "setup.iso")
	info("Creazione del CD di configurazione")
	if err := makeISO(stage, setupISO); err != nil {
		return err
	}

	info("Installazione di %s: richiede da 20 minuti a un'ora, la macchina si spegne da sola al termine", m.title)
	cmd, err := startQEMU(qemuOptions{
		machine:  m,
		disk:     disk,
		cdroms:   []string{iso, setupISO},
		bootCD:   true,
		show:     !o.headless,
		allowTCG: o.allowTCG,
	})
	if err != nil {
		return err
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case err := <-done:
		if err != nil {
			return fmt.Errorf("QEMU terminato con errore: %w", err)
		}
	case <-time.After(installTimeout):
		cmd.Process.Kill()
		return fmt.Errorf("installazione non conclusa in %s", installTimeout)
	}

	info("Verifica dell'installazione")
	s, err := startSession(m, sessionOptions{snapshot: true, allowTCG: o.allowTCG})
	if err != nil {
		return err
	}
	defer s.close()
	if err := s.waitReady(); err != nil {
		return fmt.Errorf("l'agente non risponde: l'installazione potrebbe non essere completa. "+
			"Riprova con -force; se l'installazione si ferma su una schermata, usala per capire il "+
			"problema (ISO in un'altra lingua: -lang; edizione: -edition o -index; chiave: -key). "+
			"Dettaglio: %w", err)
	}
	res, err := s.exec(nil, "cmd", []string{"/c", "ver"}, time.Minute)
	if err != nil {
		return err
	}
	if err := os.WriteFile(marker, []byte(time.Now().Format(time.RFC3339)+"\n"+res.Output), 0o644); err != nil {
		return err
	}
	info("Installata: %s", strings.TrimSpace(res.Output))
	return nil
}

// writeSetupFiles genera autounattend.xml e stsetup.cmd nella cartella stage.
func writeSetupFiles(m machine, o installOptions, stage string) error {
	data := map[string]interface{}{
		"Arch":         m.unattendArch,
		"Lang":         o.lang,
		"TimeZone":     o.timeZone,
		"ImageKey":     m.imageKey,
		"ImageValue":   m.imageValue,
		"ProductKey":   m.productKey,
		"ComputerName": "ST-" + strings.ToUpper(m.name),
		"Modern":       m.modern,
		"User":         testUser,
		"Password":     testPassword,
	}
	files := map[string]string{
		"templates/autounattend.xml.tmpl": "autounattend.xml",
		"templates/stsetup.cmd":           "stsetup.cmd",
	}
	for src, dst := range files {
		tmpl, err := template.ParseFS(templates, src)
		if err != nil {
			return err
		}
		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, data); err != nil {
			return err
		}
		// I file letti da Windows usano CRLF: il prompt dei comandi non
		// gestisce sempre bene i file batch con soli LF.
		out := strings.ReplaceAll(buf.String(), "\r\n", "\n")
		out = strings.ReplaceAll(out, "\n", "\r\n")
		if err := os.WriteFile(filepath.Join(stage, dst), []byte(out), 0o644); err != nil {
			return err
		}
	}
	return nil
}

// makeISO crea un'immagine ISO con il contenuto di dir: con hdiutil su macOS,
// con xorriso o genisoimage su Linux.
func makeISO(dir, out string) error {
	os.Remove(out)
	if runtime.GOOS == "darwin" {
		return runCmd(nil, "hdiutil", "makehybrid", "-quiet", "-iso", "-joliet",
			"-default-volume-name", "STSETUP", "-o", out, dir)
	}
	if p, err := exec.LookPath("xorriso"); err == nil {
		return runCmd(nil, p, "-as", "mkisofs", "-quiet", "-J", "-r", "-V", "STSETUP", "-o", out, dir)
	}
	if p, err := exec.LookPath("genisoimage"); err == nil {
		return runCmd(nil, p, "-quiet", "-J", "-r", "-V", "STSETUP", "-o", out, dir)
	}
	return errors.New("xorriso non trovato: esegui lo script di setup del tuo sistema in scripts/")
}
