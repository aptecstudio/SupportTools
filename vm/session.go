// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/aptecstudio/supporttools/vm/internal/protocol"
)

// bootTimeout è il tempo massimo tra l'avvio della macchina e il primo
// contatto dell'agente.
const bootTimeout = 10 * time.Minute

// session è una macchina virtuale in esecuzione con il suo agente.
type session struct {
	machine machine
	cmd     *exec.Cmd
	qmp     *qmpClient
	server  *jobServer
	socket  string
	exited  chan error
}

type sessionOptions struct {
	snapshot bool
	show     bool
	allowTCG bool
}

// startSession avvia la macchina e il server dei lavori. Il chiamante
// verifica prima che la macchina sia installata (requireInstalled).
func startSession(m machine, o sessionOptions) (*session, error) {
	disk, err := m.disk()
	if err != nil {
		return nil, err
	}
	server, err := newJobServer()
	if err != nil {
		return nil, err
	}
	// Percorso breve: i socket Unix hanno un limite di circa 100 caratteri.
	socket := filepath.Join(os.TempDir(), fmt.Sprintf("st-%s-%d.qmp", m.name, os.Getpid()))
	os.Remove(socket)

	cmd, err := startQEMU(qemuOptions{
		machine:   m,
		disk:      disk,
		snapshot:  o.snapshot,
		show:      o.show,
		qmpSocket: socket,
		allowTCG:  o.allowTCG,
	})
	if err != nil {
		server.close()
		return nil, err
	}
	s := &session{machine: m, cmd: cmd, server: server, socket: socket, exited: make(chan error, 1)}
	go func() { s.exited <- cmd.Wait() }()

	s.qmp, err = dialQMP(socket, 15*time.Second)
	if err != nil {
		s.close()
		return nil, err
	}
	return s, nil
}

// waitReady attende che l'agente contatti l'host.
func (s *session) waitReady() error {
	info("Attesa dell'avvio di Windows (fino a %s)", bootTimeout)
	select {
	case <-s.server.ready:
		return nil
	case err := <-s.exited:
		s.exited <- err
		return fmt.Errorf("QEMU è terminato prima dell'avvio di Windows: %v", err)
	case <-time.After(bootTimeout):
		return errors.New("Windows non ha risposto in tempo: avvia la macchina con -show per vedere cosa succede")
	}
}

// exec esegue un comando nella macchina e ne attende l'esito.
func (s *session) exec(files []string, command string, args []string, timeout time.Duration) (protocol.Result, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout+time.Minute)
	defer cancel()
	return s.server.run(ctx, files, command, args, true, timeout)
}

// launch avvia un programma nella macchina senza attenderne la fine.
func (s *session) launch(files []string, command string, args ...string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	res, err := s.server.run(ctx, files, command, args, false, 0)
	if err != nil {
		return err
	}
	if res.Error != "" {
		return errors.New(res.Error)
	}
	return nil
}

// wait attende che QEMU termini da solo (es. l'utente chiude la finestra).
func (s *session) wait() error {
	err := <-s.exited
	s.exited <- err
	return err
}

// close spegne la macchina; con -snapshot le modifiche vanno perse.
func (s *session) close() {
	if s.qmp != nil {
		s.qmp.quit()
		s.qmp.close()
	}
	select {
	case err := <-s.exited:
		s.exited <- err
	case <-time.After(10 * time.Second):
		s.cmd.Process.Kill()
	}
	s.server.close()
	os.Remove(s.socket)
}
