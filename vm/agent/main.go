// SPDX-License-Identifier: GPL-3.0-or-later

//go:build windows

// Il programma agent gira nella macchina virtuale Windows di test: esegue i
// comandi ricevuti dal programma vm sull'host (vedi vm/internal/protocol).
//
// Viene installato in C:\stagent durante l'installazione non presidiata di
// Windows e avviato all'accesso dell'utente di test con i privilegi più
// elevati, nella sessione interattiva: così può eseguire sia test che
// richiedono l'amministratore sia programmi grafici visibili sullo schermo.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"

	"github.com/aptecstudio/supporttools/vm/internal/protocol"
)

const (
	baseDir      = `C:\stagent`
	pollInterval = time.Second
	// maxOutput limita l'output inviato all'host.
	maxOutput = 8 << 20
)

var baseURL = fmt.Sprintf("http://%s:%d", protocol.HostAddr, protocol.Port)

func main() {
	logFile, err := os.OpenFile(filepath.Join(baseDir, "agent.log"),
		os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err == nil {
		log.SetOutput(logFile)
	}
	log.Printf("agente avviato, host %s", baseURL)

	client := &http.Client{Timeout: 30 * time.Second}
	for {
		job, err := next(client)
		if err != nil || job == nil {
			// Host non raggiungibile o nessun lavoro: si riprova.
			time.Sleep(pollInterval)
			continue
		}
		log.Printf("lavoro %s: %s %v", job.ID, job.Command, job.Args)
		res := run(client, job)
		if err := sendResult(client, job.ID, res); err != nil {
			log.Printf("invio del risultato di %s: %v", job.ID, err)
		}
	}
}

// next chiede all'host il prossimo lavoro; nil se non ce ne sono.
func next(client *http.Client) (*protocol.Job, error) {
	resp, err := client.Get(baseURL + protocol.PathNext)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNoContent {
		return nil, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("stato HTTP %s", resp.Status)
	}
	var job protocol.Job
	if err := json.NewDecoder(resp.Body).Decode(&job); err != nil {
		return nil, err
	}
	return &job, nil
}

// run scarica i file del lavoro ed esegue il comando.
func run(client *http.Client, job *protocol.Job) protocol.Result {
	dir := filepath.Join(baseDir, "jobs", job.ID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return protocol.Result{ExitCode: -1, Error: err.Error()}
	}
	for _, name := range job.Files {
		if err := download(client, job.ID, name, filepath.Join(dir, filepath.Base(name))); err != nil {
			return protocol.Result{ExitCode: -1, Error: fmt.Sprintf("download di %s: %v", name, err)}
		}
	}

	command := job.Command
	if local := filepath.Join(dir, filepath.Base(command)); fileExists(local) {
		command = local
	}

	ctx := context.Background()
	if job.Wait && job.TimeoutSeconds > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(job.TimeoutSeconds)*time.Second)
		defer cancel()
	}
	cmd := exec.CommandContext(ctx, command, job.Args...)
	cmd.Dir = dir

	if !job.Wait {
		if err := cmd.Start(); err != nil {
			return protocol.Result{ExitCode: -1, Error: err.Error()}
		}
		// Il programma resta in esecuzione da solo: si attende solo per
		// liberare le risorse del processo quando termina.
		go cmd.Wait()
		return protocol.Result{}
	}

	// I comandi a riga di comando non devono aprire finestre sul desktop.
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	var out limitedBuffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	res := protocol.Result{Output: out.String()}
	var exitErr *exec.ExitError
	switch {
	case err == nil:
	case errors.As(err, &exitErr):
		res.ExitCode = exitErr.ExitCode()
	default:
		res.ExitCode = -1
		res.Error = err.Error()
	}
	if ctx.Err() != nil {
		// Il processo è stato terminato per il timeout.
		res.ExitCode = -1
		res.Error = fmt.Sprintf("tempo scaduto dopo %d secondi", job.TimeoutSeconds)
	}
	return res
}

func download(client *http.Client, jobID, name, dst string) error {
	resp, err := client.Get(baseURL + protocol.PathFiles + jobID + "/" + name)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("stato HTTP %s", resp.Status)
	}
	f, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err := io.Copy(f, resp.Body); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

func sendResult(client *http.Client, jobID string, res protocol.Result) error {
	body, err := json.Marshal(res)
	if err != nil {
		return err
	}
	resp, err := client.Post(baseURL+protocol.PathResult+jobID, "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// limitedBuffer raccoglie l'output fino a maxOutput byte.
type limitedBuffer struct {
	bytes.Buffer
	truncated bool
}

func (b *limitedBuffer) Write(p []byte) (int, error) {
	if room := maxOutput - b.Len(); room < len(p) {
		if room > 0 {
			b.Buffer.Write(p[:room])
		}
		b.truncated = true
		return len(p), nil
	}
	return b.Buffer.Write(p)
}

func (b *limitedBuffer) String() string {
	if b.truncated {
		return b.Buffer.String() + "\n[output troncato]"
	}
	return b.Buffer.String()
}
