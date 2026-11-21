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

// absMax è il valore massimo dell'asse assoluto della tavoletta USB in QEMU:
// l'intervallo 0..absMax copre l'intera larghezza o altezza dello schermo,
// indipendentemente dalla risoluzione.
const absMax = 32767

// keyInputEvent costruisce un evento di pressione o rilascio di un tasto,
// identificato dal suo qcode QEMU.
func keyInputEvent(qcode string, down bool) map[string]interface{} {
	return map[string]interface{}{
		"type": "key",
		"data": map[string]interface{}{
			"down": down,
			"key":  map[string]interface{}{"type": "qcode", "data": qcode},
		},
	}
}

// btnInputEvent costruisce un evento di pressione o rilascio di un pulsante
// del mouse (left, right, middle).
func btnInputEvent(button string, down bool) map[string]interface{} {
	return map[string]interface{}{
		"type": "btn",
		"data": map[string]interface{}{"down": down, "button": button},
	}
}

// absInputEvent costruisce un evento di posizionamento assoluto del puntatore
// su un asse (x o y).
func absInputEvent(axis string, value int) map[string]interface{} {
	return map[string]interface{}{
		"type": "abs",
		"data": map[string]interface{}{"axis": axis, "value": value},
	}
}

// absValue converte una posizione in percentuale dello schermo (0..100) nel
// valore assoluto atteso da QEMU (0..absMax).
func absValue(percent float64) int {
	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}
	return int(percent / 100 * absMax)
}

// sendInput invia una sequenza di eventi di input con un solo comando QMP,
// così da restare atomica (es. spostamento più clic).
func (c *qmpClient) sendInput(events ...map[string]interface{}) error {
	return c.execute("input-send-event", map[string]interface{}{"events": events})
}

// moveMouse posiziona il puntatore alle coordinate indicate in percentuale
// dello schermo (0..100 su entrambi gli assi).
func (c *qmpClient) moveMouse(xPercent, yPercent float64) error {
	return c.sendInput(
		absInputEvent("x", absValue(xPercent)),
		absInputEvent("y", absValue(yPercent)),
	)
}

// pressDelay è l'attesa tra la pressione e il rilascio di un pulsante o di un
// tasto. Pressione e rilascio nello stesso evento verrebbero visti da Windows
// come una pressione di durata zero e ignorati: vanno inviati separati, con una
// breve pausa in mezzo.
const pressDelay = 60 * time.Millisecond

// clickMouse sposta il puntatore e preme e rilascia un pulsante. Un pulsante
// vuoto equivale al tasto sinistro. La pressione e il rilascio sono inviati in
// due eventi distinti, perché insieme Windows non li registra come clic.
func (c *qmpClient) clickMouse(button string, xPercent, yPercent float64) error {
	if button == "" {
		button = "left"
	}
	if err := c.moveMouse(xPercent, yPercent); err != nil {
		return err
	}
	if err := c.sendInput(btnInputEvent(button, true)); err != nil {
		return err
	}
	time.Sleep(pressDelay)
	return c.sendInput(btnInputEvent(button, false))
}

// tapKey preme una combinazione di tasti: i modificatori restano premuti
// mentre il tasto principale viene premuto e rilasciato, poi i modificatori
// vengono rilasciati in ordine inverso. Pressione e rilascio del tasto
// principale sono separati da una breve pausa, come per il clic del mouse.
func (c *qmpClient) tapKey(mods []string, key string) error {
	down := make([]map[string]interface{}, 0, len(mods)+1)
	for _, m := range mods {
		down = append(down, keyInputEvent(m, true))
	}
	down = append(down, keyInputEvent(key, true))
	if err := c.sendInput(down...); err != nil {
		return err
	}
	time.Sleep(pressDelay)
	up := make([]map[string]interface{}, 0, len(mods)+1)
	up = append(up, keyInputEvent(key, false))
	for i := len(mods) - 1; i >= 0; i-- {
		up = append(up, keyInputEvent(mods[i], false))
	}
	return c.sendInput(up...)
}

// typeText digita un testo carattere per carattere. È affidabile per lettere,
// cifre e spazio; i simboli presuppongono il layout di tastiera US del guest.
func (c *qmpClient) typeText(text string) error {
	for _, r := range text {
		kp, ok := runeToKey(r)
		if !ok {
			return fmt.Errorf("carattere non digitabile: %q", r)
		}
		var mods []string
		if kp.shift {
			mods = []string{"shift"}
		}
		if err := c.tapKey(mods, kp.qcode); err != nil {
			return err
		}
	}
	return nil
}

// quit termina QEMU immediatamente.
func (c *qmpClient) quit() error {
	return c.execute("quit", nil)
}

func (c *qmpClient) close() {
	c.conn.Close()
}
