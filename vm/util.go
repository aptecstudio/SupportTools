// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// windowsEnv restituisce l'ambiente per compilare verso Windows.
func windowsEnv(goarch string) []string {
	return append(os.Environ(),
		"GOOS=windows",
		"GOARCH="+goarch,
		"CGO_ENABLED=0",
		"GOTOOLCHAIN=local",
	)
}

// inSnap indica se un percorso sta dentro uno snap. Gli snap stanno sia in
// /snap/... sia nei dati utente ~/snap/..., quindi si cerca il segmento /snap/
// (o /snapd/) ovunque nel percorso.
func inSnap(path string) bool {
	return strings.Contains(path, "/snap/") || strings.Contains(path, "/snapd/")
}

// hostEnv restituisce l'ambiente corrente ripulito dalle variabili che
// farebbero caricare a QEMU librerie, moduli GTK/GIO/GdkPixbuf o locale da
// dentro uno snap. Quando il programma viene lanciato da un editor impacchettato
// come snap (per es. VS Code), QEMU apre la finestra GTK usando i moduli dello
// snap e trascina librerie di sistema incompatibili (la libpthread di core20),
// andando in errore all'avvio ("undefined symbol __libc_pthread_init"). Tolte
// quelle variabili, QEMU usa GTK e le librerie di sistema.
func hostEnv() []string {
	// Variabili che indirizzano il caricamento di librerie, moduli o locale:
	// nell'ambiente di uno snap puntano dentro lo snap e vanno rimosse del tutto
	// (fuori da uno snap di norma non sono impostate, quindi toglierle è innocuo).
	redirect := map[string]bool{
		"LD_PRELOAD":             true,
		"GTK_PATH":               true,
		"GTK_EXE_PREFIX":         true,
		"GTK_IM_MODULE_FILE":     true,
		"GDK_PIXBUF_MODULE_FILE": true,
		"GDK_PIXBUF_MODULEDIR":   true,
		"GIO_MODULE_DIR":         true,
		"GSETTINGS_SCHEMA_DIR":   true,
		"LOCPATH":                true,
		"GCONV_PATH":             true,
	}
	env := os.Environ()
	out := make([]string, 0, len(env))
	for _, e := range env {
		name, val, ok := strings.Cut(e, "=")
		switch {
		case ok && name == "LD_LIBRARY_PATH":
			// Tiene solo i percorsi che non sono dentro uno snap.
			var kept []string
			for _, p := range filepath.SplitList(val) {
				if !inSnap(p) {
					kept = append(kept, p)
				}
			}
			if len(kept) > 0 {
				out = append(out, name+"="+strings.Join(kept, string(os.PathListSeparator)))
			}
		case ok && redirect[name] && inSnap(val):
			// Variabile dello snap che redirige il caricamento: rimossa.
		default:
			out = append(out, e)
		}
	}
	return out
}

func runCmd(env []string, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Env = env
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s %s: %w", filepath.Base(name), strings.Join(args, " "), err)
	}
	return nil
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func info(format string, args ...interface{}) {
	fmt.Printf("==> "+format+"\n", args...)
}
