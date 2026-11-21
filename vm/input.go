// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"fmt"
	"strconv"
	"strings"
)

// keyPress è la pressione di un singolo carattere: il qcode QEMU del tasto
// fisico e se richiede il tasto Maiusc.
type keyPress struct {
	qcode string
	shift bool
}

// runeKeys mappa i caratteri ASCII sul qcode del tasto fisico corrispondente
// nel layout di tastiera US. Lettere, cifre e spazio sono affidabili su
// qualsiasi layout del guest; i simboli presuppongono il layout US.
var runeKeys = func() map[rune]keyPress {
	m := map[rune]keyPress{
		' ':  {"spc", false},
		'\n': {"ret", false},
		'\t': {"tab", false},
	}
	// Lettere: stessa posizione fisica in minuscolo e maiuscolo.
	for c := 'a'; c <= 'z'; c++ {
		m[c] = keyPress{string(c), false}
		m[c-32] = keyPress{string(c), true} // la maiuscola corrispondente
	}
	// Cifre e relativi simboli con Maiusc (riga dei numeri, layout US).
	digits := []struct {
		d, s rune
	}{
		{'1', '!'}, {'2', '@'}, {'3', '#'}, {'4', '$'}, {'5', '%'},
		{'6', '^'}, {'7', '&'}, {'8', '*'}, {'9', '('}, {'0', ')'},
	}
	for _, p := range digits {
		m[p.d] = keyPress{string(p.d), false}
		m[p.s] = keyPress{string(p.d), true}
	}
	// Tasti dei simboli, con e senza Maiusc.
	symbols := []struct {
		qcode     string
		base, top rune
	}{
		{"minus", '-', '_'},
		{"equal", '=', '+'},
		{"bracket_left", '[', '{'},
		{"bracket_right", ']', '}'},
		{"backslash", '\\', '|'},
		{"semicolon", ';', ':'},
		{"apostrophe", '\'', '"'},
		{"grave_accent", '`', '~'},
		{"comma", ',', '<'},
		{"dot", '.', '>'},
		{"slash", '/', '?'},
	}
	for _, s := range symbols {
		m[s.base] = keyPress{s.qcode, false}
		m[s.top] = keyPress{s.qcode, true}
	}
	return m
}()

// runeToKey restituisce il tasto da premere per digitare un carattere.
func runeToKey(r rune) (keyPress, bool) {
	kp, ok := runeKeys[r]
	return kp, ok
}

// keyAliases traduce i nomi dei tasti usati negli script nel qcode QEMU.
var keyAliases = map[string]string{
	"enter":     "ret",
	"return":    "ret",
	"esc":       "esc",
	"escape":    "esc",
	"tab":       "tab",
	"space":     "spc",
	"backspace": "backspace",
	"delete":    "delete",
	"del":       "delete",
	"insert":    "insert",
	"home":      "home",
	"end":       "end",
	"pageup":    "pgup",
	"pagedown":  "pgdn",
	"up":        "up",
	"down":      "down",
	"left":      "left",
	"right":     "right",
	"win":       "meta_l",
	"meta":      "meta_l",
	"super":     "meta_l",
}

// modAliases traduce i nomi dei modificatori nel qcode QEMU.
var modAliases = map[string]string{
	"ctrl":    "ctrl",
	"control": "ctrl",
	"alt":     "alt",
	"shift":   "shift",
	"win":     "meta_l",
	"meta":    "meta_l",
	"super":   "meta_l",
}

// resolveKey interpreta una combinazione di tasti come "ctrl+shift+a" o "f5"
// e restituisce i modificatori e il tasto principale come qcode QEMU.
func resolveKey(spec string) (mods []string, key string, err error) {
	parts := strings.Split(spec, "+")
	if len(parts) == 0 || parts[len(parts)-1] == "" {
		return nil, "", fmt.Errorf("combinazione di tasti vuota")
	}
	for _, p := range parts[:len(parts)-1] {
		q, ok := modAliases[strings.ToLower(p)]
		if !ok {
			return nil, "", fmt.Errorf("modificatore sconosciuto %q", p)
		}
		mods = append(mods, q)
	}
	key, err = resolveSingleKey(parts[len(parts)-1])
	if err != nil {
		return nil, "", err
	}
	return mods, key, nil
}

// resolveSingleKey traduce il nome di un tasto nel suo qcode QEMU.
func resolveSingleKey(name string) (string, error) {
	low := strings.ToLower(name)
	if q, ok := keyAliases[low]; ok {
		return q, nil
	}
	// Tasti funzione f1..f12.
	if len(low) >= 2 && low[0] == 'f' {
		if n, err := strconv.Atoi(low[1:]); err == nil && n >= 1 && n <= 12 {
			return low, nil
		}
	}
	// Singolo carattere digitabile (lettera o cifra, senza Maiusc).
	r := []rune(name)
	if len(r) == 1 {
		if kp, ok := runeToKey(r[0]); ok && !kp.shift {
			return kp.qcode, nil
		}
	}
	return "", fmt.Errorf("tasto sconosciuto %q", name)
}

// parseCoord interpreta una coordinata "x,y" in percentuale dello schermo.
func parseCoord(s string) (x, y float64, err error) {
	parts := strings.Split(s, ",")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("coordinata %q non valida: usa x,y in percentuale", s)
	}
	if x, err = strconv.ParseFloat(strings.TrimSpace(parts[0]), 64); err != nil {
		return 0, 0, fmt.Errorf("coordinata X non valida: %w", err)
	}
	if y, err = strconv.ParseFloat(strings.TrimSpace(parts[1]), 64); err != nil {
		return 0, 0, fmt.Errorf("coordinata Y non valida: %w", err)
	}
	return x, y, nil
}
