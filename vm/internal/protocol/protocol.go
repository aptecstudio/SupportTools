// SPDX-License-Identifier: GPL-3.0-or-later

// Package protocol definisce i messaggi scambiati tra il programma vm
// (sull'host) e l'agente in esecuzione nella macchina virtuale Windows.
//
// L'agente interroga l'host via HTTP all'indirizzo che QEMU riserva
// all'host nella rete virtuale (10.0.2.2): chiede il prossimo lavoro,
// scarica i file necessari, esegue il comando e invia il risultato.
//
// Il protocollo è volutamente minimo e stabile: l'agente viene installato
// una sola volta, insieme a Windows, e cambiarlo richiede di reinstallare
// le macchine virtuali.
package protocol

// Port è la porta TCP su cui l'host ascolta (solo su 127.0.0.1).
const Port = 47800

// HostAddr è l'indirizzo dell'host visto dalla macchina virtuale.
const HostAddr = "10.0.2.2"

// Percorsi HTTP.
const (
	// PathNext restituisce il prossimo Job (200) o nessun lavoro (204).
	PathNext = "/next"
	// PathFiles è il prefisso per scaricare i file: /files/<job>/<nome>.
	PathFiles = "/files/"
	// PathResult è il prefisso per inviare il Result: /result/<job>.
	PathResult = "/result/"
)

// Job è un comando da eseguire nella macchina virtuale.
type Job struct {
	ID string `json:"id"`
	// Files sono i file da scaricare nella cartella del lavoro prima di
	// eseguire il comando.
	Files []string `json:"files,omitempty"`
	// Command è il programma da eseguire: un file del lavoro o un comando
	// di sistema.
	Command string   `json:"command"`
	Args    []string `json:"args,omitempty"`
	// Wait indica di attendere la fine del comando e raccoglierne l'output;
	// se falso il comando viene solo avviato (es. un programma grafico).
	Wait bool `json:"wait"`
	// TimeoutSeconds limita la durata del comando quando Wait è vero.
	TimeoutSeconds int `json:"timeoutSeconds,omitempty"`
}

// Result è l'esito di un Job.
type Result struct {
	ExitCode int    `json:"exitCode"`
	Output   string `json:"output"`
	// Error descrive un errore dell'agente (file non scaricato, comando non
	// avviato, timeout); vuoto se il comando è stato eseguito.
	Error string `json:"error,omitempty"`
}
