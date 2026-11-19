// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"encoding/xml"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteSetupFiles(t *testing.T) {
	for _, name := range machineNames() {
		t.Run(name, func(t *testing.T) {
			m := machines[name]
			dir := t.TempDir()
			o := installOptions{lang: "it-IT", timeZone: "W. Europe Standard Time"}
			if err := writeSetupFiles(m, o, dir); err != nil {
				t.Fatal(err)
			}

			data, err := os.ReadFile(filepath.Join(dir, "autounattend.xml"))
			if err != nil {
				t.Fatal(err)
			}
			text := string(data)
			// Il file deve essere XML ben formato.
			dec := xml.NewDecoder(strings.NewReader(text))
			for {
				if _, err := dec.Token(); err == io.EOF {
					break
				} else if err != nil {
					t.Fatalf("autounattend.xml non valido: %v", err)
				}
			}
			for _, want := range []string{
				`processorArchitecture="` + m.unattendArch + `"`,
				"<Name>" + testUser + "</Name>",
				"<Key>" + m.imageKey + "</Key>",
				"<UILanguage>it-IT</UILanguage>",
			} {
				if !strings.Contains(text, want) {
					t.Errorf("autounattend.xml non contiene %q", want)
				}
			}
			if m.productKey == "" && strings.Contains(text, "<ProductKey>") {
				t.Error("ProductKey presente senza chiave")
			}
			if strings.Contains(text, "{{") {
				t.Error("segnaposto del modello non sostituiti")
			}

			cmd, err := os.ReadFile(filepath.Join(dir, "stsetup.cmd"))
			if err != nil {
				t.Fatal(err)
			}
			if strings.Count(string(cmd), "\n") != strings.Count(string(cmd), "\r\n") {
				t.Error("stsetup.cmd deve usare terminatori di riga CRLF")
			}
			if !strings.Contains(string(cmd), `/ru "`+testUser+`"`) {
				t.Error("stsetup.cmd non usa l'utente di test")
			}
		})
	}
}

func TestQEMUArgs(t *testing.T) {
	args, err := qemuArgs(qemuOptions{
		machine:   machines["win7"],
		disk:      "/vm/disk.qcow2",
		cdroms:    []string{"/iso/win.iso", "/vm/setup.iso"},
		bootCD:    true,
		snapshot:  true,
		qmpSocket: "/tmp/q.sock",
		allowTCG:  true,
	})
	if err != nil {
		t.Fatal(err)
	}
	line := strings.Join(args, " ")
	for _, want := range []string{
		"-m 2048",
		"file=/vm/disk.qcow2,if=ide,index=0,media=disk,format=qcow2",
		"file=/iso/win.iso,if=ide,index=1,media=cdrom",
		"file=/vm/setup.iso,if=ide,index=2,media=cdrom",
		"-boot order=cd",
		"-device e1000,netdev=net0",
		"-device usb-tablet",
		"-qmp unix:/tmp/q.sock,server=on,wait=off",
		"-display none",
		"-snapshot",
	} {
		if !strings.Contains(line, want) {
			t.Errorf("argomenti di QEMU senza %q:\n%s", want, line)
		}
	}
}
