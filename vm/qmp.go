// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"time"
)

// qmpClient è un client minimo del QEMU Machine Protocol, usato per gli
// screenshot e per spegnere la macchina.
type qmpClient struct {
	conn   net.Conn
	reader *bufio.Reader
}

// dialQMP si collega al socket QMP, riprovando finché QEMU non lo crea.
func dialQMP(socket string, timeout time.Duration) (*qmpClient, error) {
	deadline := time.Now().Add(timeout)
	for {
		conn, err := net.Dial("unix", socket)
		if err == nil {
			c := &qmpClient{conn: conn, reader: bufio.NewReader(conn)}
			// Messaggio di benvenuto, poi abilitazione dei comandi.
			if _, err := c.readMessage(); err != nil {
				conn.Close()
				return nil, err
			}
			if err := c.execute("qmp_capabilities", nil); err != nil {
				conn.Close()
				return nil, err
			}
			return c, nil
		}
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("collegamento a QMP: %w", err)
		}
		time.Sleep(200 * time.Millisecond)
	}
}

type qmpMessage struct {
	Return json.RawMessage `json:"return"`
	Error  *struct {
		Class string `json:"class"`
		Desc  string `json:"desc"`
	} `json:"error"`
	Event string `json:"event"`
}

func (c *qmpClient) readMessage() (qmpMessage, error) {
	line, err := c.reader.ReadBytes('\n')
	if err != nil {
		return qmpMessage{}, err
	}
	var msg qmpMessage
	err = json.Unmarshal(line, &msg)
	return msg, err
}

// execute invia un comando e ne attende la risposta, ignorando gli eventi.
func (c *qmpClient) execute(command string, args interface{}) error {
	req := map[string]interface{}{"execute": command}
	if args != nil {
		req["arguments"] = args
	}
	data, err := json.Marshal(req)
	if err != nil {
		return err
	}
	if _, err := c.conn.Write(append(data, '\n')); err != nil {
		return err
	}
	for {
		msg, err := c.readMessage()
		if err != nil {
			return err
		}
		if msg.Event != "" {
			continue
		}
		if msg.Error != nil {
			return fmt.Errorf("QMP %s: %s", command, msg.Error.Desc)
		}
		return nil
	}
}

// screendump salva una schermata della macchina in formato PNG.
func (c *qmpClient) screendump(path string) error {
	return c.execute("screendump", map[string]string{"filename": path, "format": "png"})
}

// quit termina QEMU immediatamente.
func (c *qmpClient) quit() error {
	return c.execute("quit", nil)
}

func (c *qmpClient) close() {
	c.conn.Close()
}
