// SPDX-License-Identifier: GPL-3.0-or-later

// Il programma build compila SupportTools.exe per Windows a 32 e 64 bit da
// qualsiasi sistema (Linux, macOS, Windows).
//
// Va eseguito dalla radice del repository:
//
//	go run ./build [-arch all|x64|x86] [-tags demo] [-skip-checks] [-run]
//
// Passaggi:
//  1. verifica che la toolchain sia Go 1.20.x (l'ultima che supporta Windows 7);
//  2. controlla formattazione (gofmt) e codice (go vet), salvo -skip-checks;
//  3. incorpora manifest ed eventuale icona con rsrc;
//  4. compila in dist/<arch>/SupportTools.exe con la versione letta da VERSION;
//     I tag passati con -tags (per esempio "demo", che aggiunge i tool di
//     prova) finiscono nel nome della cartella: dist/<arch>-<tag>/;
//  5. copia accanto all'eseguibile LICENSE e THIRD_PARTY_NOTICES.md, che
//     vanno distribuiti con il programma;
//  6. con -run avvia l'eseguibile appena compilato (x64, salvo -arch x86):
//     su Windows direttamente, su Linux e macOS con Wine, in un prefisso
//     dedicato che non tocca quello predefinito dell'utente.
package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// macWine è il percorso di Wine installato da scripts/setup-macos.sh.
const macWine = "/Applications/Wine Stable.app/Contents/Resources/wine/bin/wine"

const (
	modulePath  = "github.com/aptecstudio/supporttools"
	mainPackage = "./cmd/supporttools"
	mainDir     = "cmd/supporttools"
	exeName     = "SupportTools.exe"
	distDir     = "dist"
	// requiredGo è il prefisso della versione di Go ammessa per le build.
	requiredGo = "go1.20."
)

// target è un'architettura Windows di destinazione.
type target struct {
	name   string // nome usato per la cartella in dist (x64, x86)
	goarch string
}

var targets = []target{
	{name: "x64", goarch: "amd64"},
	{name: "x86", goarch: "386"},
}

// releaseFiles vanno distribuiti insieme all'eseguibile (obblighi di licenza).
var releaseFiles = []string{"LICENSE", "THIRD_PARTY_NOTICES.md"}

func main() {
	arch := flag.String("arch", "all", "architettura da compilare: all, x64 o x86")
	tags := flag.String("tags", "", "tag di build aggiuntivi separati da virgola (es. demo)")
	skipChecks := flag.Bool("skip-checks", false, "salta i controlli gofmt e go vet")
	launchExe := flag.Bool("run", false, "avvia l'eseguibile dopo la compilazione (con Wine su Linux e macOS)")
	flag.Parse()

	if err := run(*arch, *tags, *skipChecks, *launchExe); err != nil {
		fmt.Fprintln(os.Stderr, "[x]", err)
		os.Exit(1)
	}
}

func run(arch, tags string, skipChecks, launchExe bool) error {
	selected, err := selectTargets(arch)
	if err != nil {
		return err
	}
	if _, err := os.Stat("go.mod"); err != nil {
		return errors.New("esegui il programma dalla radice del repository (go run ./build)")
	}
	version, err := readVersion()
	if err != nil {
		return err
	}
	if err := checkGoVersion(); err != nil {
		return err
	}
	if !skipChecks {
		if err := checkCode(tags); err != nil {
			return err
		}
	}
	rsrc, err := findTool("rsrc")
	if err != nil {
		return err
	}

	for _, t := range selected {
		info("Compilazione di %s %s (%s)", exeName, version, t.name)
		if err := buildTarget(rsrc, version, tags, t); err != nil {
			return fmt.Errorf("%s: %w", t.name, err)
		}
	}
	info("Completato: eseguibili in %s%c", distDir, filepath.Separator)

	if launchExe {
		return launch(filepath.Join(distDir, outDirName(selected[0], tags), exeName))
	}
	return nil
}

// launch avvia exe. Su Windows passa da "start", che mostra la richiesta di
// elevazione del manifest; altrove usa Wine e attende la chiusura, così i
// messaggi del programma restano visibili nel terminale.
func launch(exe string) error {
	if runtime.GOOS == "windows" {
		info("Avvio di %s", exe)
		abs, err := filepath.Abs(exe)
		if err != nil {
			return err
		}
		return runCmd(nil, "cmd", "/c", "start", "", abs)
	}

	wine, err := findWine()
	if err != nil {
		return err
	}
	prefix, err := winePrefix()
	if err != nil {
		return err
	}
	info("Avvio di %s con Wine (prefisso %s)", exe, prefix)
	if !fileExists(filepath.Join(prefix, "system.reg")) {
		info("Primo avvio: Wine prepara il prefisso, può richiedere qualche minuto")
	}
	env := append(os.Environ(),
		"WINEPREFIX="+prefix,
		// Il programma non usa .NET né il motore HTML: si evita che Wine
		// proponga di installare Mono e Gecko.
		"WINEDLLOVERRIDES=mscoree,mshtml=",
		// Nasconde i messaggi di diagnostica di Wine.
		"WINEDEBUG=-all",
	)
	return runCmd(env, wine, exe)
}

// findWine cerca Wine nel PATH e, su macOS, nella posizione usata dallo
// script di setup.
func findWine() (string, error) {
	if p, err := exec.LookPath("wine"); err == nil {
		return p, nil
	}
	if runtime.GOOS == "darwin" && fileExists(macWine) {
		return macWine, nil
	}
	return "", errors.New("Wine non trovato: esegui lo script di setup del tuo sistema in scripts/")
}

// winePrefix restituisce il prefisso Wine dedicato a Support Tools, nella
// cartella di cache dell'utente (~/.cache su Linux, ~/Library/Caches su macOS).
func winePrefix() (string, error) {
	cache, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	prefix := filepath.Join(cache, "SupportTools", "wine")
	if err := os.MkdirAll(filepath.Dir(prefix), 0o755); err != nil {
		return "", err
	}
	return prefix, nil
}

// outDirName restituisce la cartella in dist per t e i tag indicati.
func outDirName(t target, tags string) string {
	if tags == "" {
		return t.name
	}
	return t.name + "-" + strings.ReplaceAll(tags, ",", "-")
}

func selectTargets(arch string) ([]target, error) {
	if arch == "all" {
		return targets, nil
	}
	for _, t := range targets {
		if t.name == arch {
			return []target{t}, nil
		}
	}
	return nil, fmt.Errorf("architettura sconosciuta %q: usa all, x64 o x86", arch)
}

func readVersion() (string, error) {
	data, err := os.ReadFile("VERSION")
	if err != nil {
		return "", fmt.Errorf("lettura di VERSION: %w", err)
	}
	v := strings.TrimSpace(string(data))
	if v == "" {
		return "", errors.New("il file VERSION è vuoto")
	}
	return v, nil
}

func checkGoVersion() error {
	out, err := output(nil, "go", "env", "GOVERSION")
	if err != nil {
		return err
	}
	if v := strings.TrimSpace(out); !strings.HasPrefix(v, requiredGo) {
		return fmt.Errorf("serve Go %sx per la compatibilità con Windows 7, trovato %s: "+
			"esegui lo script di setup del tuo sistema in scripts/", requiredGo, v)
	}
	return nil
}

// checkCode verifica che il codice sia formattato e superi go vet, per
// entrambe le architetture Windows.
func checkCode(tags string) error {
	info("Controllo della formattazione (gofmt)")
	out, err := output(nil, "gofmt", "-l", ".")
	if err != nil {
		return err
	}
	if files := strings.TrimSpace(out); files != "" {
		return fmt.Errorf("file non formattati (esegui gofmt -w .):\n%s", files)
	}

	for _, t := range targets {
		info("Controllo del codice (go vet, %s)", t.name)
		if err := runCmd(windowsEnv(t), "go", "vet", "-tags", tags, "./..."); err != nil {
			return err
		}
	}
	return nil
}

// findTool cerca uno strumento installato con go install: prima nel PATH,
// poi nella cartella bin del GOPATH.
func findTool(name string) (string, error) {
	if p, err := exec.LookPath(name); err == nil {
		return p, nil
	}
	gopath, err := output(nil, "go", "env", "GOPATH")
	if err != nil {
		return "", err
	}
	exe := name
	if runtime.GOOS == "windows" {
		exe += ".exe"
	}
	// GOPATH può contenere più percorsi: si usa il primo, come go install.
	first := filepath.SplitList(strings.TrimSpace(gopath))[0]
	p := filepath.Join(first, "bin", exe)
	if _, err := os.Stat(p); err != nil {
		return "", fmt.Errorf("%s non trovato: esegui lo script di setup del tuo sistema in scripts/", name)
	}
	return p, nil
}

func buildTarget(rsrc, version, tags string, t target) error {
	// Il suffisso _windows_<goarch> fa sì che Go colleghi il file .syso solo
	// nella build dell'architettura corrispondente.
	syso := filepath.Join(mainDir, "rsrc_windows_"+t.goarch+".syso")
	args := []string{
		"-manifest", filepath.Join(mainDir, "supporttools.manifest"),
		"-arch", t.goarch,
		"-o", syso,
	}
	if icon := filepath.Join(mainDir, "supporttools.ico"); fileExists(icon) {
		args = append(args, "-ico", icon)
	}
	if err := runCmd(nil, rsrc, args...); err != nil {
		return err
	}

	outDir := filepath.Join(distDir, outDirName(t, tags))
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}
	ldflags := strings.Join([]string{
		"-s", "-w",
		// Applicazione grafica: nessuna finestra di console all'avvio.
		"-H", "windowsgui",
		"-X", modulePath + "/internal/version.Version=" + version,
	}, " ")
	err := runCmd(windowsEnv(t), "go", "build",
		"-trimpath",
		"-tags", tags,
		"-ldflags", ldflags,
		"-o", filepath.Join(outDir, exeName),
		mainPackage)
	if err != nil {
		return err
	}

	for _, f := range releaseFiles {
		if err := copyFile(f, filepath.Join(outDir, f)); err != nil {
			return err
		}
	}
	return nil
}

// windowsEnv restituisce l'ambiente per compilare verso Windows senza CGO e
// senza che Go scarichi automaticamente toolchain più recenti.
func windowsEnv(t target) []string {
	return append(os.Environ(),
		"GOOS=windows",
		"GOARCH="+t.goarch,
		"CGO_ENABLED=0",
		"GOTOOLCHAIN=local",
	)
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

func output(env []string, name string, args ...string) (string, error) {
	var stdout bytes.Buffer
	cmd := exec.Command(name, args...)
	cmd.Env = env
	cmd.Stdout = &stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("%s %s: %w", name, strings.Join(args, " "), err)
	}
	return stdout.String(), nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func info(format string, args ...any) {
	fmt.Printf("==> "+format+"\n", args...)
}
