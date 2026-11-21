// SPDX-License-Identifier: GPL-3.0-or-later

package main

import "testing"

func TestRuneToKey(t *testing.T) {
	cases := []struct {
		r     rune
		qcode string
		shift bool
		ok    bool
	}{
		{'a', "a", false, true},
		{'Z', "z", true, true},
		{'5', "5", false, true},
		{'%', "5", true, true},
		{' ', "spc", false, true},
		{'\n', "ret", false, true},
		{'-', "minus", false, true},
		{'_', "minus", true, true},
		{'?', "slash", true, true},
		{'é', "", false, false}, // accentata: non mappata
	}
	for _, c := range cases {
		kp, ok := runeToKey(c.r)
		if ok != c.ok {
			t.Errorf("runeToKey(%q): ok=%v, atteso %v", c.r, ok, c.ok)
			continue
		}
		if ok && (kp.qcode != c.qcode || kp.shift != c.shift) {
			t.Errorf("runeToKey(%q) = {%q,%v}, atteso {%q,%v}", c.r, kp.qcode, kp.shift, c.qcode, c.shift)
		}
	}
}

func TestResolveKey(t *testing.T) {
	cases := []struct {
		spec string
		mods []string
		key  string
		err  bool
	}{
		{"enter", nil, "ret", false},
		{"esc", nil, "esc", false},
		{"f5", nil, "f5", false},
		{"a", nil, "a", false},
		{"ctrl+a", []string{"ctrl"}, "a", false},
		{"ctrl+shift+del", []string{"ctrl", "shift"}, "delete", false},
		{"win+d", []string{"meta_l"}, "d", false},
		{"win", nil, "meta_l", false},
		{"pageup", nil, "pgup", false},
		{"", nil, "", true},
		{"ctrl+", nil, "", true},
		{"nope+a", nil, "", true},
		{"f13", nil, "", true},
	}
	for _, c := range cases {
		mods, key, err := resolveKey(c.spec)
		if (err != nil) != c.err {
			t.Errorf("resolveKey(%q): err=%v, atteso err=%v", c.spec, err, c.err)
			continue
		}
		if c.err {
			continue
		}
		if key != c.key || !equalStrings(mods, c.mods) {
			t.Errorf("resolveKey(%q) = %v,%q, atteso %v,%q", c.spec, mods, key, c.mods, c.key)
		}
	}
}

func TestParseCoord(t *testing.T) {
	x, y, err := parseCoord("50,40.5")
	if err != nil || x != 50 || y != 40.5 {
		t.Errorf("parseCoord(\"50,40.5\") = %v,%v,%v", x, y, err)
	}
	if _, _, err := parseCoord("50"); err == nil {
		t.Error("parseCoord(\"50\"): atteso errore")
	}
	if _, _, err := parseCoord("a,b"); err == nil {
		t.Error("parseCoord(\"a,b\"): atteso errore")
	}
}

func TestAbsValue(t *testing.T) {
	cases := []struct {
		percent float64
		want    int
	}{
		{0, 0},
		{100, absMax},
		{50, absMax / 2},
		{-10, 0},      // sotto il minimo
		{150, absMax}, // sopra il massimo
	}
	for _, c := range cases {
		if got := absValue(c.percent); got != c.want {
			t.Errorf("absValue(%v) = %d, atteso %d", c.percent, got, c.want)
		}
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
