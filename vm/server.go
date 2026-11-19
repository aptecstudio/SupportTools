// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/aptecstudio/supporttools/vm/internal/protocol"
)

// jobServer distribuisce i lavori all'agente nella macchina virtuale.
type jobServer struct {
	srv *http.Server

	mu      sync.Mutex
	queue   []*pendingJob
	byID    map[string]*pendingJob
	counter int

	ready     chan struct{}
	readyOnce sync.Once
}

type pendingJob struct {
	job protocol.Job
	// files associa il nome di ogni file al suo percorso sull'host.
	files  map[string]string
	result chan protocol.Result
}

// newJobServer avvia il server su 127.0.0.1: QEMU inoltra lì le connessioni
// della macchina virtuale verso 10.0.2.2.
func newJobServer() (*jobServer, error) {
	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", protocol.Port))
	if err != nil {
		return nil, fmt.Errorf("porta %d occupata (c'è già una macchina di test in esecuzione?): %w",
			protocol.Port, err)
	}
	s := &jobServer{byID: make(map[string]*pendingJob), ready: make(chan struct{})}
	mux := http.NewServeMux()
	mux.HandleFunc(protocol.PathNext, s.handleNext)
	mux.HandleFunc(protocol.PathFiles, s.handleFile)
	mux.HandleFunc(protocol.PathResult, s.handleResult)
	s.srv = &http.Server{Handler: mux, ReadHeaderTimeout: 10 * time.Second}
	go s.srv.Serve(ln)
	return s, nil
}

func (s *jobServer) close() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	s.srv.Shutdown(ctx)
}

// handleNext consegna il prossimo lavoro; la prima richiesta segnala che
// l'agente è attivo.
func (s *jobServer) handleNext(w http.ResponseWriter, r *http.Request) {
	s.readyOnce.Do(func() { close(s.ready) })

	s.mu.Lock()
	var pj *pendingJob
	if len(s.queue) > 0 {
		pj, s.queue = s.queue[0], s.queue[1:]
	}
	s.mu.Unlock()

	if pj == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(pj.job)
}

// handleFile serve /files/<job>/<nome>.
func (s *jobServer) handleFile(w http.ResponseWriter, r *http.Request) {
	parts := strings.SplitN(strings.TrimPrefix(r.URL.Path, protocol.PathFiles), "/", 2)
	if len(parts) != 2 {
		http.NotFound(w, r)
		return
	}
	s.mu.Lock()
	pj := s.byID[parts[0]]
	s.mu.Unlock()
	if pj == nil {
		http.NotFound(w, r)
		return
	}
	hostPath, ok := pj.files[parts[1]]
	if !ok {
		http.NotFound(w, r)
		return
	}
	http.ServeFile(w, r, hostPath)
}

// handleResult riceve /result/<job>.
func (s *jobServer) handleResult(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, protocol.PathResult)
	s.mu.Lock()
	pj := s.byID[id]
	delete(s.byID, id)
	s.mu.Unlock()
	if pj == nil {
		http.NotFound(w, r)
		return
	}
	var res protocol.Result
	if err := json.NewDecoder(r.Body).Decode(&res); err != nil {
		res = protocol.Result{ExitCode: -1, Error: "risultato non leggibile: " + err.Error()}
	}
	pj.result <- res
	w.WriteHeader(http.StatusNoContent)
}

// run accoda un lavoro e ne attende il risultato. files sono percorsi
// sull'host, resi disponibili all'agente con il loro nome.
func (s *jobServer) run(ctx context.Context, files []string, command string, args []string,
	wait bool, timeout time.Duration) (protocol.Result, error) {
	s.mu.Lock()
	s.counter++
	id := fmt.Sprintf("%d-%d", time.Now().Unix(), s.counter)
	pj := &pendingJob{
		job: protocol.Job{
			ID:             id,
			Command:        command,
			Args:           args,
			Wait:           wait,
			TimeoutSeconds: int(timeout / time.Second),
		},
		files:  make(map[string]string),
		result: make(chan protocol.Result, 1),
	}
	for _, f := range files {
		name := path.Base(strings.ReplaceAll(f, `\`, "/"))
		pj.files[name] = f
		pj.job.Files = append(pj.job.Files, name)
	}
	s.queue = append(s.queue, pj)
	s.byID[id] = pj
	s.mu.Unlock()

	select {
	case res := <-pj.result:
		return res, nil
	case <-ctx.Done():
		return protocol.Result{}, ctx.Err()
	}
}
