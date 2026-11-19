// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

// qemuOptions sono le opzioni di avvio di una macchina virtuale.
type qemuOptions struct {
	machine machine
	disk    string
	// cdroms sono le ISO da collegare, in ordine.
	cdroms []string
	// bootCD avvia dal CD se il disco non è avviabile (installazione).
	bootCD bool
	// snapshot scarta tutte le modifiche al disco allo spegnimento.
	snapshot bool
	// show mostra la finestra della macchina virtuale.
	show bool
	// qmpSocket è il socket per il controllo della macchina (QMP).
	qmpSocket string
	// allowTCG permette l'emulazione software se manca l'accelerazione.
	allowTCG bool
}

// findQEMU cerca qemu-system-x86_64 nel PATH e nelle cartelle di Homebrew.
func findQEMU() (string, error) {
	if p, err := exec.LookPath("qemu-system-x86_64"); err == nil {
		return p, nil
	}
	for _, p := range []string{"/opt/homebrew/bin/qemu-system-x86_64", "/usr/local/bin/qemu-system-x86_64"} {
		if fileExists(p) {
			return p, nil
		}
	}
	return "", errors.New("qemu-system-x86_64 non trovato: esegui lo script di setup del tuo sistema in scripts/")
}

// findQEMUImg cerca qemu-img accanto a qemu-system-x86_64 o nel PATH.
func findQEMUImg() (string, error) {
	if p, err := exec.LookPath("qemu-img"); err == nil {
		return p, nil
	}
	for _, p := range []string{"/opt/homebrew/bin/qemu-img", "/usr/local/bin/qemu-img"} {
		if fileExists(p) {
			return p, nil
		}
	}
	return "", errors.New("qemu-img non trovato: esegui lo script di setup del tuo sistema in scripts/")
}

// accelArgs sceglie l'accelerazione hardware disponibile: KVM su Linux,
// Hypervisor.framework sui Mac Intel. Senza accelerazione Windows è troppo
// lento per essere usato, quindi l'emulazione va chiesta esplicitamente.
func accelArgs(allowTCG bool) ([]string, error) {
	switch runtime.GOOS {
	case "linux":
		f, err := os.OpenFile("/dev/kvm", os.O_RDWR, 0)
		if err == nil {
			f.Close()
			return []string{"-accel", "kvm", "-cpu", "host"}, nil
		}
		if !allowTCG {
			return nil, fmt.Errorf("KVM non disponibile (%v): abilita la virtualizzazione nel BIOS "+
				"e aggiungi l'utente al gruppo kvm (sudo usermod -aG kvm $USER, poi riaccedi), "+
				"oppure usa -tcg per l'emulazione software, molto lenta", err)
		}
	case "darwin":
		if runtime.GOARCH == "arm64" {
			if !allowTCG {
				return nil, errors.New("sui Mac Apple Silicon Windows x64 si può solo emulare " +
					"ed è molto lento: usa -tcg per provarlo comunque")
			}
			break
		}
		out, err := exec.Command("sysctl", "-n", "kern.hv_support").Output()
		if err == nil && strings.TrimSpace(string(out)) == "1" {
			return []string{"-accel", "hvf", "-cpu", "host"}, nil
		}
		if !allowTCG {
			return nil, errors.New("accelerazione Hypervisor.framework non disponibile: " +
				"usa -tcg per l'emulazione software, molto lenta")
		}
	default:
		if !allowTCG {
			return nil, fmt.Errorf("sistema %s non supportato per le macchine di test", runtime.GOOS)
		}
	}
	return []string{"-accel", "tcg", "-cpu", "max"}, nil
}

// qemuArgs costruisce la riga di comando di QEMU. L'hardware è scelto per
// funzionare con i driver inclusi in Windows 7 e 10, senza driver virtio:
// chipset i440fx, dischi IDE, scheda di rete e1000, tavoletta USB (per il
// puntatore assoluto) e scheda video standard.
func qemuArgs(o qemuOptions) ([]string, error) {
	accel, err := accelArgs(o.allowTCG)
	if err != nil {
		return nil, err
	}
	args := []string{
		"-name", "SupportTools-" + o.machine.name,
		"-machine", "pc",
	}
	args = append(args, accel...)
	args = append(args,
		"-smp", "2",
		"-m", strconv.Itoa(o.machine.memoryMB),
		// Windows si aspetta l'orologio hardware nell'ora locale.
		"-rtc", "base=localtime",
		"-drive", "file="+o.disk+",if=ide,index=0,media=disk,format=qcow2",
	)
	for i, iso := range o.cdroms {
		args = append(args, "-drive", fmt.Sprintf("file=%s,if=ide,index=%d,media=cdrom,readonly=on", iso, i+1))
	}
	if o.bootCD {
		args = append(args, "-boot", "order=cd")
	} else {
		args = append(args, "-boot", "order=c")
	}
	args = append(args,
		"-netdev", "user,id=net0",
		"-device", "e1000,netdev=net0",
		"-usb", "-device", "usb-tablet",
		"-vga", "std",
	)
	if o.qmpSocket != "" {
		args = append(args, "-qmp", "unix:"+o.qmpSocket+",server=on,wait=off")
	}
	if !o.show {
		args = append(args, "-display", "none")
	}
	if o.snapshot {
		args = append(args, "-snapshot")
	}
	return args, nil
}

// startQEMU avvia la macchina virtuale.
func startQEMU(o qemuOptions) (*exec.Cmd, error) {
	qemu, err := findQEMU()
	if err != nil {
		return nil, err
	}
	args, err := qemuArgs(o)
	if err != nil {
		return nil, err
	}
	cmd := exec.Command(qemu, args...)
	cmd.Env = hostEnv()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("avvio di QEMU: %w", err)
	}
	return cmd, nil
}
